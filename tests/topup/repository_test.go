package topup_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	tests "github.com/MamangRust/microservice-payment-gateway-test"
	topup_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/repository"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	saldo_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
)

type TopupRepositoryTestSuite struct {
	suite.Suite
	ts     *tests.TestSuite
	dbPool *pgxpool.Pool
	repo   topup_repo.Repositories
	
	userRepo  user_repo.UserCommandRepository
	cardRepo  card_repo.CardCommandRepository
	saldoRepo saldo_repo.Repositories

	cardNumber string
}

func (s *TopupRepositoryTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "card", "saldo", "topup"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

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
	s.repo = topup_repo.NewRepositories(queries, cardAdapter, saldoRepos)
	s.userRepo = userRepos.UserCommand()
	s.cardRepo = cardRepos.CardCommand
	s.saldoRepo = saldoRepos

	// Seed User and Card
	ctx := context.Background()
	user, err := s.userRepo.CreateUser(ctx, &requests.CreateUserRequest{
		FirstName: "Topup",
		LastName:  "Owner",
		Email:     fmt.Sprintf("topup.repo-%d@example.com", time.Now().UnixNano()),
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

func (s *TopupRepositoryTestSuite) TearDownSuite() {
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	if s.ts != nil {
		s.ts.Teardown()
	}
}

func (s *TopupRepositoryTestSuite) createSeedTopup() (*db.CreateTopupRow, error) {
	return s.repo.CreateTopup(context.Background(), &requests.CreateTopupRequest{
		CardNumber:  s.cardNumber,
		TopupAmount: 50000,
		TopupMethod: "visa",
	})
}

func (s *TopupRepositoryTestSuite) TestCreateTopup() {
	ctx := context.Background()
	req := &requests.CreateTopupRequest{
		CardNumber:  s.cardNumber,
		TopupAmount: 50000,
		TopupMethod: "visa",
	}

	topup, err := s.repo.CreateTopup(ctx, req)
	s.NoError(err)
	s.NotNil(topup)
	s.Equal(int32(req.TopupAmount), topup.TopupAmount)
}

func (s *TopupRepositoryTestSuite) TestFindAllTopups() {
	_, err := s.createSeedTopup()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.repo.FindAllTopups(ctx, &requests.FindAllTopups{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TopupRepositoryTestSuite) TestFindById() {
	topup, err := s.createSeedTopup()
	s.Require().NoError(err)
	ctx := context.Background()

	found, err := s.repo.FindById(ctx, int(topup.TopupID))
	s.NoError(err)
	s.NotNil(found)
	s.Equal(topup.TopupID, found.TopupID)
}

func (s *TopupRepositoryTestSuite) TestFindAllTopupByCardNumber() {
	_, err := s.createSeedTopup()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.repo.FindAllTopupByCardNumber(ctx, &requests.FindAllTopupsByCardNumber{
		CardNumber: s.cardNumber,
		PageSize:   10,
		Page:       1,
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TopupRepositoryTestSuite) TestFindByActive() {
	_, err := s.createSeedTopup()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.repo.FindByActive(ctx, &requests.FindAllTopups{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *TopupRepositoryTestSuite) TestFindByTrashed() {
	topup, err := s.createSeedTopup()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedTopup(ctx, int(topup.TopupID))
	s.Require().NoError(err)

	res, err := s.repo.FindByTrashed(ctx, &requests.FindAllTopups{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)

	// Restore
	_, err = s.repo.RestoreTopup(ctx, int(topup.TopupID))
	s.Require().NoError(err)
}

func (s *TopupRepositoryTestSuite) TestUpdateTopup() {
	topup, err := s.createSeedTopup()
	s.Require().NoError(err)
	ctx := context.Background()

	id := int(topup.TopupID)
	req := &requests.UpdateTopupRequest{
		TopupID:     &id,
		CardNumber:  s.cardNumber,
		TopupAmount: 75000,
		TopupMethod: "visa",
	}
	updated, err := s.repo.UpdateTopup(ctx, req)
	s.NoError(err)
	s.NotNil(updated)
	s.Equal(int32(75000), updated.TopupAmount)
}

func (s *TopupRepositoryTestSuite) TestUpdateStatus() {
	topup, err := s.createSeedTopup()
	s.Require().NoError(err)
	ctx := context.Background()

	req := &requests.UpdateTopupStatus{
		TopupID: int(topup.TopupID),
		Status:  "success",
	}
	updated, err := s.repo.UpdateTopupStatus(ctx, req)
	s.NoError(err)
	s.NotNil(updated)
	s.Equal("success", updated.Status)
}

func (s *TopupRepositoryTestSuite) TestTrashTopup() {
	topup, err := s.createSeedTopup()
	s.Require().NoError(err)
	ctx := context.Background()

	trashed, err := s.repo.TrashedTopup(ctx, int(topup.TopupID))
	s.NoError(err)
	s.NotNil(trashed)
	s.True(trashed.DeletedAt.Valid)
}

func (s *TopupRepositoryTestSuite) TestRestoreTopup() {
	topup, err := s.createSeedTopup()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedTopup(ctx, int(topup.TopupID))
	s.Require().NoError(err)

	restored, err := s.repo.RestoreTopup(ctx, int(topup.TopupID))
	s.NoError(err)
	s.NotNil(restored)
	s.False(restored.DeletedAt.Valid)
}

func (s *TopupRepositoryTestSuite) TestDeleteTopupPermanent() {
	topup, err := s.createSeedTopup()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedTopup(ctx, int(topup.TopupID))
	s.Require().NoError(err)

	success, err := s.repo.DeleteTopupPermanent(ctx, int(topup.TopupID))
	s.NoError(err)
	s.True(success)

	_, err = s.repo.FindById(ctx, int(topup.TopupID))
	s.Error(err)
}

func (s *TopupRepositoryTestSuite) TestRestoreAllTopup() {
	topup, err := s.createSeedTopup()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedTopup(ctx, int(topup.TopupID))
	s.Require().NoError(err)

	success, err := s.repo.RestoreAllTopup(ctx)
	s.NoError(err)
	s.True(success)

	found, err := s.repo.FindById(ctx, int(topup.TopupID))
	s.NoError(err)
	s.NotNil(found)
}

func (s *TopupRepositoryTestSuite) TestDeleteAllTopupPermanent() {
	topup, err := s.createSeedTopup()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedTopup(ctx, int(topup.TopupID))
	s.Require().NoError(err)

	success, err := s.repo.DeleteAllTopupPermanent(ctx)
	s.NoError(err)
	s.True(success)

	_, err = s.repo.FindById(ctx, int(topup.TopupID))
	s.Error(err)
}

func TestTopupRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(TopupRepositoryTestSuite))
}
