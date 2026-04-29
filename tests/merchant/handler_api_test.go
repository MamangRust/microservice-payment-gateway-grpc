package merchant_test

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
	api "github.com/MamangRust/microservice-payment-gateway-grpc/service/apigateway/handler/merchant"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/merchant"
	pbStats "github.com/MamangRust/microservice-payment-gateway-grpc/pb/merchant/stats"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	app_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	tests "github.com/MamangRust/microservice-payment-gateway-test"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/handler"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/service"
	stats_handler "github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/handler"
	stats_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/repository"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MerchantHandlerTestSuite struct {
	suite.Suite
	ts          *tests.TestSuite
	dbPool      *pgxpool.Pool
	redisClient redis.UniversalClient
	grpcServer  *grpc.Server
	chConn            clickhouse.Conn
	conn              *grpc.ClientConn
	statsClient       pbStats.MerchantStatsAmountServiceClient
	methodClient      pbStats.MerchantStatsMethodServiceClient
	totalAmountClient pbStats.MerchantStatsTotalAmountServiceClient
	transactionClient pb.MerchantTransactionServiceClient
	router            *echo.Echo
	userRepo    user_repo.UserCommandRepository
	userID      int
	merchantID  int
}

func (s *MerchantHandlerTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	// Run required migrations
	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "merchant", "transaction"))

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
		CREATE TABLE IF NOT EXISTS transaction_events (
			transaction_id UInt64,
			transaction_no String,
			merchant_id UInt64,
			merchant_name String,
			apikey String,
			apikey_name String,
			amount Int64,
			payment_method String,
			status String,
			created_at DateTime DEFAULT now()
		) ENGINE = MergeTree() ORDER BY (merchant_id, created_at);
	`)
	s.Require().NoError(err)

	queries := db.New(pool)
	repos := repository.NewRepositories(queries, nil)
	s.userRepo = user_repo.NewUserCommandRepository(queries)

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("test", lp)
	obs, _ := observability.NewObservability("test", log)
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.redisClient, log, cacheMetrics)

	merchantService := service.NewService(&service.Deps{
		Kafka:        nil,
		Repositories: repos,
		UserAdapter:  s.ts.UserAdapter,
		Logger:       log,
		Cache:        cacheStore,
	})

	// Seed User
	user, err := s.userRepo.CreateUser(s.ts.Ctx, &requests.CreateUserRequest{
		FirstName: "Handler",
		LastName:  "Merchant",
		Email:     "handler.merchant@example.com",
		Password:  "password123",
	})
	s.Require().NoError(err)
	s.userID = int(user.UserID)

	merchantHandler := handler.NewHandler(merchantService)
	
	// Stats Handler
	chRepo := stats_repo.NewRepository(s.chConn)
	merchantStatsHandler := stats_handler.NewMerchantStatsHandler(chRepo, log)

	server := grpc.NewServer()
	pb.RegisterMerchantCommandServiceServer(server, merchantHandler)
	pb.RegisterMerchantQueryServiceServer(server, merchantHandler)
	pbStats.RegisterMerchantStatsAmountServiceServer(server, merchantStatsHandler)
	pbStats.RegisterMerchantStatsMethodServiceServer(server, merchantStatsHandler)
	pbStats.RegisterMerchantStatsTotalAmountServiceServer(server, merchantStatsHandler)
	pb.RegisterMerchantTransactionServiceServer(server, merchantStatsHandler)
	s.grpcServer = server

	lis, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)

	go func() {
		_ = server.Serve(lis)
	}()

	// Create gRPC Client for Echo
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	s.Require().NoError(err)
	s.conn = conn

	// Setup Echo
	s.router = echo.New()
	apiErrorHandler := app_errors.NewApiHandler(obs, log)

	api.RegisterMerchantHandler(&api.DepsMerchant{
		Client:      conn,
		StatsClient: conn, // Using same conn for stats
		E:           s.router,
		Logger:      log,
		Cache:       cacheStore,
		ApiHandler:  apiErrorHandler,
	})
}

func (s *MerchantHandlerTestSuite) TearDownSuite() {
	s.conn.Close()
	s.grpcServer.Stop()
	s.redisClient.Close()
	s.dbPool.Close()
	if s.chConn != nil {
		s.chConn.Close()
	}
	s.ts.Teardown()
}

func (s *MerchantHandlerTestSuite) Test1_CreateMerchant() {
	req := requests.CreateMerchantRequest{
		Name:   "Handler Merchant",
		UserID: s.userID,
	}
	body, _ := json.Marshal(req)

	request := httptest.NewRequest(http.MethodPost, "/api/merchant-command/create", bytes.NewBuffer(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	s.router.ServeHTTP(rec, request)
	s.Require().Equal(http.StatusOK, rec.Code, rec.Body.String())

	var createRes map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &createRes)
	merchantData := createRes["data"].(map[string]interface{})
	s.merchantID = int(merchantData["id"].(float64))
}

func (s *MerchantHandlerTestSuite) Test2_FindMerchantById() {
	s.Require().NotZero(s.merchantID)

	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/merchant-query/%d", s.merchantID), nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, request)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *MerchantHandlerTestSuite) Test3_MerchantStats_MonthlyAmount() {
	ctx := context.Background()
	now := time.Now()

	err := s.chConn.Exec(ctx, "TRUNCATE TABLE transaction_events")
	s.Require().NoError(err)

	seedSQL := `INSERT INTO transaction_events (transaction_id, transaction_no, merchant_id, amount, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	err = s.chConn.Exec(ctx, seedSQL, 201, "TXM001", int32(s.merchantID), 1500, "success", now)
	s.Require().NoError(err)

	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/merchant/stats/amount/monthly?year=%d", now.Year()), nil)
	
	s.router.ServeHTTP(rec, httpReq)

	s.Equal(http.StatusOK, rec.Code)
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	s.Equal("success", resp["status"])
	s.NotEmpty(resp["data"])
}

func (s *MerchantHandlerTestSuite) Test4_MerchantStats_MonthlyMethod() {
	now := time.Now()

	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/merchant/stats/method/monthly?year=%d", now.Year()), nil)
	
	s.router.ServeHTTP(rec, httpReq)

	s.Equal(http.StatusOK, rec.Code)
}

func (s *MerchantHandlerTestSuite) Test6_MerchantStats_ByMerchant_Full() {
	now := time.Now()
	merchantId := s.merchantID

	// Amount By Merchant
	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/merchant/stats/amount/monthly/id/%d?year=%d", merchantId, now.Year()), nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)

	// Total Amount By Merchant
	rec = httptest.NewRecorder()
	httpReq = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/merchant/stats/total-amount/monthly/id/%d?year=%d", merchantId, now.Year()), nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)

	// Method By Merchant
	rec = httptest.NewRecorder()
	httpReq = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/merchant/stats/method/monthly/id/%d?year=%d", merchantId, now.Year()), nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *MerchantHandlerTestSuite) Test7_MerchantStats_ByApikey_Full() {
	now := time.Now()
	apiKey := "api-key-test"

	// Amount By Apikey
	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/merchant/stats/amount/monthly/apikey?apikey=%s&year=%d", apiKey, now.Year()), nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)

	// Total Amount By Apikey
	rec = httptest.NewRecorder()
	httpReq = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/merchant/stats/total-amount/monthly/apikey?apikey=%s&year=%d", apiKey, now.Year()), nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)

	// Method By Apikey
	rec = httptest.NewRecorder()
	httpReq = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/merchant/stats/method/monthly/apikey?apikey=%s&year=%d", apiKey, now.Year()), nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *MerchantHandlerTestSuite) Test8_MerchantStats_Yearly_Full() {
	now := time.Now()

	// Amount Yearly
	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/merchant/stats/amount/yearly?year=%d", now.Year()), nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)

	// Total Amount Yearly
	rec = httptest.NewRecorder()
	httpReq = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/merchant/stats/total-amount/yearly?year=%d", now.Year()), nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)

	// Method Yearly
	rec = httptest.NewRecorder()
	httpReq = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/merchant/stats/method/yearly?year=%d", now.Year()), nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *MerchantHandlerTestSuite) Test9_MerchantTransaction_Full() {
	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodGet, "/api/merchant/stats/transactions", nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	httpReq = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/merchant/stats/transactions/id/%d", s.merchantID), nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	httpReq = httptest.NewRequest(http.MethodGet, "/api/merchant/stats/transactions/apikey?apikey=api-key-test", nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)
}

func (s *MerchantHandlerTestSuite) Test10_BulkOperations() {
	// Restore All
	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/api/merchant-command/restore/all", nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)

	// Delete All Permanent
	rec = httptest.NewRecorder()
	httpReq = httptest.NewRequest(http.MethodPost, "/api/merchant-command/permanent/all", nil)
	s.router.ServeHTTP(rec, httpReq)
	s.Equal(http.StatusOK, rec.Code)
}

func TestMerchantHandlerSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(MerchantHandlerTestSuite))
}
