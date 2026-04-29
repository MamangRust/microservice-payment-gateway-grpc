package transfer_test

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
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/transfer"
	pbStats "github.com/MamangRust/microservice-payment-gateway-grpc/pb/transfer/stats"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	transferhandler "github.com/MamangRust/microservice-payment-gateway-grpc/service/apigateway/handler/transfer"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	stats_handler "github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/handler"
	stats_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/handler"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/service"
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

type TransferHandlerTestSuite struct {
	suite.Suite
	ts            *tests.TestSuite
	dbPool      *pgxpool.Pool
	redisClient redis.UniversalClient
	chConn      clickhouse.Conn
	grpcServer  *grpc.Server
	commandClient pb.TransferCommandServiceClient
	queryClient   pb.TransferQueryServiceClient
	conn        *grpc.ClientConn
	router      *echo.Echo
	repos       repository.Repositories
	userRepo    user_repo.UserCommandRepository
	cardRepo    card_repo.Repositories
	saldoRepo   saldo_repo.Repositories

	senderCardNumber   string
	receiverCardNumber string
	transferID         int
}

func (s *TransferHandlerTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	// Run required migrations
	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "saldo", "transfer"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	chOpts, err := clickhouse.ParseDSN(s.ts.CHURL)
	s.Require().NoError(err)
	chConn, err := clickhouse.Open(chOpts)
	s.Require().NoError(err)
	s.chConn = chConn

	// Seed CH Schema
	err = s.chConn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS transfer_events (
			transfer_id UInt64,
			transfer_no String,
			transfer_from String,
			transfer_to String,
			amount Int64,
			status String,
			created_at DateTime DEFAULT now()
		) ENGINE = MergeTree() ORDER BY (transfer_from, created_at);
	`)
	s.Require().NoError(err)

	queries := db.New(pool)
	
	// Repositories for seeding
	s.userRepo = user_repo.NewUserCommandRepository(queries)
	s.cardRepo = *card_repo.NewRepositories(queries, nil)
	s.saldoRepo = saldo_repo.NewRepositories(queries, nil)

	// Transfer repos
	s.repos = repository.NewRepositories(queries, s.saldoRepo, s.cardRepo.CardQuery)

	opts, err := redis.ParseURL(s.ts.RedisURL)
	s.Require().NoError(err)
	s.redisClient = redis.NewClient(opts)

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("test", lp)
	obs, _ := observability.NewObservability("test", log)
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.redisClient, log, cacheMetrics)

	transferService := service.NewService(&service.Deps{
		Kafka:            nil,
		Repositories:     s.repos,
		CardAdapter:      s.ts.CardAdapter,
		SaldoAdapter:     s.ts.SaldoAdapter,
		Logger:           log,
		Cache:            cacheStore,
		AISecurityClient: nil,
	})

	// Seed Sender
	sender, err := s.userRepo.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Sender",
		LastName:  "User",
		Email:     "sender@transfer.com",
		Password:  "password123",
	})
	s.Require().NoError(err)

	sCard, err := s.cardRepo.CardCommand.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID:       int(sender.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(1, 0, 0),
		CVV:          "111",
		CardProvider: "visa",
	})
	s.Require().NoError(err)
	s.senderCardNumber = sCard.CardNumber

	_, err = s.saldoRepo.CreateSaldo(context.Background(), &requests.CreateSaldoRequest{
		CardNumber:   s.senderCardNumber,
		TotalBalance: 1000000,
	})
	s.Require().NoError(err)

	// Seed Receiver
	receiver, err := s.userRepo.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Receiver",
		LastName:  "User",
		Email:     "receiver@transfer.com",
		Password:  "password123",
	})
	s.Require().NoError(err)

	rCard, err := s.cardRepo.CardCommand.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID:       int(receiver.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(1, 0, 0),
		CVV:          "222",
		CardProvider: "mastercard",
	})
	s.Require().NoError(err)
	s.receiverCardNumber = rCard.CardNumber

	_, err = s.saldoRepo.CreateSaldo(context.Background(), &requests.CreateSaldoRequest{
		CardNumber:   s.receiverCardNumber,
		TotalBalance: 0,
	})
	s.Require().NoError(err)

	transferHandlerGapi := handler.NewHandler(transferService)
	
	// Stats Handler
	chRepo := stats_repo.NewRepository(s.chConn)
	transferStatsHandler := stats_handler.NewTransferStatsHandler(chRepo, log)

	server := grpc.NewServer()
	pb.RegisterTransferCommandServiceServer(server, transferHandlerGapi)
	pb.RegisterTransferQueryServiceServer(server, transferHandlerGapi)
	pbStats.RegisterTransferStatsAmountServiceServer(server, transferStatsHandler)
	pbStats.RegisterTransferStatsStatusServiceServer(server, transferStatsHandler)
	s.grpcServer = server

	lis, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)

	go func() {
		_ = server.Serve(lis)
	}()

	// Create gRPC Client
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	s.Require().NoError(err)
	s.conn = conn
	s.commandClient = pb.NewTransferCommandServiceClient(conn)
	s.queryClient = pb.NewTransferQueryServiceClient(conn)

	// Setup Echo
	s.router = echo.New()
	apiErrorHandler := app_errors.NewApiHandler(obs, log)

	transferhandler.RegisterTransferHandler(&transferhandler.DepsTransfer{
		Client:      conn,
		StatsClient: conn, // Using same conn for stats
		E:           s.router,
		Logger:      log,
		Cache:       cacheStore,
		ApiHandler:  apiErrorHandler,
	})
}

func (s *TransferHandlerTestSuite) TearDownSuite() {
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

func (s *TransferHandlerTestSuite) Test1_CreateTransfer() {
	createReq := map[string]interface{}{
		"transfer_from":   s.senderCardNumber,
		"transfer_to":     s.receiverCardNumber,
		"transfer_amount": 100000,
		"transfer_time":   time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(createReq)

	request := httptest.NewRequest(http.MethodPost, "/api/transfer-command/create", bytes.NewBuffer(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, request)
	s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())

	var createRes map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createRes)
	transferData := createRes["data"].(map[string]interface{})
	s.transferID = int(transferData["id"].(float64))

	// Verify balances
	senderSaldo, _ := s.saldoRepo.FindByCardNumber(context.Background(), s.senderCardNumber)
	s.Equal(int32(900000), senderSaldo.TotalBalance)

	receiverSaldo, _ := s.saldoRepo.FindByCardNumber(context.Background(), s.receiverCardNumber)
	s.Equal(int32(100000), receiverSaldo.TotalBalance)
}

func (s *TransferHandlerTestSuite) Test2_FindTransferById() {
	s.Require().NotZero(s.transferID)

	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transfer-query/%d", s.transferID), nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *TransferHandlerTestSuite) Test3_FindAllTransfers() {
	request := httptest.NewRequest(http.MethodGet, "/api/transfer-query?page=1&page_size=10", nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *TransferHandlerTestSuite) Test4_UpdateTransfer() {
	s.Require().NotZero(s.transferID)

	updateReq := map[string]interface{}{
		"transfer_from":   s.senderCardNumber,
		"transfer_to":     s.receiverCardNumber,
		"transfer_amount": 150000, // Increase by 50000
		"transfer_time":   time.Now().Format(time.RFC3339),
	}
	updateBody, _ := json.Marshal(updateReq)
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/transfer-command/update/%d", s.transferID), bytes.NewBuffer(updateBody))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())

	// Verify adjusted balances (Sender 900k - 50k = 850k, Receiver 100k + 50k = 150k)
	senderSaldo, _ := s.saldoRepo.FindByCardNumber(context.Background(), s.senderCardNumber)
	s.Equal(int32(850000), senderSaldo.TotalBalance)

	receiverSaldo, _ := s.saldoRepo.FindByCardNumber(context.Background(), s.receiverCardNumber)
	s.Equal(int32(150000), receiverSaldo.TotalBalance)
}

func (s *TransferHandlerTestSuite) Test5_TrashedTransfer() {
	s.Require().NotZero(s.transferID)

	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/transfer-command/trashed/%d", s.transferID), nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *TransferHandlerTestSuite) Test6_RestoreTransfer() {
	s.Require().NotZero(s.transferID)

	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/transfer-command/restore/%d", s.transferID), nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *TransferHandlerTestSuite) Test7_PermanentDeleteTransfer() {
	s.Require().NotZero(s.transferID)

	request := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/transfer-command/permanent/%d", s.transferID), nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *TransferHandlerTestSuite) Test8_TransferStats_Amount() {
	ctx := context.Background()
	now := time.Now()

	err := s.chConn.Exec(ctx, "TRUNCATE TABLE transfer_events")
	s.Require().NoError(err)

	seedSQL := `INSERT INTO transfer_events (transfer_id, transfer_no, transfer_from, transfer_to, amount, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	// Sender data
	err = s.chConn.Exec(ctx, seedSQL, 1, "TR001", s.senderCardNumber, s.receiverCardNumber, 1000, "success", now)
	s.Require().NoError(err)
	// Receiver data
	err = s.chConn.Exec(ctx, seedSQL, 2, "TR002", "other_sender", s.senderCardNumber, 2000, "success", now)
	s.Require().NoError(err)
	// Failed data
	err = s.chConn.Exec(ctx, seedSQL, 3, "TR003", s.senderCardNumber, "some_receiver", 3000, "failed", now)
	s.Require().NoError(err)

	s.Run("MonthlyAmount", func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transfer/stats/amount/monthly?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code)
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		s.Equal("success", resp["status"])
		s.NotEmpty(resp["data"])
	})

	s.Run("YearlyAmount", func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transfer/stats/amount/yearly?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code)
	})

	s.Run("MonthlyAmountBySender", func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transfer/stats/amount/monthly/sender/%s?year=%d", s.senderCardNumber, now.Year()), nil)
		s.router.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code)
	})

	s.Run("MonthlyAmountByReceiver", func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transfer/stats/amount/monthly/receiver/%s?year=%d", s.senderCardNumber, now.Year()), nil)
		s.router.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code)
	})
}

func (s *TransferHandlerTestSuite) Test9_TransferStats_Status() {
	now := time.Now()

	s.Run("MonthlySuccess", func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transfer/stats/status/monthly/success?year=%d&month=%d", now.Year(), now.Month()), nil)
		s.router.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code)
	})

	s.Run("YearlyFailed", func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transfer/stats/status/yearly/failed?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code)
	})

	s.Run("MonthlySuccessByCard", func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transfer/stats/status/monthly/success/%s?year=%d&month=%d", s.senderCardNumber, now.Year(), now.Month()), nil)
		s.router.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code)
	})

	s.Run("YearlyFailedByCard", func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transfer/stats/status/yearly/failed/%s?year=%d", s.senderCardNumber, now.Year()), nil)
		s.router.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code)
	})
}

func (s *TransferHandlerTestSuite) Test10_BulkOperations() {
	// Restore All
	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/api/transfer-command/restore/all", nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)

	// Delete All Permanent
	rec = httptest.NewRecorder()
	httpReq = httptest.NewRequest(http.MethodPost, "/api/transfer-command/permanent/all", nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)
}

func TestTransferHandlerSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TransferHandlerTestSuite))
}
