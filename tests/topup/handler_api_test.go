package topup_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/topup"
	pbStats "github.com/MamangRust/microservice-payment-gateway-grpc/pb/topup/stats"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	api "github.com/MamangRust/microservice-payment-gateway-grpc/service/apigateway/handler/topup"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	stats_handler "github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/handler"
	stats_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/repository"
	gapi "github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/handler"
	topup_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/service"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	app_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	tests "github.com/MamangRust/microservice-payment-gateway-test"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TopupHandlerTestSuite struct {
	suite.Suite
	ts            *tests.TestSuite
	dbPool        *pgxpool.Pool
	redisClient   redis.UniversalClient
	grpcServer    *grpc.Server
	chConn        clickhouse.Conn
	commandClient pb.TopupCommandServiceClient
	queryClient   pb.TopupQueryServiceClient
	conn          *grpc.ClientConn
	router        *echo.Echo

	userRepo  user_repo.UserCommandRepository
	cardRepo  card_repo.CardCommandRepository
	saldoRepo saldo_repo.Repositories
	topupRepo topup_repo.Repositories

	cardNumber string
	topupID    int
}

func (s *TopupHandlerTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	// Run required migrations
	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "saldo", "topup"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	opts, err := redis.ParseURL(s.ts.RedisURL)
	s.Require().NoError(err)
	s.redisClient = redis.NewClient(opts)

	chOpts, err := clickhouse.ParseDSN(s.ts.CHURL)
	s.Require().NoError(err)
	chConn, err := clickhouse.Open(chOpts)
	s.Require().NoError(err)
	s.chConn = chConn

	// Seed CH Schema
	err = s.chConn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS topup_events (
			topup_id UInt64,
			topup_no String,
			card_number String,
			card_type String,
			card_provider String,
			amount Int64,
			payment_method String,
			status String,
			created_at DateTime DEFAULT now()
		) ENGINE = MergeTree() ORDER BY (card_number, created_at);
	`)
	s.Require().NoError(err)

	queries := db.New(pool)

	userRepos := user_repo.NewRepositories(queries)
	cardRepos := card_repo.NewRepositories(queries, nil)
	saldoRepos := saldo_repo.NewRepositories(queries, nil)

	cardAdapter := &topupCardRepoAdapter{
		CardQueryRepository:   cardRepos.CardQuery,
		CardCommandRepository: cardRepos.CardCommand,
	}
	s.topupRepo = topup_repo.NewRepositories(queries, cardAdapter, saldoRepos)
	s.userRepo = userRepos.UserCommand()
	s.cardRepo = cardRepos.CardCommand
	s.saldoRepo = saldoRepos

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("test", lp)
	obs, _ := observability.NewObservability("test", log)
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.redisClient, log, cacheMetrics)

	topupService := service.NewService(&service.Deps{
		Kafka:        nil,
		Cache:        cacheStore,
		Repositories: s.topupRepo,
		CardAdapter:  s.ts.CardAdapter,
		SaldoAdapter: s.ts.SaldoAdapter,
		Logger:       log,
	})

	// Seed User and Card
	user, err := s.userRepo.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Topup",
		LastName:  "Owner",
		Email:     "topup.handler@example.com",
		Password:  "password123",
	})
	s.Require().NoError(err)

	card, err := s.cardRepo.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID:       int(user.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(1, 0, 0),
		CVV:          "123",
		CardProvider: "visa",
	})
	s.Require().NoError(err)
	s.cardNumber = card.CardNumber

	// Seed Saldo
	_, err = s.saldoRepo.CreateSaldo(context.Background(), &requests.CreateSaldoRequest{
		CardNumber:   s.cardNumber,
		TotalBalance: 1000000,
	})
	s.Require().NoError(err)

	topupGapiHandler := gapi.NewHandler(topupService)

	// Stats Handler
	chRepo := stats_repo.NewRepository(s.chConn)
	topupStatsHandler := stats_handler.NewTopupStatsHandler(chRepo, log)

	server := grpc.NewServer()
	pb.RegisterTopupCommandServiceServer(server, topupGapiHandler)
	pb.RegisterTopupQueryServiceServer(server, topupGapiHandler)
	pbStats.RegisterTopupStatsAmountServiceServer(server, topupStatsHandler)
	pbStats.RegisterTopupStatsMethodServiceServer(server, topupStatsHandler)
	pbStats.RegisterTopupStatsStatusServiceServer(server, topupStatsHandler)
	s.grpcServer = server

	lis, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)
	go func() { _ = server.Serve(lis) }()

	// Create gRPC Client
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	s.Require().NoError(err)
	s.conn = conn
	s.commandClient = pb.NewTopupCommandServiceClient(conn)
	s.queryClient = pb.NewTopupQueryServiceClient(conn)

	// Setup Echo
	s.router = echo.New()
	apiErrorHandler := app_errors.NewApiHandler(obs, log)

	api.RegisterTopupHandler(&api.DepsTopup{
		Client:      s.conn,
		StatsClient: s.conn, // Using same conn for stats
		E:           s.router,
		Logger:      log,
		Cache:       cacheStore,
		ApiHandler:  apiErrorHandler,
	})
}

func (s *TopupHandlerTestSuite) TearDownSuite() {
	if s.conn != nil {
		s.conn.Close()
	}
	if s.grpcServer != nil {
		s.grpcServer.Stop()
	}
	if s.redisClient != nil {
		s.redisClient.Close()
	}
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	if s.chConn != nil {
		s.chConn.Close()
	}
	if s.ts != nil {
		s.ts.Teardown()
	}
}

func (s *TopupHandlerTestSuite) Test1_CreateTopup() {
	req := requests.CreateTopupRequest{
		CardNumber:  s.cardNumber,
		TopupAmount: 100000,
		TopupMethod: "visa",
	}
	body, _ := json.Marshal(req)

	request := httptest.NewRequest(http.MethodPost, "/api/topup-command/create", bytes.NewBuffer(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)

	var createRes map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createRes)
	topupData := createRes["data"].(map[string]interface{})
	s.topupID = int(topupData["id"].(float64))
}

func (s *TopupHandlerTestSuite) Test2_FindById() {
	s.Require().NotZero(s.topupID)
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/topup-query/%d", s.topupID), nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *TopupHandlerTestSuite) Test3_Update() {
	s.Require().NotZero(s.topupID)
	updateReq := requests.UpdateTopupRequest{
		TopupID:     &s.topupID,
		CardNumber:  s.cardNumber,
		TopupAmount: 150000,
		TopupMethod: "mastercard",
	}
	updateBody, _ := json.Marshal(updateReq)
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/topup-command/update/%d", s.topupID), bytes.NewBuffer(updateBody))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *TopupHandlerTestSuite) Test4_DeletePermanent() {
	s.Require().NotZero(s.topupID)
	request := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/topup-command/permanent/%d", s.topupID), nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *TopupHandlerTestSuite) Test7_TopupStats_Amount() {
	ctx := context.Background()
	now := time.Now()

	err := s.chConn.Exec(ctx, "TRUNCATE TABLE topup_events")
	s.Require().NoError(err)

	seedSQL := `INSERT INTO topup_events (topup_id, topup_no, card_number, amount, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	err = s.chConn.Exec(ctx, seedSQL, 1, "TP001", s.cardNumber, 100000, "success", now)
	s.Require().NoError(err)

	s.T().Run("MonthlyAmount", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/topup/stats/amount/monthly?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		s.Equal("success", resp["status"])
		s.NotEmpty(resp["data"])
	})

	s.T().Run("YearlyAmount", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/topup/stats/amount/yearly?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
	})

	s.T().Run("MonthlyAmountByCard", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/topup/stats/amount/monthly/%s?year=%d", s.cardNumber, now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
	})
}

func (s *TopupHandlerTestSuite) Test8_TopupStats_Method() {
	ctx := context.Background()
	now := time.Now()

	err := s.chConn.Exec(ctx, "TRUNCATE TABLE topup_events")
	s.Require().NoError(err)

	seedSQL := `INSERT INTO topup_events (topup_id, topup_no, card_number, amount, payment_method, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	err = s.chConn.Exec(ctx, seedSQL, 1, "TP001", s.cardNumber, 100000, "bri", "success", now)
	s.Require().NoError(err)

	s.T().Run("MonthlyMethod", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/topup/stats/method/monthly?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		s.Equal("success", resp["status"])
		s.NotEmpty(resp["data"])
	})

	s.T().Run("YearlyMethod", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/topup/stats/method/yearly?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
	})
}

func (s *TopupHandlerTestSuite) Test9_TopupStats_Status() {
	ctx := context.Background()
	now := time.Now()

	err := s.chConn.Exec(ctx, "TRUNCATE TABLE topup_events")
	s.Require().NoError(err)

	seedSQL := `INSERT INTO topup_events (topup_id, topup_no, card_number, amount, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	err = s.chConn.Exec(ctx, seedSQL, 1, "TP001", s.cardNumber, 100000, "success", now)
	s.Require().NoError(err)
	err = s.chConn.Exec(ctx, seedSQL, 2, "TP002", s.cardNumber, 50000, "failed", now)
	s.Require().NoError(err)

	s.T().Run("MonthlySuccess", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/topup/stats/status/monthly/success?year=%d&month=%d", now.Year(), now.Month()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		s.Equal("success", resp["status"])
		s.NotEmpty(resp["data"])
	})

	s.T().Run("YearlyFailed", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/topup/stats/status/yearly/failed?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
	})
}

func (s *TopupHandlerTestSuite) Test10_BulkOperations() {
	// Restore All
	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/api/topup-command/restore/all", nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)

	// Delete All Permanent
	rec = httptest.NewRecorder()
	httpReq = httptest.NewRequest(http.MethodPost, "/api/topup-command/permanent/all", nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)
}

func TestTopupHandlerSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TopupHandlerTestSuite))
}
