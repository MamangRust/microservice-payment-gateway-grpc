package saldo_test

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
	api "github.com/MamangRust/microservice-payment-gateway-grpc/service/apigateway/handler/saldo"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/saldo"
	pbStats "github.com/MamangRust/microservice-payment-gateway-grpc/pb/saldo/stats"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	gapi "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/handler"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/service"
	stats_handler "github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/handler"
	stats_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/repository"
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

type SaldoHandlerTestSuite struct {
	suite.Suite
	ts          *tests.TestSuite
	dbPool      *pgxpool.Pool
	redisClient redis.UniversalClient
	grpcServer  *grpc.Server
	chConn      clickhouse.Conn
	commandClient pb.SaldoCommandServiceClient
	queryClient   pb.SaldoQueryServiceClient
	conn        *grpc.ClientConn
	router      *echo.Echo
	
	userRepo  user_repo.UserCommandRepository
	cardRepo  card_repo.CardCommandRepository
	saldoRepo saldo_repo.Repositories

	cardNumber string
	saldoID    int
}

func (s *SaldoHandlerTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	// Run required migrations
	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "saldo"))

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
		CREATE TABLE IF NOT EXISTS saldo_events (
			saldo_id UInt64,
			card_number String,
			total_balance Int64,
			status String,
			created_at DateTime DEFAULT now()
		) ENGINE = MergeTree() ORDER BY (card_number, created_at);
	`)
	s.Require().NoError(err)

	queries := db.New(pool)
	saldoRepos := saldo_repo.NewRepositories(queries, nil)
	userRepos := user_repo.NewRepositories(queries)
	cardRepos := card_repo.NewRepositories(queries, nil)

	s.userRepo = userRepos.UserCommand()
	s.cardRepo = cardRepos.CardCommand
	s.saldoRepo = saldoRepos

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("test", lp)
	obs, _ := observability.NewObservability("test", log)
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.redisClient, log, cacheMetrics)

	saldoService := service.NewService(&service.Deps{
		Repositories: s.saldoRepo,
		CardAdapter:  s.ts.CardAdapter,
		Logger:       log,
		Cache:        cacheStore,
	})

	// Seed User and Card
	user, err := s.userRepo.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Saldo",
		LastName:  "Owner",
		Email:     "saldo.handler@example.com",
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

	saldoGapiHandler := gapi.NewHandler(saldoService)
	
	// Stats Handler
	chRepo := stats_repo.NewRepository(s.chConn)
	saldoStatsHandler := stats_handler.NewSaldoStatsHandler(chRepo, log)

	server := grpc.NewServer()
	pb.RegisterSaldoCommandServiceServer(server, saldoGapiHandler)
	pb.RegisterSaldoQueryServiceServer(server, saldoGapiHandler)
	pbStats.RegisterSaldoStatsBalanceServiceServer(server, saldoStatsHandler)
	pbStats.RegisterSaldoStatsTotalBalanceServer(server, saldoStatsHandler)
	s.grpcServer = server

	lis, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)
	go func() { _ = server.Serve(lis) }()

	// Create gRPC Client
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	s.Require().NoError(err)
	s.conn = conn
	s.commandClient = pb.NewSaldoCommandServiceClient(conn)
	s.queryClient = pb.NewSaldoQueryServiceClient(conn)

	// Setup Echo
	s.router = echo.New()
	apiErrorHandler := app_errors.NewApiHandler(obs, log)

	api.RegisterSaldoHandler(&api.DepsSaldo{
		Client:      s.conn,
		StatsClient: s.conn, // Using same conn for stats
		E:           s.router,
		Logger:      log,
		Cache:       cacheStore,
		ApiHandler:  apiErrorHandler,
	})
}

func (s *SaldoHandlerTestSuite) TearDownSuite() {
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

func (s *SaldoHandlerTestSuite) Test1_CreateSaldo() {
	req := requests.CreateSaldoRequest{
		CardNumber:   s.cardNumber,
		TotalBalance: 100000,
	}
	body, _ := json.Marshal(req)

	request := httptest.NewRequest(http.MethodPost, "/api/saldo-command/create", bytes.NewBuffer(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)

	var createRes map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createRes)
	saldoData := createRes["data"].(map[string]interface{})
	s.saldoID = int(saldoData["id"].(float64))
}

func (s *SaldoHandlerTestSuite) Test2_FindById() {
	s.Require().NotZero(s.saldoID)
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/saldo-query/%d", s.saldoID), nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *SaldoHandlerTestSuite) Test3_FindByCardNumber() {
	s.Require().NotEmpty(s.cardNumber)
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/saldo-query/card_number/%s", s.cardNumber), nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *SaldoHandlerTestSuite) Test4_Update() {
	s.Require().NotZero(s.saldoID)
	updateReq := requests.UpdateSaldoRequest{
		CardNumber:   s.cardNumber,
		TotalBalance: 150000,
	}
	updateBody, _ := json.Marshal(updateReq)
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/saldo-command/update/%d", s.saldoID), bytes.NewBuffer(updateBody))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *SaldoHandlerTestSuite) Test5_DeletePermanent() {
	s.Require().NotZero(s.saldoID)
	request := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/saldo-command/permanent/%d", s.saldoID), nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *SaldoHandlerTestSuite) Test7_SaldoStats() {
	ctx := context.Background()
	now := time.Now()

	err := s.chConn.Exec(ctx, "TRUNCATE TABLE saldo_events")
	s.Require().NoError(err)

	seedSQL := `INSERT INTO saldo_events (saldo_id, card_number, total_balance, status, created_at) VALUES (?, ?, ?, ?, ?)`
	err = s.chConn.Exec(ctx, seedSQL, 1, s.cardNumber, 1000000, "success", now)
	s.Require().NoError(err)

	s.T().Run("MonthlyBalance", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/saldo/stats/balance/monthly?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		s.Equal("success", resp["status"])
		s.NotEmpty(resp["data"])
	})

	s.T().Run("YearlyBalance", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/saldo/stats/balance/yearly?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
	})

	s.T().Run("MonthlyTotal", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/saldo/stats/total/monthly?year=%d&month=%d", now.Year(), now.Month()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
	})

	s.T().Run("YearlyTotal", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/saldo/stats/total/yearly?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
	})
}

func (s *SaldoHandlerTestSuite) Test8_BulkOperations() {
	// Restore All
	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/api/saldo-command/restore/all", nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)

	// Delete All Permanent
	rec = httptest.NewRecorder()
	httpReq = httptest.NewRequest(http.MethodPost, "/api/saldo-command/permanent/all", nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)
}

func TestSaldoHandlerSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(SaldoHandlerTestSuite))
}
