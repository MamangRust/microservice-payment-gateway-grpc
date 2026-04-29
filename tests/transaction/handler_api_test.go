package transaction_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	pbAISecurity "github.com/MamangRust/microservice-payment-gateway-grpc/pb/ai_security"
	pb_merchant "github.com/MamangRust/microservice-payment-gateway-grpc/pb/merchant"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/transaction"
	pbStats "github.com/MamangRust/microservice-payment-gateway-grpc/pb/transaction/stats"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	api_transaction "github.com/MamangRust/microservice-payment-gateway-grpc/service/apigateway/handler/transaction"
	mencache "github.com/MamangRust/microservice-payment-gateway-grpc/service/apigateway/redis"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	merchant_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	stats_handler "github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/handler"
	stats_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/handler"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/service"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
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

type TransactionHandlerTestSuite struct {
	suite.Suite
	ts             *tests.TestSuite
	dbPool         *pgxpool.Pool
	redisClient    redis.UniversalClient
	grpcServer     *grpc.Server
	chConn         clickhouse.Conn
	commandClient  pb.TransactionCommandServiceClient
	queryClient    pb.TransactionQueryServiceClient
	merchantClient pb_merchant.MerchantCommandServiceClient
	conn           *grpc.ClientConn
	router         *echo.Echo

	// Repositories for seeding
	userRepo     user_repo.UserCommandRepository
	cardRepo     card_repo.Repositories
	saldoRepo    saldo_repo.Repositories
	merchantRepo merchant_repo.Repositories

	customerCardNumber string
	merchantApiKey     string
	merchantID         int
	merchantCardNumber string
	transactionID      int
}

func (s *TransactionHandlerTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	// Run required migrations
	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "merchant", "saldo", "transaction"))

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
		CREATE TABLE IF NOT EXISTS transaction_events (
			transaction_id UInt64,
			transaction_no String,
			merchant_id UInt64,
			merchant_name String,
			card_number String,
			amount Int64,
			payment_method String,
			status String,
			created_at DateTime DEFAULT now()
		) ENGINE = MergeTree() ORDER BY (merchant_id, created_at);
	`)
	s.Require().NoError(err)

	queries := db.New(pool)

	// Repositories for seeding
	s.userRepo = user_repo.NewUserCommandRepository(queries)
	s.cardRepo = *card_repo.NewRepositories(queries, nil)
	s.saldoRepo = saldo_repo.NewRepositories(queries, nil)
	s.merchantRepo = merchant_repo.NewRepositories(queries, nil)

	opts, err := redis.ParseURL(s.ts.RedisURL)
	s.Require().NoError(err)
	s.redisClient = redis.NewClient(opts)

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("test", lp)
	obs, _ := observability.NewObservability("test", log)
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.redisClient, log, cacheMetrics)

	// Transaction module expects specific interfaces. We use the real ones but may need wrappers.
	cardRepoWrapper := &transactionCardRepo{
		query:   s.cardRepo.CardQuery,
		command: s.cardRepo.CardCommand,
	}

	transactionRepos := repository.NewRepositories(queries, s.saldoRepo, cardRepoWrapper, s.merchantRepo)
	transactionService := service.NewService(&service.Deps{
		Kafka:            nil,
		Repositories:     transactionRepos,
		MerchantAdapter:  s.ts.MerchantAdapter,
		CardAdapter:      s.ts.CardAdapter,
		SaldoAdapter:     s.ts.SaldoAdapter,
		Logger:           log,
		Cache:            cacheStore,
		AISecurityClient: nil,
	})

	// Seed Customer
	customer, err := s.userRepo.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Transaction",
		LastName:  "Customer",
		Email:     "customer@transaction.com",
		Password:  "password123",
	})
	s.Require().NoError(err)

	cCard, err := s.cardRepo.CardCommand.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID:       int(customer.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(1, 0, 0),
		CVV:          "123",
		CardProvider: "visa",
	})
	s.Require().NoError(err)
	s.customerCardNumber = cCard.CardNumber

	_, err = s.saldoRepo.CreateSaldo(context.Background(), &requests.CreateSaldoRequest{
		CardNumber:   s.customerCardNumber,
		TotalBalance: 1000000,
	})
	s.Require().NoError(err)

	// Seed Merchant
	owner, err := s.userRepo.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Merchant",
		LastName:  "Owner",
		Email:     "merchant.owner@transaction.com",
		Password:  "password123",
	})
	s.Require().NoError(err)

	merchant, err := s.merchantRepo.CreateMerchant(context.Background(), &requests.CreateMerchantRequest{
		UserID: int(owner.UserID),
		Name:   "Transaction Merchant",
	})
	s.Require().NoError(err)
	s.merchantID = int(merchant.MerchantID)

	_, err = s.merchantRepo.UpdateMerchantStatus(context.Background(), &requests.UpdateMerchantStatusRequest{
		MerchantID: &s.merchantID,
		Status:     "active",
	})
	s.Require().NoError(err)

	mFull, _ := s.merchantRepo.FindByMerchantId(context.Background(), s.merchantID)
	s.merchantApiKey = mFull.ApiKey

	mCard, err := s.cardRepo.CardCommand.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID:       int(owner.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(1, 0, 0),
		CVV:          "321",
		CardProvider: "mastercard",
	})
	s.Require().NoError(err)
	s.merchantCardNumber = mCard.CardNumber

	_, err = s.saldoRepo.CreateSaldo(context.Background(), &requests.CreateSaldoRequest{
		CardNumber:   s.merchantCardNumber,
		TotalBalance: 0,
	})
	s.Require().NoError(err)

	transactionHandlerGapi := handler.NewHandler(transactionService)

	// Stats Handler
	chRepo := stats_repo.NewRepository(s.chConn)
	transactionStatsHandler := stats_handler.NewTransactionStatsHandler(chRepo, log)

	server := grpc.NewServer()
	pb.RegisterTransactionCommandServiceServer(server, transactionHandlerGapi)
	pb.RegisterTransactionQueryServiceServer(server, transactionHandlerGapi)
	pbStats.RegisterTransactionStatsAmountServiceServer(server, transactionStatsHandler)
	pbStats.RegisterTransactionStatsMethodServiceServer(server, transactionStatsHandler)
	pbStats.RegisterTransactionStatsStatusServiceServer(server, transactionStatsHandler)
	pbAISecurity.RegisterAISecurityServiceServer(server, &mockAISecurityServer{})
	s.grpcServer = server

	lis, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)
	go func() { _ = server.Serve(lis) }()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	s.Require().NoError(err)
	s.conn = conn
	s.commandClient = pb.NewTransactionCommandServiceClient(conn)
	s.queryClient = pb.NewTransactionQueryServiceClient(conn)
	s.merchantClient = pb_merchant.NewMerchantCommandServiceClient(conn)

	// Setup Echo
	s.router = echo.New()
	apiErrorHandler := errors.NewApiHandler(obs, log)

	// Seed Merchant Cache to bypass Kafka validation
	merchantCache := mencache.NewMerchantCache(cacheStore)
	merchantCache.SetMerchantCache(context.Background(), strconv.Itoa(s.merchantID), s.merchantApiKey)

	// Use Refactored Handlers
	api_transaction.RegisterTransactionHandler(&api_transaction.DepsTransaction{
		Client:          conn,
		StatsClient:     conn, // Using same conn for stats
		E:               s.router,
		Kafka:           nil,
		Logger:          log,
		Cache:           cacheStore,
		ApiHandler:      apiErrorHandler,
		CacheApiGateway: mencache.NewCacheApiGateway(cacheStore),
		AISecurity:      conn,
	})
}

func (s *TransactionHandlerTestSuite) TearDownSuite() {
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

func (s *TransactionHandlerTestSuite) Test1_CreateTransaction() {
	createReq := map[string]interface{}{
		"card_number":      s.customerCardNumber,
		"amount":           50000,
		"payment_method":   "visa",
		"merchant_id":      s.merchantID,
		"transaction_time": time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(createReq)

	request := httptest.NewRequest(http.MethodPost, "/api/transaction-command/create", bytes.NewBuffer(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	request.Header.Set("X-API-Key", s.merchantApiKey)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, request)
	s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())

	var res map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &res)
	data := res["data"].(map[string]interface{})
	s.transactionID = int(data["id"].(float64))

	// Verify balances
	customerSaldo, _ := s.saldoRepo.FindByCardNumber(context.Background(), s.customerCardNumber)
	s.Equal(int32(950000), customerSaldo.TotalBalance)

	merchantSaldo, _ := s.saldoRepo.FindByCardNumber(context.Background(), s.merchantCardNumber)
	s.Equal(int32(50000), merchantSaldo.TotalBalance)
}

func (s *TransactionHandlerTestSuite) Test2_FindTransactionById() {
	s.Require().NotZero(s.transactionID)
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transaction-query/%d", s.transactionID), nil)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, request)
	s.Require().Equal(http.StatusOK, rec.Code)
}

func (s *TransactionHandlerTestSuite) Test3_FindAllTransactions() {
	request := httptest.NewRequest(http.MethodGet, "/api/transaction-query?page=1&page_size=10", nil)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, request)
	s.Require().Equal(http.StatusOK, rec.Code)
}

func (s *TransactionHandlerTestSuite) Test4_UpdateTransaction() {
	s.Require().NotZero(s.transactionID)
	updateReq := map[string]interface{}{
		"card_number":      s.customerCardNumber,
		"amount":           60000, // Increase by 10000
		"payment_method":   "visa",
		"merchant_id":      s.merchantID,
		"transaction_time": time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(updateReq)

	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/transaction-command/update/%d", s.transactionID), bytes.NewBuffer(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	request.Header.Set("X-API-Key", s.merchantApiKey)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, request)
	s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())

	// Verify adjusted balance (950k - 10k = 940k)
	customerSaldo, _ := s.saldoRepo.FindByCardNumber(context.Background(), s.customerCardNumber)
	s.Equal(int32(940000), customerSaldo.TotalBalance)
}

func (s *TransactionHandlerTestSuite) Test5_TrashedTransaction() {
	s.Require().NotZero(s.transactionID)
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/transaction-command/trashed/%d", s.transactionID), nil)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, request)
	s.Require().Equal(http.StatusOK, rec.Code)
}

func (s *TransactionHandlerTestSuite) Test6_RestoreTransaction() {
	s.Require().NotZero(s.transactionID)
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/transaction-command/restore/%d", s.transactionID), nil)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, request)
	s.Require().Equal(http.StatusOK, rec.Code)
}

func (s *TransactionHandlerTestSuite) Test7_PermanentDeleteTransaction() {
	s.Require().NotZero(s.transactionID)
	request := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/transaction-command/permanent/%d", s.transactionID), nil)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, request)
	s.Require().Equal(http.StatusOK, rec.Code)
}

func (s *TransactionHandlerTestSuite) Test8_TransactionStats_Amount() {
	ctx := context.Background()
	now := time.Now()

	err := s.chConn.Exec(ctx, "TRUNCATE TABLE transaction_events")
	s.Require().NoError(err)

	seedSQL := `INSERT INTO transaction_events (transaction_id, transaction_no, merchant_id, card_number, amount, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	err = s.chConn.Exec(ctx, seedSQL, 1, "TXN001", int32(s.merchantID), s.customerCardNumber, 100000, "success", now)
	s.Require().NoError(err)

	s.T().Run("MonthlyAmount", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transaction/stats/amount/monthly?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		s.Equal("success", resp["status"])
		s.NotEmpty(resp["data"])
	})

	s.T().Run("YearlyAmount", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transaction/stats/amount/yearly?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
	})

	s.T().Run("MonthlyAmountByCard", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transaction/stats/amount/monthly/%s?year=%d", s.customerCardNumber, now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
	})
}

func (s *TransactionHandlerTestSuite) Test9_TransactionStats_Method() {
	ctx := context.Background()
	now := time.Now()

	err := s.chConn.Exec(ctx, "TRUNCATE TABLE transaction_events")
	s.Require().NoError(err)

	seedSQL := `INSERT INTO transaction_events (transaction_id, transaction_no, merchant_id, card_number, amount, payment_method, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	err = s.chConn.Exec(ctx, seedSQL, 1, "TXN001", int32(s.merchantID), s.customerCardNumber, 100000, "visa", "success", now)
	s.Require().NoError(err)

	s.T().Run("MonthlyMethod", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transaction/stats/method/monthly?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		s.Equal("success", resp["status"])
		s.NotEmpty(resp["data"])
	})

	s.T().Run("YearlyMethod", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transaction/stats/method/yearly?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
	})
}

func (s *TransactionHandlerTestSuite) Test10_TransactionStats_Status() {
	ctx := context.Background()
	now := time.Now()

	err := s.chConn.Exec(ctx, "TRUNCATE TABLE transaction_events")
	s.Require().NoError(err)

	seedSQL := `INSERT INTO transaction_events (transaction_id, transaction_no, merchant_id, card_number, amount, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	err = s.chConn.Exec(ctx, seedSQL, 1, "TXN001", int32(s.merchantID), s.customerCardNumber, 100000, "success", now)
	s.Require().NoError(err)
	err = s.chConn.Exec(ctx, seedSQL, 2, "TXN002", int32(s.merchantID), s.customerCardNumber, 50000, "failed", now)
	s.Require().NoError(err)

	s.T().Run("MonthlySuccess", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transaction/stats/status/monthly/success?year=%d&month=%d", now.Year(), now.Month()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)
		s.Equal("success", resp["status"])
		s.NotEmpty(resp["data"])
	})

	s.T().Run("YearlyFailed", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/transaction/stats/status/yearly/failed?year=%d", now.Year()), nil)
		s.router.ServeHTTP(rec, httpReq)
		s.Equal(http.StatusOK, rec.Code)
	})
}

func (s *TransactionHandlerTestSuite) Test11_BulkOperations() {
	// Restore All
	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/api/transaction-command/restore/all", nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)

	// Delete All Permanent
	rec = httptest.NewRecorder()
	httpReq = httptest.NewRequest(http.MethodPost, "/api/transaction-command/permanent/all", nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)
}

func TestTransactionHandlerSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TransactionHandlerTestSuite))
}
