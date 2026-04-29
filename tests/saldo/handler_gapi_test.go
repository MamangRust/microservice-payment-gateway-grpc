package saldo_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	card_pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/card"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/saldo"
	pbStats "github.com/MamangRust/microservice-payment-gateway-grpc/pb/saldo/stats"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	gapi "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/handler"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/service"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	tests "github.com/MamangRust/microservice-payment-gateway-test"
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

type SaldoGapiTestSuite struct {
	suite.Suite
	ts          *tests.TestSuite
	dbPool      *pgxpool.Pool
	redisClient redis.UniversalClient
	chConn      clickhouse.Conn
	grpcServer  *grpc.Server
	commandClient pb.SaldoCommandServiceClient
	queryClient   pb.SaldoQueryServiceClient
	statsClient      pbStats.SaldoStatsBalanceServiceClient
	statsTotalClient pbStats.SaldoStatsTotalBalanceClient
	conn        *grpc.ClientConn
	
	userRepo     user_repo.UserCommandRepository
	cardRepo     card_repo.CardCommandRepository
	saldoRepo    saldo_repo.Repositories

	cardNumber string
	saldoID    int32
}

func (s *SaldoGapiTestSuite) SetupSuite() {
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
		FirstName: "Saldo", LastName: "Gapi", Email: "saldo.gapi@test.com", Password: "password123",
	})
	s.Require().NoError(err)
	
	card, err := s.cardRepo.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID: int(user.UserID), CardType: "debit", ExpireDate: time.Now().AddDate(1, 0, 0), CVV: "333", CardProvider: "visa",
	})
	s.Require().NoError(err)
	s.cardNumber = card.CardNumber

	// Start gRPC Server
	saldoHandler := gapi.NewHandler(saldoService)
	
	// Stats Handler
	chRepo := stats_repo.NewRepository(s.chConn)
	saldoStatsHandler := stats_handler.NewSaldoStatsHandler(chRepo, log)

	server := grpc.NewServer()
	pb.RegisterSaldoCommandServiceServer(server, saldoHandler)
	pb.RegisterSaldoQueryServiceServer(server, saldoHandler)
	pbStats.RegisterSaldoStatsBalanceServiceServer(server, saldoStatsHandler)
	pbStats.RegisterSaldoStatsTotalBalanceServer(server, saldoStatsHandler)
	s.grpcServer = server

	lis, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)
	go func() { _ = server.Serve(lis) }()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	s.Require().NoError(err)
	s.conn = conn
	s.commandClient = pb.NewSaldoCommandServiceClient(conn)
	s.queryClient = pb.NewSaldoQueryServiceClient(conn)
	s.statsClient = pbStats.NewSaldoStatsBalanceServiceClient(conn)
	s.statsTotalClient = pbStats.NewSaldoStatsTotalBalanceClient(conn)
}

func (s *SaldoGapiTestSuite) TearDownSuite() {
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

func (s *SaldoGapiTestSuite) Test1_Create() {
	ctx := context.Background()

	createReq := &pb.CreateSaldoRequest{
		CardNumber:   s.cardNumber,
		TotalBalance: 1000000,
	}
	res, err := s.commandClient.CreateSaldo(ctx, createReq)
	s.NoError(err)
	s.Equal(int32(1000000), res.Data.TotalBalance)

	s.saldoID = res.Data.SaldoId
}

func (s *SaldoGapiTestSuite) Test2_FindByCardNumber() {
	s.Require().NotEmpty(s.cardNumber)
	ctx := context.Background()

	found, err := s.queryClient.FindByCardNumber(ctx, &card_pb.FindByCardNumberRequest{CardNumber: s.cardNumber})
	s.NoError(err)
	s.Equal(int32(1000000), found.Data.TotalBalance)
}

func (s *SaldoGapiTestSuite) Test3_Update() {
	s.Require().NotZero(s.saldoID)
	ctx := context.Background()

	_, err := s.commandClient.UpdateSaldo(ctx, &pb.UpdateSaldoRequest{
		SaldoId:      s.saldoID,
		CardNumber:   s.cardNumber,
		TotalBalance: 1200000,
	})
	s.NoError(err)

	// Verify update
	updated, _ := s.queryClient.FindByCardNumber(ctx, &card_pb.FindByCardNumberRequest{CardNumber: s.cardNumber})
	s.Equal(int32(1200000), updated.Data.TotalBalance)
}

func (s *SaldoGapiTestSuite) Test4_Trashed() {
	s.Require().NotZero(s.saldoID)
	ctx := context.Background()

	_, err := s.commandClient.TrashedSaldo(ctx, &pb.FindByIdSaldoRequest{SaldoId: s.saldoID})
	s.NoError(err)
}

func (s *SaldoGapiTestSuite) Test5_Restore() {
	s.Require().NotZero(s.saldoID)
	ctx := context.Background()

	_, err := s.commandClient.RestoreSaldo(ctx, &pb.FindByIdSaldoRequest{SaldoId: s.saldoID})
	s.NoError(err)
}

func (s *SaldoGapiTestSuite) Test6_DeletePermanent() {
	s.Require().NotZero(s.saldoID)
	ctx := context.Background()

	_, err := s.commandClient.DeleteSaldoPermanent(ctx, &pb.FindByIdSaldoRequest{SaldoId: s.saldoID})
	s.NoError(err)
}

func (s *SaldoGapiTestSuite) Test7_SaldoStats_Balance() {
	ctx := context.Background()
	now := time.Now()

	err := s.chConn.Exec(ctx, "TRUNCATE TABLE saldo_events")
	s.Require().NoError(err)

	seedSQL := `INSERT INTO saldo_events (saldo_id, card_number, total_balance, status, created_at) VALUES (?, ?, ?, ?, ?)`
	err = s.chConn.Exec(ctx, seedSQL, 1, s.cardNumber, 1000000, "success", now)
	s.Require().NoError(err)

	s.T().Run("MonthlyBalances", func(t *testing.T) {
		resp, err := s.statsClient.FindMonthlySaldoBalances(ctx, &pb.FindYearlySaldo{Year: int32(now.Year())})
		s.NoError(err)
		s.Equal("success", resp.Status)
		s.NotEmpty(resp.Data)
	})

	s.T().Run("YearlyBalances", func(t *testing.T) {
		resp, err := s.statsClient.FindYearlySaldoBalances(ctx, &pb.FindYearlySaldo{Year: int32(now.Year())})
		s.NoError(err)
		s.Equal("success", resp.Status)
		s.NotEmpty(resp.Data)
	})
}

func (s *SaldoGapiTestSuite) Test8_SaldoStats_TotalBalance() {
	ctx := context.Background()
	now := time.Now()

	s.T().Run("MonthlyTotal", func(t *testing.T) {
		resp, err := s.statsTotalClient.FindMonthlyTotalSaldoBalance(ctx, &pb.FindMonthlySaldoTotalBalance{
			Year:  int32(now.Year()),
			Month: int32(now.Month()),
		})
		s.NoError(err)
		s.Equal("success", resp.Status)
		s.NotEmpty(resp.Data)
	})

	s.T().Run("YearlyTotal", func(t *testing.T) {
		resp, err := s.statsTotalClient.FindYearTotalSaldoBalance(ctx, &pb.FindYearlySaldo{Year: int32(now.Year())})
		s.NoError(err)
		s.Equal("success", resp.Status)
		s.NotEmpty(resp.Data)
	})
}

func (s *SaldoGapiTestSuite) Test9_BulkOperations() {
	ctx := context.Background()

	// Restore All
	_, err := s.commandClient.RestoreAllSaldo(ctx, &emptypb.Empty{})
	s.NoError(err)

	// Delete All Permanent
	_, err = s.commandClient.DeleteAllSaldoPermanent(ctx, &emptypb.Empty{})
	s.NoError(err)
}

func TestSaldoGapiSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(SaldoGapiTestSuite))
}
