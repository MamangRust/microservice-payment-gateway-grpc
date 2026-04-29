package transfer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	tests "github.com/MamangRust/microservice-payment-gateway-test"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
)

type TransferRepositoryTestSuite struct {
	suite.Suite
	ts     *tests.TestSuite
	dbPool *pgxpool.Pool
	commandRepo   repository.TransferCommandRepository
	queryRepo     repository.TransferQueryRepository
	userRepo    user_repo.UserCommandRepository
	cardRepo    card_repo.Repositories
	saldoRepo   saldo_repo.Repositories

	senderCardNumber   string
	receiverCardNumber string
}

func (s *TransferRepositoryTestSuite) SetupSuite() {
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

	s.commandRepo = repository.NewTransferCommandRepository(queries)
	s.queryRepo = repository.NewTransferQueryRepository(queries)

	// Seed Sender
	ctx := context.Background()
	sender, err := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Sender",
		LastName:  "Repo",
		Email:     fmt.Sprintf("sender.repo-%d@test.com", time.Now().UnixNano()),
		Password:  "password123",
	})
	s.Require().NoError(err)

	sCard, err := s.cardRepo.CardCommand.CreateCard(ctx, &requests.CreateCardRequest{
		UserID:       int(sender.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(1, 0, 0),
		CVV:          "111",
		CardProvider: "visa",
	})
	s.Require().NoError(err)
	s.senderCardNumber = sCard.CardNumber

	// Seed Receiver
	receiver, err := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Receiver",
		LastName:  "Repo",
		Email:     fmt.Sprintf("receiver.repo-%d@test.com", time.Now().UnixNano()),
		Password:  "password123",
	})
	s.Require().NoError(err)

	rCard, err := s.cardRepo.CardCommand.CreateCard(ctx, &requests.CreateCardRequest{
		UserID:       int(receiver.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(1, 0, 0),
		CVV:          "222",
		CardProvider: "mastercard",
	})
	s.Require().NoError(err)
	s.receiverCardNumber = rCard.CardNumber
}

func (s *TransferRepositoryTestSuite) TearDownSuite() {
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	if s.ts != nil {
		s.ts.Teardown()
	}
}

func (s *TransferRepositoryTestSuite) createSeedTransfer() (*db.CreateTransferRow, error) {
	return s.commandRepo.CreateTransfer(context.Background(), &requests.CreateTransferRequest{
		TransferFrom:   s.senderCardNumber,
		TransferTo:     s.receiverCardNumber,
		TransferAmount: 25000,
	})
}

func (s *TransferRepositoryTestSuite) TestCreateTransfer() {
	ctx := context.Background()
	req := &requests.CreateTransferRequest{
		TransferFrom:   s.senderCardNumber,
		TransferTo:     s.receiverCardNumber,
		TransferAmount: 25000,
	}

	transfer, err := s.commandRepo.CreateTransfer(ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(transfer)
	s.Equal(int32(req.TransferAmount), transfer.TransferAmount)
}

func (s *TransferRepositoryTestSuite) TestFindAll() {
	_, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.queryRepo.FindAll(ctx, &requests.FindAllTransfers{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TransferRepositoryTestSuite) TestFindById() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.queryRepo.FindById(ctx, int(transfer.TransferID))
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(transfer.TransferID, res.TransferID)
}

func (s *TransferRepositoryTestSuite) TestFindByActive() {
	_, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.queryRepo.FindByActive(ctx, &requests.FindAllTransfers{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TransferRepositoryTestSuite) TestFindByTrashed() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.commandRepo.TrashedTransfer(ctx, int(transfer.TransferID))
	s.Require().NoError(err)

	res, err := s.queryRepo.FindByTrashed(ctx, &requests.FindAllTransfers{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TransferRepositoryTestSuite) TestFindTransferByTransferFrom() {
	_, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.queryRepo.FindTransferByTransferFrom(ctx, s.senderCardNumber)
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TransferRepositoryTestSuite) TestFindTransferByTransferTo() {
	_, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.queryRepo.FindTransferByTransferTo(ctx, s.receiverCardNumber)
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TransferRepositoryTestSuite) TestUpdateTransfer() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	id := int(transfer.TransferID)
	req := &requests.UpdateTransferRequest{
		TransferID:     &id,
		TransferFrom:   s.senderCardNumber,
		TransferTo:     s.receiverCardNumber,
		TransferAmount: 50000,
	}

	res, err := s.commandRepo.UpdateTransfer(ctx, req)
	s.NoError(err)
	s.NotNil(res)
	s.Equal(int32(50000), res.TransferAmount)
}

func (s *TransferRepositoryTestSuite) TestUpdateTransferAmount() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	req := &requests.UpdateTransferAmountRequest{
		TransferID:     int(transfer.TransferID),
		TransferAmount: 60000,
	}

	res, err := s.commandRepo.UpdateTransferAmount(ctx, req)
	s.NoError(err)
	s.NotNil(res)
	s.Equal(int32(60000), res.TransferAmount)
}

func (s *TransferRepositoryTestSuite) TestUpdateTransferStatus() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	req := &requests.UpdateTransferStatus{
		TransferID: int(transfer.TransferID),
		Status:     "success",
	}

	res, err := s.commandRepo.UpdateTransferStatus(ctx, req)
	s.NoError(err)
	s.NotNil(res)
	s.Equal("success", res.Status)
}

func (s *TransferRepositoryTestSuite) TestTrashTransfer() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	trashed, err := s.commandRepo.TrashedTransfer(ctx, int(transfer.TransferID))
	s.NoError(err)
	s.NotNil(trashed)
	s.True(trashed.DeletedAt.Valid)
}

func (s *TransferRepositoryTestSuite) TestRestoreTransfer() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.commandRepo.TrashedTransfer(ctx, int(transfer.TransferID))
	s.Require().NoError(err)

	restored, err := s.commandRepo.RestoreTransfer(ctx, int(transfer.TransferID))
	s.NoError(err)
	s.NotNil(restored)
	s.False(restored.DeletedAt.Valid)
}

func (s *TransferRepositoryTestSuite) TestDeleteTransferPermanent() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.commandRepo.TrashedTransfer(ctx, int(transfer.TransferID))
	s.Require().NoError(err)

	success, err := s.commandRepo.DeleteTransferPermanent(ctx, int(transfer.TransferID))
	s.NoError(err)
	s.True(success)

	_, err = s.queryRepo.FindById(ctx, int(transfer.TransferID))
	s.Error(err)
}

func (s *TransferRepositoryTestSuite) TestRestoreAllTransfer() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.commandRepo.TrashedTransfer(ctx, int(transfer.TransferID))
	s.Require().NoError(err)

	success, err := s.commandRepo.RestoreAllTransfer(ctx)
	s.NoError(err)
	s.True(success)

	found, err := s.queryRepo.FindById(ctx, int(transfer.TransferID))
	s.NoError(err)
	s.NotNil(found)
}

func (s *TransferRepositoryTestSuite) TestDeleteAllTransferPermanent() {
	transfer, err := s.createSeedTransfer()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.commandRepo.TrashedTransfer(ctx, int(transfer.TransferID))
	s.Require().NoError(err)

	success, err := s.commandRepo.DeleteAllTransferPermanent(ctx)
	s.NoError(err)
	s.True(success)

	_, err = s.queryRepo.FindById(ctx, int(transfer.TransferID))
	s.Error(err)
}

func TestTransferRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TransferRepositoryTestSuite))
}
