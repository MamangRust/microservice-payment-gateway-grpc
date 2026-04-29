package topup_test

import (
	"context"
	"testing"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	tests "github.com/MamangRust/microservice-payment-gateway-test"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	topup_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/repository"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type TopupServiceTestSuite struct {
	suite.Suite
	ts           *tests.TestSuite
	dbPool       *pgxpool.Pool
	redisClient  redis.UniversalClient
	topupService service.Service
	userRepo     user_repo.UserCommandRepository
	cardRepo     card_repo.CardCommandRepository
	saldoRepo    saldo_repo.SaldoCommandRepository
	topupID      int32
	cardNumber   string
}

func (s *TopupServiceTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "saldo", "topup"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	opts, err := redis.ParseURL(s.ts.RedisURL)
	s.Require().NoError(err)
	s.redisClient = redis.NewClient(opts)

	queries := db.New(pool)
	
	// Initialize repos from their modules
	userRepos := user_repo.NewRepositories(queries)
	cardRepos := card_repo.NewRepositories(queries, nil)
	saldoRepos := saldo_repo.NewRepositories(queries, nil)
	
	// Match topup repository interfaces
	cardAdapter := &topupCardRepoAdapter{
		CardQueryRepository:   cardRepos.CardQuery,
		CardCommandRepository: cardRepos.CardCommand,
	}
	topupRepos := topup_repo.NewRepositories(queries, cardAdapter, saldoRepos)

	s.userRepo = userRepos.UserCommand()
	s.cardRepo = cardRepos.CardCommand
	s.saldoRepo = saldoRepos

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("test", lp)
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.redisClient, log, cacheMetrics)

	s.topupService = service.NewService(&service.Deps{
		Kafka:        nil,
		Cache:        cacheStore,
		Repositories: topupRepos,
		CardAdapter:  s.ts.CardAdapter,
		SaldoAdapter: s.ts.SaldoAdapter,
		Logger:       log,
	})

	// Seed User and Card
	ctx := context.Background()
	user, err := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Topup",
		LastName:  "Owner",
		Email:     "topup.service@example.com",
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
	s.cardNumber = card.CardNumber

	_, err = s.saldoRepo.CreateSaldo(ctx, &requests.CreateSaldoRequest{
		CardNumber:   s.cardNumber,
		TotalBalance: 0,
	})
	s.Require().NoError(err)
}

func (s *TopupServiceTestSuite) TearDownSuite() {
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

func (s *TopupServiceTestSuite) Test1_CreateTopup() {
	ctx := context.Background()

	req := &requests.CreateTopupRequest{
		CardNumber:  s.cardNumber,
		TopupAmount: 100000,
		TopupMethod: "visa",
	}
	topup, err := s.topupService.CreateTopup(ctx, req)
	s.NoError(err)
	s.NotNil(topup)
	s.Equal(int32(req.TopupAmount), topup.TopupAmount)
	s.topupID = topup.TopupID
}

func (s *TopupServiceTestSuite) Test2_FindById() {
	s.Require().NotZero(s.topupID)
	ctx := context.Background()

	found, err := s.topupService.FindById(ctx, int(s.topupID))
	s.NoError(err)
	s.NotNil(found)
	s.Equal(int32(100000), found.TopupAmount)
}

func (s *TopupServiceTestSuite) Test3_FindAll() {
	ctx := context.Background()
	req := &requests.FindAllTopups{
		Page:     1,
		PageSize: 10,
	}
	topups, total, err := s.topupService.FindAll(ctx, req)
	s.NoError(err)
	s.NotNil(topups)
	s.NotZero(*total)
}

func (s *TopupServiceTestSuite) Test4_UpdateTopup() {
	s.Require().NotZero(s.topupID)
	ctx := context.Background()

	id := int(s.topupID)
	req := &requests.UpdateTopupRequest{
		TopupID:     &id,
		CardNumber:  s.cardNumber,
		TopupAmount: 150000,
		TopupMethod: "visa",
	}
	updated, err := s.topupService.UpdateTopup(ctx, req)
	s.NoError(err)
	s.NotNil(updated)
	s.Equal(int32(150000), updated.TopupAmount)
}

func (s *TopupServiceTestSuite) Test5_TrashAndRestore() {
	s.Require().NotZero(s.topupID)
	ctx := context.Background()

	_, err := s.topupService.TrashedTopup(ctx, int(s.topupID))
	s.NoError(err)

	_, err = s.topupService.RestoreTopup(ctx, int(s.topupID))
	s.NoError(err)
}

func (s *TopupServiceTestSuite) Test6_DeletePermanent() {
	s.Require().NotZero(s.topupID)
	ctx := context.Background()

	success, err := s.topupService.DeleteTopupPermanent(ctx, int(s.topupID))
	s.NoError(err)
	s.True(success)
}

func (s *TopupServiceTestSuite) Test7_BulkOperations() {
	ctx := context.Background()

	// Restore All
	success, err := s.topupService.RestoreAllTopup(ctx)
	s.NoError(err)
	s.True(success)

	// Delete All Permanent
	success, err = s.topupService.DeleteAllTopupPermanent(ctx)
	s.NoError(err)
	s.True(success)
}

func TestTopupServiceSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TopupServiceTestSuite))
}
