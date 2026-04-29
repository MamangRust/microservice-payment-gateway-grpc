package merchant_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/merchant"
	pbStats "github.com/MamangRust/microservice-payment-gateway-grpc/pb/merchant/stats"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	tests "github.com/MamangRust/microservice-payment-gateway-test"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/handler"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/service"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	stats_handler "github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/handler"
	stats_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/repository"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type MerchantGapiTestSuite struct {
	suite.Suite
	ts              *tests.TestSuite
	dbPool          *pgxpool.Pool
	redisClient     redis.UniversalClient
	grpcServer      *grpc.Server
	chConn            clickhouse.Conn
	commandClient     pb.MerchantCommandServiceClient
	queryClient       pb.MerchantQueryServiceClient
	statsClient       pbStats.MerchantStatsAmountServiceClient
	methodClient      pbStats.MerchantStatsMethodServiceClient
	totalAmountClient pbStats.MerchantStatsTotalAmountServiceClient
	transactionClient pb.MerchantTransactionServiceClient
	conn              *grpc.ClientConn
	userRepo          user_repo.UserCommandRepository
	userID          int
	merchantID      int
}

func (s *MerchantGapiTestSuite) SetupSuite() {
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
	user, err := s.userRepo.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Gapi",
		LastName:  "Merchant",
		Email:     "gapi.merchant@example.com",
		Password:  "password123",
	})
	s.Require().NoError(err)
	s.userID = int(user.UserID)

	// Start gRPC Server
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

	// Create Client
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	s.Require().NoError(err)
	s.conn = conn
	s.commandClient = pb.NewMerchantCommandServiceClient(conn)
	s.queryClient = pb.NewMerchantQueryServiceClient(conn)
	s.statsClient = pbStats.NewMerchantStatsAmountServiceClient(conn)
	s.methodClient = pbStats.NewMerchantStatsMethodServiceClient(conn)
	s.totalAmountClient = pbStats.NewMerchantStatsTotalAmountServiceClient(conn)
	s.transactionClient = pb.NewMerchantTransactionServiceClient(conn)
}

func (s *MerchantGapiTestSuite) TearDownSuite() {
	s.conn.Close()
	s.grpcServer.Stop()
	s.redisClient.Close()
	s.dbPool.Close()
	if s.chConn != nil {
		s.chConn.Close()
	}
	s.ts.Teardown()
}

func (s *MerchantGapiTestSuite) Test1_CreateMerchant() {
	ctx := context.Background()

	createReq := &pb.CreateMerchantRequest{
		Name:   "Gapi Merchant",
		UserId: int32(s.userID),
	}
	res, err := s.commandClient.CreateMerchant(ctx, createReq)
	s.NoError(err)
	s.Equal(createReq.Name, res.Data.Name)
	s.merchantID = int(res.Data.Id)
}

func (s *MerchantGapiTestSuite) Test2_FindMerchantById() {
	s.Require().NotZero(s.merchantID)
	ctx := context.Background()

	findReq := &pb.FindByIdMerchantRequest{
		MerchantId: int32(s.merchantID),
	}
	found, err := s.queryClient.FindByIdMerchant(ctx, findReq)
	s.NoError(err)
	s.Equal(int32(s.merchantID), found.Data.Id)
}

func (s *MerchantGapiTestSuite) Test3_MerchantStats_MonthlyAmount() {
	ctx := context.Background()
	now := time.Now()

	err := s.chConn.Exec(ctx, "TRUNCATE TABLE transaction_events")
	s.Require().NoError(err)

	seedSQL := `INSERT INTO transaction_events (transaction_id, transaction_no, merchant_id, amount, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	err = s.chConn.Exec(ctx, seedSQL, 201, "TXM001", int32(s.merchantID), 1500, "success", now)
	s.Require().NoError(err)

	req := &pb.FindYearMerchant{Year: int32(now.Year())}
	resp, err := s.statsClient.FindMonthlyAmountMerchant(ctx, req)

	s.NoError(err)
	s.Equal("success", resp.Status)
	s.NotEmpty(resp.Data)
}

func (s *MerchantGapiTestSuite) Test4_MerchantStats_Method_Monthly() {
	ctx := context.Background()
	now := time.Now()

	req := &pb.FindYearMerchant{Year: int32(now.Year())}
	resp, err := s.methodClient.FindMonthlyPaymentMethodsMerchant(ctx, req)

	s.NoError(err)
	s.Equal("success", resp.Status)
}

func (s *MerchantGapiTestSuite) Test5_MerchantStats_TotalAmount_Monthly() {
	ctx := context.Background()
	now := time.Now()

	req := &pb.FindYearMerchant{Year: int32(now.Year())}
	resp, err := s.totalAmountClient.FindMonthlyTotalAmountMerchant(ctx, req)

	s.NoError(err)
	s.Equal("success", resp.Status)
}

func (s *MerchantGapiTestSuite) Test6_MerchantStats_ByMerchant() {
	ctx := context.Background()
	now := time.Now()
	merchantId := int32(s.merchantID)

	// Amount
	reqAmount := &pb.FindYearMerchantById{Year: int32(now.Year()), MerchantId: merchantId}
	respAmount, err := s.statsClient.FindMonthlyAmountByMerchants(ctx, reqAmount)
	s.NoError(err)
	s.Equal("success", respAmount.Status)

	// Method
	reqMethod := &pb.FindYearMerchantById{Year: int32(now.Year()), MerchantId: merchantId}
	respMethod, err := s.methodClient.FindMonthlyPaymentMethodByMerchants(ctx, reqMethod)
	s.NoError(err)
	s.Equal("success", respMethod.Status)

	// Total Amount
	reqTotal := &pb.FindYearMerchantById{Year: int32(now.Year()), MerchantId: merchantId}
	respTotal, err := s.totalAmountClient.FindMonthlyTotalAmountByMerchants(ctx, reqTotal)
	s.NoError(err)
	s.Equal("success", respTotal.Status)
}

func (s *MerchantGapiTestSuite) Test7_MerchantStats_ByApikey() {
	ctx := context.Background()
	now := time.Now()
	apiKey := "test-api-key"

	err := s.chConn.Exec(ctx, `INSERT INTO transaction_events (transaction_id, transaction_no, merchant_id, apikey, amount, payment_method, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		301, "TXM301", int32(s.merchantID), apiKey, 2000, "debit", "success", now)
	s.Require().NoError(err)

	// Amount
	reqAmount := &pb.FindYearMerchantByApikey{Year: int32(now.Year()), ApiKey: apiKey}
	respAmount, err := s.statsClient.FindMonthlyAmountByApikey(ctx, reqAmount)
	s.NoError(err)
	s.Equal("success", respAmount.Status)

	// Method
	reqMethod := &pb.FindYearMerchantByApikey{Year: int32(now.Year()), ApiKey: apiKey}
	respMethod, err := s.methodClient.FindMonthlyPaymentMethodByApikey(ctx, reqMethod)
	s.NoError(err)
	s.Equal("success", respMethod.Status)

	// Total Amount
	reqTotal := &pb.FindYearMerchantByApikey{Year: int32(now.Year()), ApiKey: apiKey}
	respTotal, err := s.totalAmountClient.FindMonthlyTotalAmountByApikey(ctx, reqTotal)
	s.NoError(err)
	s.Equal("success", respTotal.Status)
}

func (s *MerchantGapiTestSuite) Test8_MerchantStats_Yearly() {
	ctx := context.Background()
	now := time.Now()

	// Amount
	reqAmount := &pb.FindYearMerchant{Year: int32(now.Year())}
	respAmount, err := s.statsClient.FindYearlyAmountMerchant(ctx, reqAmount)
	s.NoError(err)
	s.Equal("success", respAmount.Status)

	// Method
	reqMethod := &pb.FindYearMerchant{Year: int32(now.Year())}
	respMethod, err := s.methodClient.FindYearlyPaymentMethodMerchant(ctx, reqMethod)
	s.NoError(err)
	s.Equal("success", respMethod.Status)

	// Total Amount
	reqTotal := &pb.FindYearMerchant{Year: int32(now.Year())}
	respTotal, err := s.totalAmountClient.FindYearlyTotalAmountMerchant(ctx, reqTotal)
	s.NoError(err)
	s.Equal("success", respTotal.Status)
}

func (s *MerchantGapiTestSuite) Test9_MerchantTransaction_Full() {
	ctx := context.Background()

	// All
	respAll, err := s.transactionClient.FindAllTransactionMerchant(ctx, &pb.FindAllMerchantRequest{})
	s.NoError(err)
	s.Equal("success", respAll.Status)

	// By Merchant
	respMerchant, err := s.transactionClient.FindAllTransactionByMerchant(ctx, &pb.FindAllMerchantTransaction{MerchantId: int32(s.merchantID)})
	s.NoError(err)
	s.Equal("success", respMerchant.Status)

	// By Apikey
	respApiKey, err := s.transactionClient.FindAllTransactionByApikey(ctx, &pb.FindAllMerchantApikey{ApiKey: "test-api-key"})
	s.NoError(err)
	s.Equal("success", respApiKey.Status)
}

func (s *MerchantGapiTestSuite) Test10_BulkOperations() {
	ctx := context.Background()

	// Restore All
	_, err := s.commandClient.RestoreAllMerchant(ctx, &emptypb.Empty{})
	s.NoError(err)

	// Delete All Permanent
	_, err = s.commandClient.DeleteAllMerchantPermanent(ctx, &emptypb.Empty{})
	s.NoError(err)
}

func TestMerchantGapiSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(MerchantGapiTestSuite))
}
