package transfer_test

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
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type TransferServiceTestSuite struct {
	suite.Suite
	ts              *tests.TestSuite
	dbPool          *pgxpool.Pool
	redisClient     redis.UniversalClient
	transferService service.Service
	userRepo        user_repo.UserCommandRepository
	cardRepo        card_repo.Repositories
	saldoRepo       saldo_repo.Repositories
	repos           repository.Repositories

	senderCardNumber   string
	receiverCardNumber string
	transferID         int
}

func (s *TransferServiceTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "saldo", "transfer"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	queries := db.New(pool)
	
	// Repositories for seeding
	s.userRepo = user_repo.NewUserCommandRepository(queries)
	s.cardRepo = *card_repo.NewRepositories(queries, nil)
	s.saldoRepo = saldo_repo.NewRepositories(queries, nil)

	// Transfer repos
	s.repos = repository.NewRepositories(queries, s.saldoRepo, s.cardRepo.CardQuery)

	opts, err := redis.ParseURL(s.ts.RedisURL)
	s.Require().NoError(err)
	s.redisClient = redis.NewClient(opts)

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("test", lp)
	_ , _ = observability.NewObservability("test", log)
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.redisClient, log, cacheMetrics)

	s.transferService = service.NewService(&service.Deps{
		Kafka:            nil,
		Repositories:     s.repos,
		CardAdapter:      s.ts.CardAdapter,
		SaldoAdapter:     s.ts.SaldoAdapter,
		Logger:           log,
		Cache:            cacheStore,
		AISecurityClient: nil,
	})

	// Seed Sender
	sender, err := s.userRepo.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Sender",
		LastName:  "Service",
		Email:     "sender.service@test.com",
		Password:  "password123",
	})
	s.Require().NoError(err)

	sCard, err := s.cardRepo.CardCommand.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID:       int(sender.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(1, 0, 0),
		CVV:          "111",
		CardProvider: "visa",
	})
	s.Require().NoError(err)
	s.senderCardNumber = sCard.CardNumber

	_, err = s.saldoRepo.CreateSaldo(context.Background(), &requests.CreateSaldoRequest{
		CardNumber:   s.senderCardNumber,
		TotalBalance: 1000000,
	})
	s.Require().NoError(err)

	// Seed Receiver
	receiver, err := s.userRepo.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Receiver",
		LastName:  "Service",
		Email:     "receiver.service@test.com",
		Password:  "password123",
	})
	s.Require().NoError(err)

	rCard, err := s.cardRepo.CardCommand.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID:       int(receiver.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(1, 0, 0),
		CVV:          "222",
		CardProvider: "mastercard",
	})
	s.Require().NoError(err)
	s.receiverCardNumber = rCard.CardNumber

	_, err = s.saldoRepo.CreateSaldo(context.Background(), &requests.CreateSaldoRequest{
		CardNumber:   s.receiverCardNumber,
		TotalBalance: 0,
	})
	s.Require().NoError(err)
}

func (s *TransferServiceTestSuite) TearDownSuite() {
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

func (s *TransferServiceTestSuite) Test1_CreateTransfer() {
	ctx := context.Background()
	req := &requests.CreateTransferRequest{
		TransferFrom:   s.senderCardNumber,
		TransferTo:     s.receiverCardNumber,
		TransferAmount: 100000,
	}

	res, err := s.transferService.CreateTransaction(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.transferID = int(res.TransferID)

	// Verify balances
	senderSaldo, _ := s.saldoRepo.FindByCardNumber(ctx, s.senderCardNumber)
	s.Equal(int32(900000), senderSaldo.TotalBalance)

	receiverSaldo, _ := s.saldoRepo.FindByCardNumber(ctx, s.receiverCardNumber)
	s.Equal(int32(100000), receiverSaldo.TotalBalance)
}

func (s *TransferServiceTestSuite) Test2_FindTransferById() {
	s.Require().NotZero(s.transferID)
	ctx := context.Background()

	res, err := s.transferService.FindById(ctx, s.transferID)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(int32(s.transferID), res.TransferID)
}

func (s *TransferServiceTestSuite) Test3_FindAllTransfers() {
	ctx := context.Background()
	res, _, err := s.transferService.FindAll(ctx, &requests.FindAllTransfers{
		Page:     1,
		PageSize: 10,
	})
	s.Require().NoError(err)
	s.Require().NotNil(res)
}

func (s *TransferServiceTestSuite) Test4_TrashAndRestore() {
	s.Require().NotZero(s.transferID)
	ctx := context.Background()

	_, err := s.transferService.TrashedTransfer(ctx, s.transferID)
	s.NoError(err)

	_, err = s.transferService.RestoreTransfer(ctx, s.transferID)
	s.NoError(err)
}

func (s *TransferServiceTestSuite) Test5_DeletePermanent() {
	s.Require().NotZero(s.transferID)
	ctx := context.Background()

	success, err := s.transferService.DeleteTransferPermanent(ctx, s.transferID)
	s.NoError(err)
	s.True(success)
}

func (s *TransferServiceTestSuite) Test6_BulkOperations() {
	ctx := context.Background()

	// Restore All
	success, err := s.transferService.RestoreAllTransfer(ctx)
	s.NoError(err)
	s.True(success)

	// Delete All Permanent
	success, err = s.transferService.DeleteAllTransferPermanent(ctx)
	s.NoError(err)
	s.True(success)
}

func TestTransferServiceSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TransferServiceTestSuite))
}
