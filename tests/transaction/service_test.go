package transaction_test

import (
	"context"
	"errors"
	carddb "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	merchantdb "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/database/schema"
	saldodb "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	userdb "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/database/schema"
	"net/http"
	"testing"
	"time"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"

	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	merchant_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/service"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	app_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	tests "github.com/MamangRust/microservice-payment-gateway-test"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type TransactionServiceTestSuite struct {
	suite.Suite
	ts                 *tests.TestSuite
	dbPool             *pgxpool.Pool
	redisClient        redis.UniversalClient
	transactionService service.Service

	// Repositories for seeding
	userRepo     user_repo.UserCommandRepository
	cardRepo     card_repo.Repositories
	saldoRepo    saldo_repo.Repositories
	merchantRepo merchant_repo.Repositories

	customerCardNumber string
	merchantID         int
	merchantApiKey     string
	merchantCardNumber string
	transactionID      int
}

func (s *TransactionServiceTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "merchant", "saldo", "transaction"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	queries := db.New(pool)

	merchantdbQueries := merchantdb.New(pool)

	saldodbQueries := saldodb.New(pool)

	carddbQueries := carddb.New(pool)

	userdbQueries := userdb.New(pool)
	s.userRepo = user_repo.NewUserCommandRepository(userdbQueries)
	s.cardRepo = *card_repo.NewRepositories(carddbQueries, nil)
	s.saldoRepo = saldo_repo.NewRepositories(saldodbQueries, nil)
	s.merchantRepo = merchant_repo.NewRepositories(merchantdbQueries, nil)

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

	transactionRepos := repository.NewRepositories(queries, s.saldoRepo, cardRepoWrapper, s.merchantRepo)
	s.transactionService = service.NewService(&service.Deps{
		Kafka:            nil,
		Repositories:     transactionRepos,
		MerchantAdapter:  s.ts.MerchantAdapter,
		CardAdapter:      s.ts.CardAdapter,
		SaldoAdapter:     s.ts.SaldoAdapter,
		Logger:           log,
		Cache:            cacheStore,
		AISecurityClient: nil,
	})

	// Seed User, Card, Merchant, Saldo
	ctx := context.Background()
	user, _ := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Tx", LastName: "Owner", Email: "tx.service@example.com", Password: "password123",
	})
	card, _ := s.cardRepo.CardCommand.CreateCard(ctx, &requests.CreateCardRequest{
		UserID: int(user.UserID), CardType: "debit", ExpireDate: time.Now().AddDate(2, 0, 0), CVV: "123", CardProvider: "visa",
	})
	s.customerCardNumber = card.CardNumber

	merchant, _ := s.merchantRepo.CreateMerchant(ctx, &requests.CreateMerchantRequest{
		Name: "Service Merchant", UserID: int(user.UserID),
	})
	s.merchantID = int(merchant.MerchantID)
	s.merchantApiKey = merchant.ApiKey

	s.saldoRepo.CreateSaldo(ctx, &requests.CreateSaldoRequest{
		CardNumber: s.customerCardNumber, TotalBalance: 1000000,
	})

	merchantCard, _ := s.cardRepo.CardCommand.CreateCard(ctx, &requests.CreateCardRequest{
		UserID: int(user.UserID), CardType: "debit", ExpireDate: time.Now().AddDate(2, 0, 0), CVV: "456", CardProvider: "mastercard",
	})
	s.merchantCardNumber = merchantCard.CardNumber
	s.saldoRepo.CreateSaldo(ctx, &requests.CreateSaldoRequest{
		CardNumber: s.merchantCardNumber, TotalBalance: 0,
	})
}

func (s *TransactionServiceTestSuite) TearDownSuite() {
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

func (s *TransactionServiceTestSuite) Test1_CreateTransaction() {
	ctx := context.Background()
	merchantID := s.merchantID
	req := &requests.CreateTransactionRequest{
		CardNumber:      s.customerCardNumber,
		Amount:          100000,
		PaymentMethod:   "visa",
		MerchantID:      &merchantID,
		TransactionTime: time.Now(),
	}
	tx, err := s.transactionService.Create(ctx, s.merchantApiKey, req)
	s.NoError(err)
	s.NotNil(tx)
	s.transactionID = int(tx.TransactionID)
}

func (s *TransactionServiceTestSuite) Test2_FindTransactionById() {
	ctx := context.Background()
	s.Require().NotZero(s.transactionID)
	res, err := s.transactionService.FindById(ctx, s.transactionID)
	s.NoError(err)
	s.NotNil(res)
	s.Equal(int32(s.transactionID), res.TransactionID)
}

func (s *TransactionServiceTestSuite) Test3_UpdateTransaction() {
	ctx := context.Background()
	s.Require().NotZero(s.transactionID)
	merchantID := s.merchantID
	req := &requests.UpdateTransactionRequest{
		TransactionID:   &s.transactionID,
		CardNumber:      s.customerCardNumber,
		Amount:          150000,
		MerchantID:      &merchantID,
		PaymentMethod:   "visa",
		TransactionTime: time.Now(),
	}
	res, err := s.transactionService.Update(ctx, s.merchantApiKey, req)
	s.NoError(err)
	s.NotNil(res)
	s.Equal(int64(150000), res.Amount)
}

func (s *TransactionServiceTestSuite) Test4_TrashedTransaction() {
	ctx := context.Background()
	s.Require().NotZero(s.transactionID)
	res, err := s.transactionService.TrashedTransaction(ctx, s.transactionID)
	s.NoError(err)
	s.NotNil(res.DeletedAt)
}

func (s *TransactionServiceTestSuite) Test5_RestoreTransaction() {
	ctx := context.Background()
	s.Require().NotZero(s.transactionID)
	res, err := s.transactionService.RestoreTransaction(ctx, s.transactionID)
	s.NoError(err)
	s.True(res.DeletedAt.Time.IsZero())
}

func (s *TransactionServiceTestSuite) Test6_PermanentDeleteTransaction() {
	ctx := context.Background()
	s.Require().NotZero(s.transactionID)
	success, err := s.transactionService.DeleteTransactionPermanent(ctx, s.transactionID)
	s.NoError(err)
	s.True(success)
}

func (s *TransactionServiceTestSuite) Test7_BulkOperations() {
	ctx := context.Background()

	// Restore All
	success, err := s.transactionService.RestoreAllTransaction(ctx)
	s.NoError(err)
	s.True(success)

	// Delete All Permanent
	success, err = s.transactionService.DeleteAllTransactionPermanent(ctx)
	s.NoError(err)
	s.True(success)
}

func (s *TransactionServiceTestSuite) Test8_Idempotency_SameKeyReplaysWithoutDoubleDebit() {
	ctx := context.Background()

	// Fresh customer card with a known balance so balance assertions stay deterministic.
	user, err := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Idem", LastName: "Tx", Email: "idem.tx@example.com", Password: "password123",
	})
	s.Require().NoError(err)
	card, err := s.cardRepo.CardCommand.CreateCard(ctx, &requests.CreateCardRequest{
		UserID: int(user.UserID), CardType: "debit", ExpireDate: time.Now().AddDate(1, 0, 0), CVV: "888", CardProvider: "visa",
	})
	s.Require().NoError(err)
	_, err = s.saldoRepo.CreateSaldo(ctx, &requests.CreateSaldoRequest{CardNumber: card.CardNumber, TotalBalance: 300000})
	s.Require().NoError(err)

	merchantID := s.merchantID
	req := &requests.CreateTransactionRequest{
		CardNumber:      card.CardNumber,
		Amount:          100000,
		PaymentMethod:   "visa",
		MerchantID:      &merchantID,
		TransactionTime: time.Now(),
		IdempotencyKey:  "transaction-idem-key-1",
	}

	// First call executes and debits the customer balance once.
	first, err := s.transactionService.Create(ctx, s.merchantApiKey, req)
	s.Require().NoError(err)
	s.Require().NotNil(first)

	// Retry with the exact same key + payload must replay, not re-execute.
	replay, err := s.transactionService.Create(ctx, s.merchantApiKey, req)
	s.Require().NoError(err)
	s.Require().NotNil(replay)
	s.Equal(first.TransactionID, replay.TransactionID, "replay must return the original record")

	bal, err := s.saldoRepo.FindByCardNumber(ctx, card.CardNumber)
	s.Require().NoError(err)
	s.Equal(int64(200000), bal.TotalBalance, "balance must be debited exactly once")

	// Same key with a different payload must be rejected with a conflict.
	conflictReq := &requests.CreateTransactionRequest{
		CardNumber:      card.CardNumber,
		Amount:          150000,
		PaymentMethod:   "visa",
		MerchantID:      &merchantID,
		TransactionTime: time.Now(),
		IdempotencyKey:  "transaction-idem-key-1",
	}
	_, err = s.transactionService.Create(ctx, s.merchantApiKey, conflictReq)
	s.Require().Error(err)
	var appErr *app_errors.AppError
	s.True(errors.As(err, &appErr), "conflict must surface as an AppError")
	s.Equal(http.StatusConflict, appErr.Code)

	// Balance is still debited exactly once after the rejected retry.
	bal, err = s.saldoRepo.FindByCardNumber(ctx, card.CardNumber)
	s.Require().NoError(err)
	s.Equal(int64(200000), bal.TotalBalance)
}

func TestTransactionServiceSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TransactionServiceTestSuite))
}
