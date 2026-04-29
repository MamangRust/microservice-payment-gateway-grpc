package saldo_test

import (
	"context"
	"testing"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/service"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	tests "github.com/MamangRust/microservice-payment-gateway-test"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type SaldoServiceTestSuite struct {
	suite.Suite
	ts           *tests.TestSuite
	dbPool       *pgxpool.Pool
	redisClient  redis.UniversalClient
	saldoService service.Service
	userRepo     user_repo.UserCommandRepository
	cardRepo     card_repo.CardCommandRepository
}

func (s *SaldoServiceTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "saldo"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	opts, err := redis.ParseURL(s.ts.RedisURL)
	s.Require().NoError(err)
	s.redisClient = redis.NewClient(opts)

	queries := db.New(pool)
	
	userRepos := user_repo.NewRepositories(queries)
	cardRepos := card_repo.NewRepositories(queries, nil)
	saldoRepos := saldo_repo.NewRepositories(queries, nil)

	s.userRepo = userRepos.UserCommand()
	s.cardRepo = cardRepos.CardCommand

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("test", lp)
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.redisClient, log, cacheMetrics)

	s.saldoService = service.NewService(&service.Deps{
		Repositories: saldoRepos,
		CardAdapter:  s.ts.CardAdapter,
		Logger:       log,
		Cache:        cacheStore,
	})
}

func (s *SaldoServiceTestSuite) TearDownSuite() {
	if s.redisClient != nil {
		s.redisClient.Close()
	}
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	if s.ts != nil {
		s.ts.Teardown()
	}
}

func (s *SaldoServiceTestSuite) TestSaldoLifecycle() {
	ctx := context.Background()

	// Seed User and Card
	user, err := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Saldo",
		LastName:  "Owner",
		Email:     "saldo.service@example.com",
		Password:  "password123",
	})
	s.Require().NoError(err)

	card, err := s.cardRepo.CreateCard(ctx, &requests.CreateCardRequest{
		UserID:       int(user.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(2, 0, 0),
		CVV:          "123",
		CardProvider: "visa",
	})
	s.Require().NoError(err)

	// 1. Create Saldo
	req := &requests.CreateSaldoRequest{
		CardNumber:   card.CardNumber,
		TotalBalance: 500000,
	}
	saldo, err := s.saldoService.CreateSaldo(ctx, req)
	s.NoError(err)
	s.NotNil(saldo)
	s.Equal(int32(req.TotalBalance), saldo.TotalBalance)

	// 2. Find By Card Number
	found, err := s.saldoService.FindByCardNumber(ctx, card.CardNumber)
	s.NoError(err)
	s.NotNil(found)
	s.Equal(int32(req.TotalBalance), found.TotalBalance)
}

func (s *SaldoServiceTestSuite) TestBulkOperations() {
	ctx := context.Background()

	// Restore All
	success, err := s.saldoService.RestoreAllSaldo(ctx)
	s.NoError(err)
	s.True(success)

	// Delete All Permanent
	success, err = s.saldoService.DeleteAllSaldoPermanent(ctx)
	s.NoError(err)
	s.True(success)
}

func TestSaldoServiceSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(SaldoServiceTestSuite))
}
