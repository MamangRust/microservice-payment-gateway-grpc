package merchant_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	tests "github.com/MamangRust/microservice-payment-gateway-test"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
)

type MerchantRepositoryTestSuite struct {
	suite.Suite
	ts         *tests.TestSuite
	dbPool     *pgxpool.Pool
	repo       repository.MerchantCommandRepository
	queryRepo  repository.MerchantQueryRepository
	userRepo   user_repo.UserCommandRepository
	userID     int
}

func (s *MerchantRepositoryTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "merchant"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	queries := db.New(pool)
	s.repo = repository.NewMerchantCommandRepository(queries)
	s.queryRepo = repository.NewMerchantQueryRepository(queries)
	s.userRepo = user_repo.NewUserCommandRepository(queries)

	// Seed User
	user, err := s.userRepo.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Merchant",
		LastName:  "Owner",
		Email:     fmt.Sprintf("merchant.owner-%d@example.com", time.Now().UnixNano()),
		Password:  "password123",
	})
	s.Require().NoError(err)
	s.userID = int(user.UserID)
}

func (s *MerchantRepositoryTestSuite) TearDownSuite() {
	s.dbPool.Close()
	s.ts.Teardown()
}

func (s *MerchantRepositoryTestSuite) createSeedMerchant() (*db.CreateMerchantRow, error) {
	return s.repo.CreateMerchant(context.Background(), &requests.CreateMerchantRequest{
		Name:   fmt.Sprintf("Test Merchant-%d", time.Now().UnixNano()),
		UserID: s.userID,
	})
}

func (s *MerchantRepositoryTestSuite) TestCreateMerchant() {
	req := &requests.CreateMerchantRequest{
		Name:   "Test Merchant",
		UserID: s.userID,
	}

	merchant, err := s.repo.CreateMerchant(context.Background(), req)
	s.NoError(err)
	s.NotNil(merchant)
	s.Equal(req.Name, merchant.Name)
	s.Equal(int32(s.userID), merchant.UserID)
}

func (s *MerchantRepositoryTestSuite) TestFindAllMerchants() {
	_, err := s.createSeedMerchant()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.queryRepo.FindAllMerchants(ctx, &requests.FindAllMerchants{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *MerchantRepositoryTestSuite) TestFindById() {
	merchant, err := s.createSeedMerchant()
	s.Require().NoError(err)
	ctx := context.Background()

	found, err := s.queryRepo.FindByMerchantId(ctx, int(merchant.MerchantID))
	s.NoError(err)
	s.NotNil(found)
	s.Equal(merchant.MerchantID, found.MerchantID)
}

func (s *MerchantRepositoryTestSuite) TestFindByActive() {
	_, err := s.createSeedMerchant()
	s.Require().NoError(err)
	ctx := context.Background()

	res, err := s.queryRepo.FindByActive(ctx, &requests.FindAllMerchants{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *MerchantRepositoryTestSuite) TestFindByTrashed() {
	merchant, err := s.createSeedMerchant()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedMerchant(ctx, int(merchant.MerchantID))
	s.Require().NoError(err)

	res, err := s.queryRepo.FindByTrashed(ctx, &requests.FindAllMerchants{
		Page:     1,
		PageSize: 10,
		Search:   "",
	})
	s.NoError(err)
	s.GreaterOrEqual(len(res), 1)
}

func (s *MerchantRepositoryTestSuite) TestUpdateMerchant() {
	merchant, err := s.createSeedMerchant()
	s.Require().NoError(err)
	ctx := context.Background()

	id := int(merchant.MerchantID)
	updateReq := &requests.UpdateMerchantRequest{
		MerchantID: &id,
		Name:       "Updated Merchant",
		UserID:     s.userID,
		Status:     "active",
	}

	updated, err := s.repo.UpdateMerchant(ctx, updateReq)
	s.NoError(err)
	s.NotNil(updated)
	s.Equal(updateReq.Name, updated.Name)
	s.Equal(updateReq.Status, updated.Status)
}

func (s *MerchantRepositoryTestSuite) TestTrashMerchant() {
	merchant, err := s.createSeedMerchant()
	s.Require().NoError(err)
	ctx := context.Background()

	trashed, err := s.repo.TrashedMerchant(ctx, int(merchant.MerchantID))
	s.NoError(err)
	s.NotNil(trashed)
	s.True(trashed.DeletedAt.Valid)
}

func (s *MerchantRepositoryTestSuite) TestRestoreMerchant() {
	merchant, err := s.createSeedMerchant()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedMerchant(ctx, int(merchant.MerchantID))
	s.Require().NoError(err)

	restored, err := s.repo.RestoreMerchant(ctx, int(merchant.MerchantID))
	s.NoError(err)
	s.NotNil(restored)
	s.False(restored.DeletedAt.Valid)
}

func (s *MerchantRepositoryTestSuite) TestDeleteMerchantPermanent() {
	merchant, err := s.createSeedMerchant()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedMerchant(ctx, int(merchant.MerchantID))
	s.Require().NoError(err)

	success, err := s.repo.DeleteMerchantPermanent(ctx, int(merchant.MerchantID))
	s.NoError(err)
	s.True(success)

	_, err = s.queryRepo.FindByMerchantId(ctx, int(merchant.MerchantID))
	s.Error(err)
}

func (s *MerchantRepositoryTestSuite) TestRestoreAllMerchant() {
	merchant, err := s.createSeedMerchant()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedMerchant(ctx, int(merchant.MerchantID))
	s.Require().NoError(err)

	success, err := s.repo.RestoreAllMerchant(ctx)
	s.NoError(err)
	s.True(success)

	found, err := s.queryRepo.FindByMerchantId(ctx, int(merchant.MerchantID))
	s.NoError(err)
	s.NotNil(found)
}

func (s *MerchantRepositoryTestSuite) TestDeleteAllMerchantPermanent() {
	merchant, err := s.createSeedMerchant()
	s.Require().NoError(err)
	ctx := context.Background()

	_, err = s.repo.TrashedMerchant(ctx, int(merchant.MerchantID))
	s.Require().NoError(err)

	success, err := s.repo.DeleteAllMerchantPermanent(ctx)
	s.NoError(err)
	s.True(success)

	_, err = s.queryRepo.FindByMerchantId(ctx, int(merchant.MerchantID))
	s.Error(err)
}

func TestMerchantRepositorySuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(MerchantRepositoryTestSuite))
}
