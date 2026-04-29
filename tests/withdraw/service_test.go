package withdraw_test

import (
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	withdraw_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/repository"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/service"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	"github.com/MamangRust/microservice-payment-gateway-test"
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type WithdrawServiceTestSuite struct {
	suite.Suite
	ts              *tests.TestSuite
	dbPool          *pgxpool.Pool
	redisClient     redis.UniversalClient
	withdrawService service.Service
	userRepo        user_repo.UserCommandRepository
	cardRepo        card_repo.CardCommandRepository
	saldoRepo       saldo_repo.Repositories
	withdrawID      int32
	cardNumber      string
}

func (s *WithdrawServiceTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "saldo", "withdraw"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	opts, err := redis.ParseURL(s.ts.RedisURL)
	s.Require().NoError(err)
	s.redisClient = redis.NewClient(opts)

	queries := db.New(pool)
	
	// Create individual repositories from their respective modules
	userCommandRepo := user_repo.NewUserCommandRepository(queries)
	cardRepos := card_repo.NewRepositories(queries, nil)
	saldoRepos := saldo_repo.NewRepositories(queries, nil)
	
	repos := withdraw_repo.NewRepositories(queries, cardRepos.CardQuery, saldoRepos)
	s.userRepo = userCommandRepo
	s.cardRepo = cardRepos.CardCommand
	s.saldoRepo = saldoRepos

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("test", lp)
	obs, _ := observability.NewObservability("test", log)
	_ = obs
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.redisClient, log, cacheMetrics)

	s.withdrawService = service.NewService(&service.Deps{
		Kafka:            nil,
		Repositories:     repos,
		CardAdapter:      s.ts.CardAdapter,
		SaldoAdapter:     s.ts.SaldoAdapter,
		Logger:           log,
		Cache:            cacheStore,
		AISecurityClient: nil,
	})

	// Seed User, Card and Saldo
	ctx := context.Background()
	user, _ := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Withdraw",
		LastName:  "User",
		Email:     "withdraw.service@example.com",
		Password:  "password123",
	})
	card, _ := s.cardRepo.CreateCard(ctx, &requests.CreateCardRequest{
		UserID:       int(user.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(2, 0, 0),
		CVV:          "123",
		CardProvider: "visa",
	})
	s.cardNumber = card.CardNumber
	s.saldoRepo.CreateSaldo(ctx, &requests.CreateSaldoRequest{
		CardNumber:   s.cardNumber,
		TotalBalance: 500000,
	})
}

func (s *WithdrawServiceTestSuite) TearDownSuite() {
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

func (s *WithdrawServiceTestSuite) Test1_Create() {
	req := &requests.CreateWithdrawRequest{
		CardNumber:     s.cardNumber,
		WithdrawAmount: 100000,
		WithdrawTime:   time.Now(),
	}
	withdraw, err := s.withdrawService.Create(context.Background(), req)
	s.NoError(err)
	s.NotNil(withdraw)
	s.Equal(int32(req.WithdrawAmount), withdraw.WithdrawAmount)
	s.withdrawID = withdraw.WithdrawID
}

func (s *WithdrawServiceTestSuite) Test2_FindById() {
	s.Require().NotZero(s.withdrawID)

	found, err := s.withdrawService.FindById(context.Background(), int(s.withdrawID))
	s.NoError(err)
	s.NotNil(found)
	s.Equal(s.withdrawID, found.WithdrawID)
}

func (s *WithdrawServiceTestSuite) Test3_FindAll() {
	req := &requests.FindAllWithdraws{
		Page:     1,
		PageSize: 10,
	}
	withdraws, total, err := s.withdrawService.FindAll(context.Background(), req)
	s.NoError(err)
	s.NotNil(withdraws)
	s.NotZero(*total)
}

func (s *WithdrawServiceTestSuite) Test4_Update() {
	s.Require().NotZero(s.withdrawID)

	id := int(s.withdrawID)
	req := &requests.UpdateWithdrawRequest{
		WithdrawID:     &id,
		CardNumber:     s.cardNumber,
		WithdrawAmount: 150000,
		WithdrawTime:   time.Now(),
	}
	updated, err := s.withdrawService.Update(context.Background(), req)
	s.NoError(err)
	s.NotNil(updated)
	s.Equal(int32(req.WithdrawAmount), updated.WithdrawAmount)
}

func (s *WithdrawServiceTestSuite) Test5_Trashed() {
	s.Require().NotZero(s.withdrawID)

	withdraw, err := s.withdrawService.TrashedWithdraw(context.Background(), int(s.withdrawID))
	s.NoError(err)
	s.NotNil(withdraw)
	s.True(withdraw.DeletedAt.Valid)
}

func (s *WithdrawServiceTestSuite) Test6_Restore() {
	s.Require().NotZero(s.withdrawID)

	withdraw, err := s.withdrawService.RestoreWithdraw(context.Background(), int(s.withdrawID))
	s.NoError(err)
	s.NotNil(withdraw)
	s.False(withdraw.DeletedAt.Valid)
}

func (s *WithdrawServiceTestSuite) Test7_DeletePermanent() {
	s.Require().NotZero(s.withdrawID)

	success, err := s.withdrawService.DeleteWithdrawPermanent(context.Background(), int(s.withdrawID))
	s.NoError(err)
	s.True(success)
}

func (s *WithdrawServiceTestSuite) Test8_BulkOperations() {
	ctx := context.Background()

	// Restore All
	success, err := s.withdrawService.RestoreAllWithdraw(ctx)
	s.NoError(err)
	s.True(success)

	// Delete All Permanent
	success, err = s.withdrawService.DeleteAllWithdrawPermanent(ctx)
	s.NoError(err)
	s.True(success)
}

func TestWithdrawServiceSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(WithdrawServiceTestSuite))
}
