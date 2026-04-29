package transaction_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	pbAISecurity "github.com/MamangRust/microservice-payment-gateway-grpc/pb/ai_security"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/transaction"
	pbStats "github.com/MamangRust/microservice-payment-gateway-grpc/pb/transaction/stats"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
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
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	tests "github.com/MamangRust/microservice-payment-gateway-test"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type TransactionGapiTestSuite struct {
	suite.Suite
	ts            *tests.TestSuite
	dbPool        *pgxpool.Pool
	redisClient   redis.UniversalClient
	chConn        clickhouse.Conn
	grpcServer    *grpc.Server
	commandClient pb.TransactionCommandServiceClient
	queryClient   pb.TransactionQueryServiceClient
	statsClient   pbStats.TransactionStatsAmountServiceClient
	methodClient  pbStats.TransactionStatsMethodServiceClient
	statusClient  pbStats.TransactionStatsStatusServiceClient
	conn          *grpc.ClientConn

	// Repositories for seeding
	userRepo     user_repo.UserCommandRepository
	cardRepo     card_repo.Repositories
	saldoRepo    saldo_repo.Repositories
	merchantRepo merchant_repo.Repositories

	customerCardNumber string
	merchantID         int32
	merchantApiKey     string
	transactionID      int32
}

func (s *TransactionGapiTestSuite) SetupSuite() {
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
	_ = log
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.redisClient, log, cacheMetrics)

	cardRepoWrapper := &transactionCardRepo{
		query:   s.cardRepo.CardQuery,
		command: s.cardRepo.CardCommand,
	}

	lis, err := net.Listen("tcp", "localhost:0")
	s.Require().NoError(err)

	transactionRepos := repository.NewRepositories(queries, s.saldoRepo, cardRepoWrapper, s.merchantRepo)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	s.Require().NoError(err)
	s.conn = conn

	aiSecurityClient := pbAISecurity.NewAISecurityServiceClient(conn)

	transactionService := service.NewService(&service.Deps{
		Kafka:            nil,
		Repositories:     transactionRepos,
		MerchantAdapter:  s.ts.MerchantAdapter,
		CardAdapter:      s.ts.CardAdapter,
		SaldoAdapter:     s.ts.SaldoAdapter,
		Logger:           log,
		Cache:            cacheStore,
		AISecurityClient: aiSecurityClient,
	})

	transactionHandler := handler.NewHandler(transactionService)

	// Stats Handler
	chRepo := stats_repo.NewRepository(s.chConn)
	transactionStatsHandler := stats_handler.NewTransactionStatsHandler(chRepo, log)

	server := grpc.NewServer()
	pb.RegisterTransactionCommandServiceServer(server, transactionHandler)
	pb.RegisterTransactionQueryServiceServer(server, transactionHandler)
	pbStats.RegisterTransactionStatsAmountServiceServer(server, transactionStatsHandler)
	pbStats.RegisterTransactionStatsMethodServiceServer(server, transactionStatsHandler)
	pbStats.RegisterTransactionStatsStatusServiceServer(server, transactionStatsHandler)
	pbAISecurity.RegisterAISecurityServiceServer(server, &mockAISecurityServer{})
	s.grpcServer = server

	go func() { _ = server.Serve(lis) }()

	s.commandClient = pb.NewTransactionCommandServiceClient(conn)
	s.queryClient = pb.NewTransactionQueryServiceClient(conn)
	s.statsClient = pbStats.NewTransactionStatsAmountServiceClient(conn)
	s.methodClient = pbStats.NewTransactionStatsMethodServiceClient(conn)
	s.statusClient = pbStats.NewTransactionStatsStatusServiceClient(conn)

	// Seed User, Card, Merchant, Saldo
	user, err := s.userRepo.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Trans", LastName: "Gapi", Email: "trans.gapi@test.com", Password: "password123",
	})
	s.Require().NoError(err)

	card, err := s.cardRepo.CardCommand.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID: int(user.UserID), CardType: "debit", ExpireDate: time.Now().AddDate(1, 0, 0), CVV: "123", CardProvider: "visa",
	})
	s.Require().NoError(err)
	s.customerCardNumber = card.CardNumber

	merchant, err := s.merchantRepo.CreateMerchant(context.Background(), &requests.CreateMerchantRequest{
		UserID: int(user.UserID), Name: "Gapi Merchant",
	})
	s.Require().NoError(err)
	s.merchantID = merchant.MerchantID
	s.merchantApiKey = merchant.ApiKey

	_, err = s.saldoRepo.CreateSaldo(context.Background(), &requests.CreateSaldoRequest{
		CardNumber: s.customerCardNumber, TotalBalance: 1000000,
	})
	s.Require().NoError(err)
}

func (s *TransactionGapiTestSuite) TearDownSuite() {
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

func (s *TransactionGapiTestSuite) Test1_CreateTransaction() {
	ctx := context.Background()
	createReq := &pb.CreateTransactionRequest{
		ApiKey:          s.merchantApiKey,
		CardNumber:      s.customerCardNumber,
		Amount:          100000,
		PaymentMethod:   "visa",
		MerchantId:      s.merchantID,
		TransactionTime: timestamppb.New(time.Now()),
	}
	res, err := s.commandClient.CreateTransaction(ctx, createReq)
	s.Require().NoError(err)
	s.Equal("success", res.Status)
	s.transactionID = res.Data.Id
}

func (s *TransactionGapiTestSuite) Test2_FindTransactionById() {
	ctx := context.Background()
	s.Require().NotZero(s.transactionID)
	res, err := s.queryClient.FindByIdTransaction(ctx, &pb.FindByIdTransactionRequest{TransactionId: s.transactionID})
	s.Require().NoError(err)
	s.Equal(s.transactionID, res.Data.Id)
}

func (s *TransactionGapiTestSuite) Test3_UpdateTransaction() {
	ctx := context.Background()
	s.Require().NotZero(s.transactionID)
	updateReq := &pb.UpdateTransactionRequest{
		ApiKey:          s.merchantApiKey,
		TransactionId:   s.transactionID,
		CardNumber:      s.customerCardNumber,
		Amount:          200000,
		PaymentMethod:   "visa",
		MerchantId:      s.merchantID,
		TransactionTime: timestamppb.New(time.Now()),
	}
	res, err := s.commandClient.UpdateTransaction(ctx, updateReq)
	s.Require().NoError(err)
	s.Equal(int32(200000), res.Data.Amount)
}

func (s *TransactionGapiTestSuite) Test4_TrashedTransaction() {
	ctx := context.Background()
	s.Require().NotZero(s.transactionID)
	res, err := s.commandClient.TrashedTransaction(ctx, &pb.FindByIdTransactionRequest{TransactionId: s.transactionID})
	s.Require().NoError(err)
	s.Equal("success", res.Status)
}

func (s *TransactionGapiTestSuite) Test5_RestoreTransaction() {
	ctx := context.Background()
	s.Require().NotZero(s.transactionID)
	res, err := s.commandClient.RestoreTransaction(ctx, &pb.FindByIdTransactionRequest{TransactionId: s.transactionID})
	s.Require().NoError(err)
	s.Equal("success", res.Status)
}

func (s *TransactionGapiTestSuite) Test6_PermanentDeleteTransaction() {
	ctx := context.Background()
	s.Require().NotZero(s.transactionID)
	res, err := s.commandClient.DeleteTransactionPermanent(ctx, &pb.FindByIdTransactionRequest{TransactionId: s.transactionID})
	s.Require().NoError(err)
	s.Equal("success", res.Status)
}

func (s *TransactionGapiTestSuite) Test7_TransactionStats_Amount() {
	ctx := context.Background()
	now := time.Now()

	err := s.chConn.Exec(ctx, "TRUNCATE TABLE transaction_events")
	s.Require().NoError(err)

	seedSQL := `INSERT INTO transaction_events (transaction_id, transaction_no, merchant_id, card_number, amount, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	err = s.chConn.Exec(ctx, seedSQL, 1, "TXN001", int32(s.merchantID), s.customerCardNumber, 100000, "success", now)
	s.Require().NoError(err)

	s.T().Run("MonthlyAmount", func(t *testing.T) {
		resp, err := s.statsClient.FindMonthlyAmounts(ctx, &pb.FindYearTransactionStatus{Year: int32(now.Year())})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Equal("success", resp.Status)
		s.NotEmpty(resp.Data)
	})

	s.T().Run("YearlyAmount", func(t *testing.T) {
		resp, err := s.statsClient.FindYearlyAmounts(ctx, &pb.FindYearTransactionStatus{Year: int32(now.Year())})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Equal("success", resp.Status)
		s.NotEmpty(resp.Data)
	})

	s.T().Run("MonthlyAmountByCard", func(t *testing.T) {
		resp, err := s.statsClient.FindMonthlyAmountsByCardNumber(ctx, &pb.FindByYearCardNumberTransactionRequest{
			Year: int32(now.Year()), CardNumber: s.customerCardNumber,
		})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Equal("success", resp.Status)
		s.NotEmpty(resp.Data)
	})
}

func (s *TransactionGapiTestSuite) Test8_TransactionStats_Method() {
	ctx := context.Background()
	now := time.Now()

	err := s.chConn.Exec(ctx, "TRUNCATE TABLE transaction_events")
	s.Require().NoError(err)

	seedSQL := `INSERT INTO transaction_events (transaction_id, transaction_no, merchant_id, card_number, amount, payment_method, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	err = s.chConn.Exec(ctx, seedSQL, 1, "TXN001", int32(s.merchantID), s.customerCardNumber, 100000, "visa", "success", now)
	s.Require().NoError(err)

	s.T().Run("MonthlyMethod", func(t *testing.T) {
		resp, err := s.methodClient.FindMonthlyPaymentMethods(ctx, &pb.FindYearTransactionStatus{Year: int32(now.Year())})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Equal("success", resp.Status)
		s.NotEmpty(resp.Data)
	})

	s.T().Run("YearlyMethod", func(t *testing.T) {
		resp, err := s.methodClient.FindYearlyPaymentMethods(ctx, &pb.FindYearTransactionStatus{Year: int32(now.Year())})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Equal("success", resp.Status)
		s.NotEmpty(resp.Data)
	})
}

func (s *TransactionGapiTestSuite) Test9_TransactionStats_Status() {
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
		resp, err := s.statusClient.FindMonthlyTransactionStatusSuccess(ctx, &pb.FindMonthlyTransactionStatus{
			Year: int32(now.Year()), Month: int32(now.Month()),
		})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Equal("success", resp.Status)
		s.NotEmpty(resp.Data)
	})

	s.T().Run("YearlyFailed", func(t *testing.T) {
		resp, err := s.statusClient.FindYearlyTransactionStatusFailed(ctx, &pb.FindYearTransactionStatus{Year: int32(now.Year())})
		s.Require().NoError(err)
		s.Require().NotNil(resp)
		s.Equal("success", resp.Status)
		s.NotEmpty(resp.Data)
	})
}

func (s *TransactionGapiTestSuite) Test11_BulkOperations() {
	ctx := context.Background()

	// Restore All
	_, err := s.commandClient.RestoreAllTransaction(ctx, &emptypb.Empty{})
	s.Require().NoError(err)

	// Delete All Permanent
	_, err = s.commandClient.DeleteAllTransactionPermanent(ctx, &emptypb.Empty{})
	s.Require().NoError(err)
}

func TestTransactionGapiSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TransactionGapiTestSuite))
}
