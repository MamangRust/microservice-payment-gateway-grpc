package topup_test

import (
	"context"
	"errors"
	carddb "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	saldodb "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	userdb "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/database/schema"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/database/schema"
	topup_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/service"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	app_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	tests "github.com/MamangRust/microservice-payment-gateway-test"
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

	saldodbQueries := saldodb.New(s.dbPool)

	carddbQueries := carddb.New(s.dbPool)

	userdbQueries := userdb.New(s.dbPool)

	// Initialize repos from their modules
	userRepos := user_repo.NewRepositories(userdbQueries)
	cardRepos := card_repo.NewRepositories(carddbQueries, nil)
	saldoRepos := saldo_repo.NewRepositories(saldodbQueries, nil)

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
	s.Equal(int64(req.TopupAmount), topup.TopupAmount)
	s.topupID = topup.TopupID
}

func (s *TopupServiceTestSuite) Test2_FindById() {
	s.Require().NotZero(s.topupID)
	ctx := context.Background()

	found, err := s.topupService.FindById(ctx, int(s.topupID))
	s.NoError(err)
	s.NotNil(found)
	s.Equal(int64(100000), found.TopupAmount)
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
	s.Equal(int64(150000), updated.TopupAmount)
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

func (s *TopupServiceTestSuite) Test8_Idempotency_SameKeyReplaysWithoutDoubleCredit() {
	ctx := context.Background()

	// Fresh card with an empty balance so balance assertions stay deterministic.
	user, err := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Idem", LastName: "Topup", Email: "idem.topup@example.com", Password: "password123",
	})
	s.Require().NoError(err)
	card, err := s.cardRepo.CreateCard(ctx, &requests.CreateCardRequest{
		UserID:       int(user.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(1, 0, 0),
		CVV:          "555",
		CardProvider: "visa",
	})
	s.Require().NoError(err)
	_, err = s.saldoRepo.CreateSaldo(ctx, &requests.CreateSaldoRequest{CardNumber: card.CardNumber, TotalBalance: 0})
	s.Require().NoError(err)

	req := &requests.CreateTopupRequest{
		CardNumber:     card.CardNumber,
		TopupAmount:    50000,
		TopupMethod:    "visa",
		IdempotencyKey: "topup-idem-key-1",
	}

	// First call executes and credits the balance once.
	first, err := s.topupService.CreateTopup(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(first)

	// Retry with the exact same key + payload must replay, not re-execute.
	replay, err := s.topupService.CreateTopup(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(replay)
	s.Equal(first.TopupID, replay.TopupID, "replay must return the original record")

	bal, err := saldodb.New(s.dbPool).GetSaldoByCardNumber(ctx, card.CardNumber)
	s.Require().NoError(err)
	s.Equal(int64(50000), bal.TotalBalance, "balance must be credited exactly once")

	// Same key with a different payload must be rejected with a conflict.
	conflictReq := &requests.CreateTopupRequest{
		CardNumber:     card.CardNumber,
		TopupAmount:    75000,
		TopupMethod:    "visa",
		IdempotencyKey: "topup-idem-key-1",
	}
	_, err = s.topupService.CreateTopup(ctx, conflictReq)
	s.Require().Error(err)
	var appErr *app_errors.AppError
	s.True(errors.As(err, &appErr), "conflict must surface as an AppError")
	s.Equal(http.StatusConflict, appErr.Code)

	// Balance is still credited exactly once after the rejected retry.
	bal, err = saldodb.New(s.dbPool).GetSaldoByCardNumber(ctx, card.CardNumber)
	s.Require().NoError(err)
	s.Equal(int64(50000), bal.TotalBalance)
}

func (s *TopupServiceTestSuite) Test9_Idempotency_ConcurrentSameKeyCreditsOnce() {
	ctx := context.Background()

	// Fresh card with an empty balance so balance assertions stay deterministic.
	user, err := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Idem", LastName: "Concurrent", Email: "idem.concurrent@example.com", Password: "password123",
	})
	s.Require().NoError(err)
	card, err := s.cardRepo.CreateCard(ctx, &requests.CreateCardRequest{
		UserID:       int(user.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(1, 0, 0),
		CVV:          "999",
		CardProvider: "visa",
	})
	s.Require().NoError(err)
	_, err = s.saldoRepo.CreateSaldo(ctx, &requests.CreateSaldoRequest{CardNumber: card.CardNumber, TotalBalance: 0})
	s.Require().NoError(err)

	req := &requests.CreateTopupRequest{
		CardNumber:     card.CardNumber,
		TopupAmount:    50000,
		TopupMethod:    "visa",
		IdempotencyKey: "topup-concurrent-key-1",
	}

	const workers = 5
	var wg sync.WaitGroup
	results := make([]error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := s.topupService.CreateTopup(ctx, req)
			results[idx] = err
		}(i)
	}
	wg.Wait()

	// Exactly one goroutine may succeed; the rest must be replay or "still
	// processing" conflicts. No request may succeed more than once because the
	// balance must reflect a single credit.
	bal, err := saldodb.New(s.dbPool).GetSaldoByCardNumber(ctx, card.CardNumber)
	s.Require().NoError(err)
	s.Equal(int64(50000), bal.TotalBalance, "balance must be credited exactly once under concurrency")
}

func TestTopupServiceSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TopupServiceTestSuite))
}
