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

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// faultInjectingSaldoAdapter wraps the real local saldo adapter and can inject
// transport failures into DebitSaldo / CreditSaldo on demand.
type faultInjectingSaldoAdapter struct {
	inner      adapter.SaldoAdapter
	failDebit  bool
	failCredit bool
}

func (f *faultInjectingSaldoAdapter) FindByCardNumber(ctx context.Context, cardNumber string) (*saldodb.Saldo, error) {
	return f.inner.FindByCardNumber(ctx, cardNumber)
}

func (f *faultInjectingSaldoAdapter) UpdateSaldoBalance(ctx context.Context, req *requests.UpdateSaldoBalance) (*saldodb.UpdateSaldoBalanceRow, error) {
	return f.inner.UpdateSaldoBalance(ctx, req)
}

func (f *faultInjectingSaldoAdapter) DebitSaldo(ctx context.Context, req *requests.DebitSaldoRequest) (*saldodb.DebitSaldoRow, error) {
	if f.failDebit {
		return nil, status.Error(codes.Unavailable, "saldo service unavailable (injected)")
	}
	return f.inner.DebitSaldo(ctx, req)
}

func (f *faultInjectingSaldoAdapter) CreditSaldo(ctx context.Context, req *requests.CreditSaldoRequest) (*saldodb.CreditSaldoRow, error) {
	if f.failCredit {
		return nil, status.Error(codes.Unavailable, "saldo service unavailable (injected)")
	}
	return f.inner.CreditSaldo(ctx, req)
}

func (f *faultInjectingSaldoAdapter) UpdateSaldoWithdraw(ctx context.Context, req *requests.UpdateSaldoWithdraw) (*saldodb.UpdateSaldoWithdrawRow, error) {
	return f.inner.UpdateSaldoWithdraw(ctx, req)
}

type TransactionFailureInjectionTestSuite struct {
	suite.Suite
	ts             *tests.TestSuite
	dbPool         *pgxpool.Pool
	redisClient    redis.UniversalClient
	queries        *db.Queries
	transactionSvc service.Service
	injectedSaldo  *faultInjectingSaldoAdapter
	customerCard   string
	merchantID     int
	merchantApiKey string
}

func (s *TransactionFailureInjectionTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "merchant", "saldo", "transaction"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool
	s.queries = db.New(pool)

	userRepo := user_repo.NewUserCommandRepository(userdb.New(pool))
	cardRepo := *card_repo.NewRepositories(carddb.New(pool), nil)
	saldoRepo := saldo_repo.NewRepositories(saldodb.New(pool), nil)
	merchantRepo := merchant_repo.NewRepositories(merchantdb.New(pool), nil)

	opts, err := redis.ParseURL(s.ts.RedisURL)
	s.Require().NoError(err)
	s.redisClient = redis.NewClient(opts)

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("failure-injection-test", lp)
	cacheMetrics, _ := observability.NewCacheMetrics("failure-injection-test")
	cacheStore := cache.NewCacheStore(s.redisClient, log, cacheMetrics)

	// Real adapters, wrapped with fault injection for the saldo dependency.
	s.injectedSaldo = &faultInjectingSaldoAdapter{inner: s.ts.SaldoAdapter}

	cardRepoWrapper := &transactionCardRepo{
		query:   cardRepo.CardQuery,
		command: cardRepo.CardCommand,
	}

	transactionRepos := repository.NewRepositories(s.queries, s.injectedSaldo, cardRepoWrapper, merchantRepo)
	s.transactionSvc = service.NewService(&service.Deps{
		Kafka:            nil,
		Repositories:     transactionRepos,
		MerchantAdapter:  s.ts.MerchantAdapter,
		CardAdapter:      s.ts.CardAdapter,
		SaldoAdapter:     s.injectedSaldo,
		Logger:           log,
		Cache:            cacheStore,
		AISecurityClient: nil,
	})

	// Seed: user -> card + saldo -> merchant.
	ctx := context.Background()
	user, err := userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Fail", LastName: "Inject", Email: "fail.inject@example.com", Password: "password123",
	})
	s.Require().NoError(err)
	card, err := cardRepo.CardCommand.CreateCard(ctx, &requests.CreateCardRequest{
		UserID: int(user.UserID), CardType: "debit", ExpireDate: time.Now().AddDate(2, 0, 0), CVV: "999", CardProvider: "visa",
	})
	s.Require().NoError(err)
	s.customerCard = card.CardNumber
	_, err = saldoRepo.CreateSaldo(ctx, &requests.CreateSaldoRequest{CardNumber: card.CardNumber, TotalBalance: 100000})
	s.Require().NoError(err)

	merchant, err := merchantRepo.CreateMerchant(ctx, &requests.CreateMerchantRequest{Name: "Failure Merchant", UserID: int(user.UserID)})
	s.Require().NoError(err)
	s.merchantID = int(merchant.MerchantID)
	s.merchantApiKey = merchant.ApiKey

	merchantCard, err := cardRepo.CardCommand.CreateCard(ctx, &requests.CreateCardRequest{
		UserID: int(user.UserID), CardType: "debit", ExpireDate: time.Now().AddDate(2, 0, 0), CVV: "111", CardProvider: "mastercard",
	})
	s.Require().NoError(err)
	_, err = saldoRepo.CreateSaldo(ctx, &requests.CreateSaldoRequest{CardNumber: merchantCard.CardNumber, TotalBalance: 0})
	s.Require().NoError(err)
}

func (s *TransactionFailureInjectionTestSuite) TearDownSuite() {
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

func (s *TransactionFailureInjectionTestSuite) createReq(amount int) *requests.CreateTransactionRequest {
	merchantID := s.merchantID
	return &requests.CreateTransactionRequest{
		CardNumber:      s.customerCard,
		Amount:          amount,
		PaymentMethod:   "visa",
		MerchantID:      &merchantID,
		TransactionTime: time.Now(),
	}
}

// TestF1_InvalidAmount_RejectedWith400 verifies amount validation happens
// before any remote call and surfaces as a 400, never a 500.
func (s *TransactionFailureInjectionTestSuite) TestF1_InvalidAmount_RejectedWith400() {
	ctx := context.Background()

	for _, amount := range []int{0, -100} {
		_, err := s.transactionSvc.Create(ctx, s.merchantApiKey, s.createReq(amount))
		s.Require().Error(err, "amount %d must be rejected", amount)

		var appErr *app_errors.AppError
		s.True(errors.As(err, &appErr), "validation error must be an AppError, got %T", err)
		s.Equal(http.StatusBadRequest, appErr.Code, "invalid amount must map to 400")
	}
}

// TestF2_InsufficientBalance_Conflict verifies insufficient funds surface as a
// 409 Conflict (P1.5 error contract), not an internal 500.
func (s *TransactionFailureInjectionTestSuite) TestF2_InsufficientBalance_Conflict() {
	ctx := context.Background()
	_, err := s.transactionSvc.Create(ctx, s.merchantApiKey, s.createReq(100000000))
	s.Require().Error(err)

	var appErr *app_errors.AppError
	s.True(errors.As(err, &appErr), "insufficient balance must be an AppError, got %T", err)
	s.Equal(http.StatusConflict, appErr.Code, "insufficient balance must map to 409")
}

// TestF3_DebitFailure_MarksFailed verifies that when the atomic debit fails the
// transaction is moved to 'failed' and no nil dereference / panic occurs.
func (s *TransactionFailureInjectionTestSuite) TestF3_DebitFailure_MarksFailed() {
	ctx := context.Background()
	s.injectedSaldo.failDebit = true
	defer func() { s.injectedSaldo.failDebit = false }()

	_, err := s.transactionSvc.Create(ctx, s.merchantApiKey, s.createReq(1000))
	s.Require().Error(err)

	// The durable row must exist and be marked failed.
	rows, qErr := s.queries.GetTransactionByCardNumber(ctx, db.GetTransactionByCardNumberParams{
		CardNumber: s.customerCard, Limit: 10, Offset: 0,
	})
	s.Require().NoError(qErr)
	s.Require().NotEmpty(rows)
	s.Equal("failed", rows[0].Status, "transaction must be failed after debit failure")
}

// TestF4_CreditFailure_QuarantinesUnknown verifies that an ambiguous merchant
// credit outcome quarantines the transaction as 'unknown' instead of claiming
// success or compensating blindly.
func (s *TransactionFailureInjectionTestSuite) TestF4_CreditFailure_QuarantinesUnknown() {
	ctx := context.Background()
	s.injectedSaldo.failCredit = true
	defer func() { s.injectedSaldo.failCredit = false }()

	_, err := s.transactionSvc.Create(ctx, s.merchantApiKey, s.createReq(1000))
	s.Require().Error(err)

	rows, qErr := s.queries.GetTransactionByCardNumber(ctx, db.GetTransactionByCardNumberParams{
		CardNumber: s.customerCard, Limit: 10, Offset: 0,
	})
	s.Require().NoError(qErr)
	s.Require().NotEmpty(rows)
	s.Equal("unknown", rows[0].Status, "ambiguous credit must quarantine as unknown for reconciliation")
}

func TestTransactionFailureInjectionSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TransactionFailureInjectionTestSuite))
}
