package withdraw_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-test"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
)

type WithdrawRepositoryTestSuite struct {
	suite.Suite
	ts        *tests.TestSuite
	dbPool    *pgxpool.Pool
	repo      repository.WithdrawCommandRepository
	queryRepo repository.WithdrawQueryRepository
}

func (s *WithdrawRepositoryTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "saldo", "withdraw"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	queries := db.New(pool)
	s.repo = repository.NewWithdrawCommandRepository(queries)
	s.queryRepo = repository.NewWithdrawQueryRepository(queries)
}

func (s *WithdrawRepositoryTestSuite) TearDownSuite() {
	s.dbPool.Close()
	s.ts.Teardown()
}

func (s *WithdrawRepositoryTestSuite) createSeedCard() (*db.CreateCardRow, error) {
	userReq := &requests.CreateUserRequest{
		FirstName: "Withdraw",
		LastName:  "Owner",
		Email:     fmt.Sprintf("withdrawowner-%d@example.com", time.Now().UnixNano()),
		Password:  "password123",
	}
	queries := db.New(s.dbPool)
	userRepo := user_repo.NewUserCommandRepository(queries)
	user, err := userRepo.CreateUser(context.Background(), userReq)
	if err != nil {
		return nil, err
	}

	cardRepos := card_repo.NewRepositories(queries, nil)
	return cardRepos.CardCommand.CreateCard(context.Background(), &requests.CreateCardRequest{
		UserID:       int(user.UserID),
		CardType:     "debit",
		ExpireDate:   time.Now().AddDate(2, 0, 0),
		CVV:          "123",
		CardProvider: "visa",
	})
}

func (s *WithdrawRepositoryTestSuite) TestCreateWithdraw() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)

	req := &requests.CreateWithdrawRequest{
		CardNumber:     card.CardNumber,
		WithdrawAmount: 50000,
		WithdrawTime:   time.Now(),
	}

	withdraw, err := s.repo.CreateWithdraw(context.Background(), req)
	s.NoError(err)
	s.NotNil(withdraw)
	s.Equal(int32(req.WithdrawAmount), withdraw.WithdrawAmount)
}

func (s *WithdrawRepositoryTestSuite) TestFindAll() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)

	_, err = s.repo.CreateWithdraw(context.Background(), &requests.CreateWithdrawRequest{
		CardNumber:     card.CardNumber,
		WithdrawAmount: 50000,
		WithdrawTime:   time.Now(),
	})
	s.Require().NoError(err)

	res, err := s.queryRepo.FindAll(context.Background(), &requests.FindAllWithdraws{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *WithdrawRepositoryTestSuite) TestFindById() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)

	withdraw, err := s.repo.CreateWithdraw(context.Background(), &requests.CreateWithdrawRequest{
		CardNumber:     card.CardNumber,
		WithdrawAmount: 50000,
		WithdrawTime:   time.Now(),
	})
	s.Require().NoError(err)

	res, err := s.queryRepo.FindById(context.Background(), int(withdraw.WithdrawID))
	s.NoError(err)
	s.NotNil(res)
	s.Equal(withdraw.WithdrawID, res.WithdrawID)
}

func (s *WithdrawRepositoryTestSuite) TestFindByActive() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)

	_, err = s.repo.CreateWithdraw(context.Background(), &requests.CreateWithdrawRequest{
		CardNumber:     card.CardNumber,
		WithdrawAmount: 50000,
		WithdrawTime:   time.Now(),
	})
	s.Require().NoError(err)

	res, err := s.queryRepo.FindByActive(context.Background(), &requests.FindAllWithdraws{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *WithdrawRepositoryTestSuite) TestFindByTrashed() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)

	withdraw, err := s.repo.CreateWithdraw(context.Background(), &requests.CreateWithdrawRequest{
		CardNumber:     card.CardNumber,
		WithdrawAmount: 50000,
		WithdrawTime:   time.Now(),
	})
	s.Require().NoError(err)

	_, err = s.repo.TrashedWithdraw(context.Background(), int(withdraw.WithdrawID))
	s.Require().NoError(err)

	res, err := s.queryRepo.FindByTrashed(context.Background(), &requests.FindAllWithdraws{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *WithdrawRepositoryTestSuite) TestUpdateWithdraw() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)

	withdraw, err := s.repo.CreateWithdraw(context.Background(), &requests.CreateWithdrawRequest{
		CardNumber:     card.CardNumber,
		WithdrawAmount: 50000,
		WithdrawTime:   time.Now(),
	})
	s.Require().NoError(err)

	updateAmount := 60000
	id := int(withdraw.WithdrawID)
	req := &requests.UpdateWithdrawRequest{
		WithdrawID:     &id,
		CardNumber:     card.CardNumber,
		WithdrawAmount: updateAmount,
		WithdrawTime:   time.Now(),
	}

	updated, err := s.repo.UpdateWithdraw(context.Background(), req)
	s.NoError(err)
	s.NotNil(updated)
	s.Equal(int32(updateAmount), updated.WithdrawAmount)
}

func (s *WithdrawRepositoryTestSuite) TestTrashWithdraw() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)

	withdraw, err := s.repo.CreateWithdraw(context.Background(), &requests.CreateWithdrawRequest{
		CardNumber:     card.CardNumber,
		WithdrawAmount: 50000,
		WithdrawTime:   time.Now(),
	})
	s.Require().NoError(err)

	trashed, err := s.repo.TrashedWithdraw(context.Background(), int(withdraw.WithdrawID))
	s.NoError(err)
	s.NotNil(trashed)
	s.True(trashed.DeletedAt.Valid)
}

func (s *WithdrawRepositoryTestSuite) TestRestoreWithdraw() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)

	withdraw, err := s.repo.CreateWithdraw(context.Background(), &requests.CreateWithdrawRequest{
		CardNumber:     card.CardNumber,
		WithdrawAmount: 50000,
		WithdrawTime:   time.Now(),
	})
	s.Require().NoError(err)

	_, err = s.repo.TrashedWithdraw(context.Background(), int(withdraw.WithdrawID))
	s.Require().NoError(err)

	restored, err := s.repo.RestoreWithdraw(context.Background(), int(withdraw.WithdrawID))
	s.NoError(err)
	s.NotNil(restored)
	s.False(restored.DeletedAt.Valid)
}

func (s *WithdrawRepositoryTestSuite) TestDeleteWithdrawPermanent() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)

	withdraw, err := s.repo.CreateWithdraw(context.Background(), &requests.CreateWithdrawRequest{
		CardNumber:     card.CardNumber,
		WithdrawAmount: 50000,
		WithdrawTime:   time.Now(),
	})
	s.Require().NoError(err)

	_, err = s.repo.TrashedWithdraw(context.Background(), int(withdraw.WithdrawID))
	s.Require().NoError(err)

	deleted, err := s.repo.DeleteWithdrawPermanent(context.Background(), int(withdraw.WithdrawID))
	s.NoError(err)
	s.True(deleted)

	_, err = s.queryRepo.FindById(context.Background(), int(withdraw.WithdrawID))
	s.Error(err)
}

func (s *WithdrawRepositoryTestSuite) TestRestoreAllWithdraw() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)

	withdraw, err := s.repo.CreateWithdraw(context.Background(), &requests.CreateWithdrawRequest{
		CardNumber:     card.CardNumber,
		WithdrawAmount: 50000,
		WithdrawTime:   time.Now(),
	})
	s.Require().NoError(err)

	_, err = s.repo.TrashedWithdraw(context.Background(), int(withdraw.WithdrawID))
	s.Require().NoError(err)

	restored, err := s.repo.RestoreAllWithdraw(context.Background())
	s.NoError(err)
	s.True(restored)

	res, err := s.queryRepo.FindById(context.Background(), int(withdraw.WithdrawID))
	s.NoError(err)
	s.NotNil(res)
}

func (s *WithdrawRepositoryTestSuite) TestDeleteAllWithdrawPermanent() {
	card, err := s.createSeedCard()
	s.Require().NoError(err)

	withdraw, err := s.repo.CreateWithdraw(context.Background(), &requests.CreateWithdrawRequest{
		CardNumber:     card.CardNumber,
		WithdrawAmount: 50000,
		WithdrawTime:   time.Now(),
	})
	s.Require().NoError(err)

	_, err = s.repo.TrashedWithdraw(context.Background(), int(withdraw.WithdrawID))
	s.Require().NoError(err)

	deleted, err := s.repo.DeleteAllWithdrawPermanent(context.Background())
	s.NoError(err)
	s.True(deleted)

	_, err = s.queryRepo.FindById(context.Background(), int(withdraw.WithdrawID))
	s.Error(err)
}

func TestWithdrawRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(WithdrawRepositoryTestSuite))
}
