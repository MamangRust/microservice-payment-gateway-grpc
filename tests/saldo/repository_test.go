package saldo_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	tests "github.com/MamangRust/microservice-payment-gateway-test"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
)

type SaldoRepositoryTestSuite struct {
	suite.Suite
	ts     *tests.TestSuite
	dbPool *pgxpool.Pool
	repo   saldo_repo.Repositories
	
	userRepo  user_repo.UserCommandRepository
	cardRepo  card_repo.CardCommandRepository

	cardNumber string
}

func (s *SaldoRepositoryTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "saldo"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	queries := db.New(pool)
	s.repo = saldo_repo.NewRepositories(queries, nil)

	userRepos := user_repo.NewRepositories(queries)
	cardRepos := card_repo.NewRepositories(queries, nil)
	s.userRepo = userRepos.UserCommand()
	s.cardRepo = cardRepos.CardCommand

	// Seed User and Card
	ctx := context.Background()
	user, err := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Saldo",
		LastName:  "Owner",
		Email:     fmt.Sprintf("saldo.repo-%d@example.com", time.Now().UnixNano()),
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
}

func (s *SaldoRepositoryTestSuite) TearDownSuite() {
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	if s.ts != nil {
		s.ts.Teardown()
	}
}

func (s *SaldoRepositoryTestSuite) createSeedSaldo() (*db.CreateSaldoRow, error) {
	return s.repo.CreateSaldo(context.Background(), &requests.CreateSaldoRequest{
		CardNumber:   s.cardNumber,
		TotalBalance: 1000000,
	})
}

func (s *SaldoRepositoryTestSuite) TestCreateSaldo() {
	ctx := context.Background()
	req := &requests.CreateSaldoRequest{
		CardNumber:   s.cardNumber,
		TotalBalance: 1000000,
	}

	saldo, err := s.repo.CreateSaldo(ctx, req)
	s.NoError(err)
	s.Require().NotNil(saldo)
	s.Equal(int32(req.TotalBalance), saldo.TotalBalance)
}

func (s *SaldoRepositoryTestSuite) TestFindAllSaldos() {
	_, err := s.createSeedSaldo()
	// Ignore error because it might exist already for this card
	ctx := context.Background()

	res, err := s.repo.FindAllSaldos(ctx, &requests.FindAllSaldos{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *SaldoRepositoryTestSuite) TestFindById() {
	saldo, err := s.createSeedSaldo()
	if err != nil {
		// If already exists, find it
		found, err2 := s.repo.FindByCardNumber(context.Background(), s.cardNumber)
		s.Require().NoError(err2)
		ctx := context.Background()
		res, err3 := s.repo.FindById(ctx, int(found.SaldoID))
		s.NoError(err3)
		s.NotNil(res)
		return
	}
	ctx := context.Background()
	found, err := s.repo.FindById(ctx, int(saldo.SaldoID))
	s.NoError(err)
	s.NotNil(found)
	s.Equal(saldo.SaldoID, found.SaldoID)
}

func (s *SaldoRepositoryTestSuite) TestFindByCardNumber() {
	s.Require().NotEmpty(s.cardNumber)
	ctx := context.Background()

	found, err := s.repo.FindByCardNumber(ctx, s.cardNumber)
	s.NoError(err)
	s.NotNil(found)
}

func (s *SaldoRepositoryTestSuite) TestFindByActive() {
	ctx := context.Background()

	res, err := s.repo.FindByActive(ctx, &requests.FindAllSaldos{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *SaldoRepositoryTestSuite) TestFindByTrashed() {
	found, err := s.repo.FindByCardNumber(context.Background(), s.cardNumber)
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedSaldo(ctx, int(found.SaldoID))
	s.Require().NoError(err)

	res, err := s.repo.FindByTrashed(ctx, &requests.FindAllSaldos{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)

	// Restore
	_, err = s.repo.RestoreSaldo(ctx, int(found.SaldoID))
	s.Require().NoError(err)
}

func (s *SaldoRepositoryTestSuite) TestUpdateBalance() {
	s.Require().NotEmpty(s.cardNumber)
	ctx := context.Background()

	req := &requests.UpdateSaldoBalance{
		CardNumber:   s.cardNumber,
		TotalBalance: 1200000,
	}
	updated, err := s.repo.UpdateSaldoBalance(ctx, req)
	s.NoError(err)
	s.NotNil(updated)
	s.Equal(int32(1200000), updated.TotalBalance)
}

func (s *SaldoRepositoryTestSuite) TestTrashSaldo() {
	_, _ = s.createSeedSaldo()
	found, err := s.repo.FindByCardNumber(context.Background(), s.cardNumber)
	s.Require().NoError(err)
	ctx := context.Background()

	trashed, err := s.repo.TrashedSaldo(ctx, int(found.SaldoID))
	s.NoError(err)
	s.NotNil(trashed)
	s.True(trashed.DeletedAt.Valid)
}

func (s *SaldoRepositoryTestSuite) TestRestoreSaldo() {
	_, _ = s.createSeedSaldo()
	found, err := s.repo.FindByCardNumber(context.Background(), s.cardNumber)
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedSaldo(ctx, int(found.SaldoID))
	s.Require().NoError(err)

	restored, err := s.repo.RestoreSaldo(ctx, int(found.SaldoID))
	s.NoError(err)
	s.NotNil(restored)
	s.False(restored.DeletedAt.Valid)
}

func (s *SaldoRepositoryTestSuite) TestDeleteSaldoPermanent() {
	// Need a new card for permanent delete to not break other tests if they run later
	ctx := context.Background()
	user, _ := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Saldo",
		LastName:  "Owner",
		Email:     fmt.Sprintf("saldo.repo.del-%d@example.com", time.Now().UnixNano()),
		Password:  "password123",
	})
	card, _ := s.cardRepo.CreateCard(ctx, &requests.CreateCardRequest{
		UserID:       int(user.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(2, 0, 0),
		CVV:          "123",
		CardProvider: "visa",
	})
	saldo, err := s.repo.CreateSaldo(ctx, &requests.CreateSaldoRequest{
		CardNumber:   card.CardNumber,
		TotalBalance: 1000000,
	})
	s.Require().NoError(err)

	_, err = s.repo.TrashedSaldo(ctx, int(saldo.SaldoID))
	s.Require().NoError(err)

	success, err := s.repo.DeleteSaldoPermanent(ctx, int(saldo.SaldoID))
	s.NoError(err)
	s.True(success)
}

func (s *SaldoRepositoryTestSuite) TestRestoreAllSaldo() {
	// Ensure at least one trashed saldo exists
	ctx := context.Background()
	user, _ := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Saldo",
		LastName:  "Owner",
		Email:     fmt.Sprintf("saldo.repo.res-%d@example.com", time.Now().UnixNano()),
		Password:  "password123",
	})
	card, _ := s.cardRepo.CreateCard(ctx, &requests.CreateCardRequest{
		UserID:       int(user.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(2, 0, 0),
		CVV:          "123",
		CardProvider: "visa",
	})
	saldo, err := s.repo.CreateSaldo(ctx, &requests.CreateSaldoRequest{
		CardNumber:   card.CardNumber,
		TotalBalance: 1000000,
	})
	s.Require().NoError(err)

	_, err = s.repo.TrashedSaldo(ctx, int(saldo.SaldoID))
	s.Require().NoError(err)

	success, err := s.repo.RestoreAllSaldo(ctx)
	s.NoError(err)
	s.True(success)
}

func (s *SaldoRepositoryTestSuite) TestDeleteAllSaldoPermanent() {
	// Ensure at least one trashed saldo exists
	ctx := context.Background()
	user, _ := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Saldo",
		LastName:  "Owner",
		Email:     fmt.Sprintf("saldo.repo.delall-%d@example.com", time.Now().UnixNano()),
		Password:  "password123",
	})
	card, _ := s.cardRepo.CreateCard(ctx, &requests.CreateCardRequest{
		UserID:       int(user.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(2, 0, 0),
		CVV:          "123",
		CardProvider: "visa",
	})
	saldo, err := s.repo.CreateSaldo(ctx, &requests.CreateSaldoRequest{
		CardNumber:   card.CardNumber,
		TotalBalance: 1000000,
	})
	s.Require().NoError(err)

	_, err = s.repo.TrashedSaldo(ctx, int(saldo.SaldoID))
	s.Require().NoError(err)

	success, err := s.repo.DeleteAllSaldoPermanent(ctx)
	s.NoError(err)
	s.True(success)
}

func TestSaldoRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(SaldoRepositoryTestSuite))
}
