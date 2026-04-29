package transaction_test

import (
	"context"
	"fmt"
	"testing"
	"time"


	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	tests "github.com/MamangRust/microservice-payment-gateway-test"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/repository"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	merchant_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
)

type TransactionRepositoryTestSuite struct {
	suite.Suite
	ts            *tests.TestSuite
	dbPool        *pgxpool.Pool
	commandRepo   repository.TransactionCommandRepository
	queryRepo     repository.TransactionQueryRepository
	
	// Repositories for seeding
	userRepo     user_repo.UserCommandRepository
	cardRepo     card_repo.Repositories
	merchantRepo merchant_repo.Repositories

	customerCardNumber string
	merchantID         int
}

func (s *TransactionRepositoryTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "merchant", "saldo", "transaction"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	queries := db.New(pool)
	s.userRepo = user_repo.NewUserCommandRepository(queries)
	s.cardRepo = *card_repo.NewRepositories(queries, nil)
	s.merchantRepo = merchant_repo.NewRepositories(queries, nil)

	transactionRepos := repository.NewRepositories(queries, nil, nil, nil)
	s.commandRepo = transactionRepos
	s.queryRepo = transactionRepos

	// Seed initial data for tests
	ctx := context.Background()
	user, err := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Repo", LastName: "Owner", Email: fmt.Sprintf("repo-%d@test.com", time.Now().UnixNano()), Password: "password123",
	})
	s.Require().NoError(err)

	card, err := s.cardRepo.CardCommand.CreateCard(ctx, &requests.CreateCardRequest{
		UserID: int(user.UserID), CardType: "debit", ExpireDate: time.Now().AddDate(2, 0, 0), CVV: "123", CardProvider: "visa",
	})
	s.Require().NoError(err)
	s.customerCardNumber = card.CardNumber

	merchant, err := s.merchantRepo.CreateMerchant(ctx, &requests.CreateMerchantRequest{
		Name: "Repo Merchant", UserID: int(user.UserID),
	})
	s.Require().NoError(err)
	s.merchantID = int(merchant.MerchantID)
}

func (s *TransactionRepositoryTestSuite) TearDownSuite() {
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	if s.ts != nil {
		s.ts.Teardown()
	}
}

func (s *TransactionRepositoryTestSuite) createSeedTransaction() (*db.CreateTransactionRow, error) {
	ctx := context.Background()
	req := &requests.CreateTransactionRequest{
		CardNumber:      s.customerCardNumber,
		Amount:          100000,
		MerchantID:      &s.merchantID,
		PaymentMethod:   "visa",
		TransactionTime: time.Now(),
	}
	return s.commandRepo.CreateTransaction(ctx, req)
}

func (s *TransactionRepositoryTestSuite) TestCreateTransaction() {
	ctx := context.Background()
	req := &requests.CreateTransactionRequest{
		CardNumber:      s.customerCardNumber,
		Amount:          100000,
		MerchantID:      &s.merchantID,
		PaymentMethod:   "visa",
		TransactionTime: time.Now(),
	}

	res, err := s.commandRepo.CreateTransaction(ctx, req)
	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(int32(req.Amount), res.Amount)
}

func (s *TransactionRepositoryTestSuite) TestFindAllTransactions() {
	_, err := s.createSeedTransaction()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.queryRepo.FindAllTransactions(ctx, &requests.FindAllTransactions{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TransactionRepositoryTestSuite) TestFindById() {
	transaction, err := s.createSeedTransaction()
	s.Require().NoError(err)
	ctx := context.Background()

	found, err := s.queryRepo.FindById(ctx, int(transaction.TransactionID))
	s.NoError(err)
	s.NotNil(found)
	s.Equal(transaction.TransactionID, found.TransactionID)
}

func (s *TransactionRepositoryTestSuite) TestFindByActive() {
	_, err := s.createSeedTransaction()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.queryRepo.FindByActive(ctx, &requests.FindAllTransactions{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TransactionRepositoryTestSuite) TestFindByTrashed() {
	transaction, err := s.createSeedTransaction()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.commandRepo.TrashedTransaction(ctx, int(transaction.TransactionID))
	s.Require().NoError(err)

	res, err := s.queryRepo.FindByTrashed(ctx, &requests.FindAllTransactions{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TransactionRepositoryTestSuite) TestFindAllTransactionByCardNumber() {
	_, err := s.createSeedTransaction()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.queryRepo.FindAllTransactionByCardNumber(ctx, &requests.FindAllTransactionCardNumber{
		CardNumber: s.customerCardNumber,
		Page:       1,
		PageSize:   10,
		Search:     "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TransactionRepositoryTestSuite) TestFindTransactionByMerchantId() {
	_, err := s.createSeedTransaction()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.queryRepo.FindTransactionByMerchantId(ctx, s.merchantID)
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TransactionRepositoryTestSuite) TestUpdateTransaction() {
	transaction, err := s.createSeedTransaction()
	s.Require().NoError(err)
	ctx := context.Background()

	id := int(transaction.TransactionID)
	req := &requests.UpdateTransactionRequest{
		TransactionID:   &id,
		CardNumber:      s.customerCardNumber,
		Amount:          200000,
		MerchantID:      &s.merchantID,
		PaymentMethod:   "visa",
		TransactionTime: time.Now(),
	}
	res, err := s.commandRepo.UpdateTransaction(ctx, req)
	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal(int32(200000), res.Amount)
}

func (s *TransactionRepositoryTestSuite) TestUpdateTransactionStatus() {
	transaction, err := s.createSeedTransaction()
	s.Require().NoError(err)
	ctx := context.Background()

	req := &requests.UpdateTransactionStatus{
		TransactionID: int(transaction.TransactionID),
		Status:        "success",
	}
	res, err := s.commandRepo.UpdateTransactionStatus(ctx, req)
	s.NoError(err)
	s.Require().NotNil(res)
	s.Equal("success", res.Status)
}

func (s *TransactionRepositoryTestSuite) TestTrashTransaction() {
	transaction, err := s.createSeedTransaction()
	s.Require().NoError(err)
	ctx := context.Background()

	trashed, err := s.commandRepo.TrashedTransaction(ctx, int(transaction.TransactionID))
	s.NoError(err)
	s.NotNil(trashed.DeletedAt.Valid)
}

func (s *TransactionRepositoryTestSuite) TestRestoreTransaction() {
	transaction, err := s.createSeedTransaction()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.commandRepo.TrashedTransaction(ctx, int(transaction.TransactionID))
	s.Require().NoError(err)

	restored, err := s.commandRepo.RestoreTransaction(ctx, int(transaction.TransactionID))
	s.NoError(err)
	s.False(restored.DeletedAt.Valid)
}

func (s *TransactionRepositoryTestSuite) TestDeleteTransactionPermanent() {
	transaction, err := s.createSeedTransaction()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.commandRepo.TrashedTransaction(ctx, int(transaction.TransactionID))
	s.Require().NoError(err)

	success, err := s.commandRepo.DeleteTransactionPermanent(ctx, int(transaction.TransactionID))
	s.NoError(err)
	s.True(success)

	_, err = s.queryRepo.FindById(ctx, int(transaction.TransactionID))
	s.Error(err)
}

func (s *TransactionRepositoryTestSuite) TestRestoreAllTransaction() {
	transaction, err := s.createSeedTransaction()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.commandRepo.TrashedTransaction(ctx, int(transaction.TransactionID))
	s.Require().NoError(err)

	success, err := s.commandRepo.RestoreAllTransaction(ctx)
	s.NoError(err)
	s.True(success)

	found, err := s.queryRepo.FindById(ctx, int(transaction.TransactionID))
	s.NoError(err)
	s.NotNil(found)
}

func (s *TransactionRepositoryTestSuite) TestDeleteAllTransactionPermanent() {
	transaction, err := s.createSeedTransaction()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.commandRepo.TrashedTransaction(ctx, int(transaction.TransactionID))
	s.Require().NoError(err)

	success, err := s.commandRepo.DeleteAllTransactionPermanent(ctx)
	s.NoError(err)
	s.True(success)

	_, err = s.queryRepo.FindById(ctx, int(transaction.TransactionID))
	s.Error(err)
}

func TestTransactionRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TransactionRepositoryTestSuite))
}
