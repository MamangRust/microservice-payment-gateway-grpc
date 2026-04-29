package merchant_test

import (
	"context"
	"testing"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	tests "github.com/MamangRust/microservice-payment-gateway-test"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/service"
	user_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type MerchantServiceTestSuite struct {
	suite.Suite
	ts              *tests.TestSuite
	dbPool          *pgxpool.Pool
	redisClient     redis.UniversalClient
	merchantService service.Service
	userRepo        user_repo.UserCommandRepository
	userID          int
	merchantID      int
}

func (s *MerchantServiceTestSuite) SetupSuite() {
	ts, err := tests.SetupTestSuite()
	s.Require().NoError(err)
	s.ts = ts

	s.Require().NoError(s.ts.RunMigrations("user", "role", "auth", "merchant"))

	pool, err := pgxpool.New(s.ts.Ctx, s.ts.DBURL)
	s.Require().NoError(err)
	s.dbPool = pool

	opts, err := redis.ParseURL(s.ts.RedisURL)
	s.Require().NoError(err)
	s.redisClient = redis.NewClient(opts)

	queries := db.New(pool)
	repos := repository.NewRepositories(queries, nil)
	s.userRepo = user_repo.NewUserCommandRepository(queries)

	logger.ResetInstance()
	lp := sdklog.NewLoggerProvider()
	log, _ := logger.NewLogger("test", lp)
	cacheMetrics, _ := observability.NewCacheMetrics("test")
	cacheStore := cache.NewCacheStore(s.redisClient, log, cacheMetrics)

	s.merchantService = service.NewService(&service.Deps{
		Kafka:        nil,
		Repositories: repos,
		UserAdapter:  s.ts.UserAdapter,
		Logger:       log,
		Cache:        cacheStore,
	})

	// Seed User
	user, err := s.userRepo.CreateUser(context.Background(), &requests.CreateUserRequest{
		FirstName: "Merchant",
		LastName:  "ServiceOwner",
		Email:     "merchant.service.owner@example.com",
		Password:  "password123",
	})
	s.Require().NoError(err)
	s.userID = int(user.UserID)
}

func (s *MerchantServiceTestSuite) TearDownSuite() {
	s.redisClient.Close()
	s.dbPool.Close()
	s.ts.Teardown()
}

func (s *MerchantServiceTestSuite) Test1_CreateMerchant() {
	ctx := context.Background()

	req := &requests.CreateMerchantRequest{
		Name:   "Service Merchant",
		UserID: s.userID,
	}
	merchant, err := s.merchantService.MerchantCommandService().CreateMerchant(ctx, req)
	s.NoError(err)
	s.NotNil(merchant)
	s.Equal(req.Name, merchant.Name)
	s.merchantID = int(merchant.MerchantID)
}

func (s *MerchantServiceTestSuite) Test2_FindMerchantById() {
	s.Require().NotZero(s.merchantID)
	ctx := context.Background()

	found, err := s.merchantService.MerchantQueryService().FindById(ctx, s.merchantID)
	s.NoError(err)
	s.NotNil(found)
	s.Equal(s.merchantID, int(found.MerchantID))
}

func (s *MerchantServiceTestSuite) Test3_UpdateMerchant() {
	s.Require().NotZero(s.merchantID)
	ctx := context.Background()

	updateReq := &requests.UpdateMerchantRequest{
		MerchantID: &s.merchantID,
		Name:       "Updated Service Merchant",
		UserID:     s.userID,
		Status:     "active",
	}
	updated, err := s.merchantService.MerchantCommandService().UpdateMerchant(ctx, updateReq)
	s.NoError(err)
	s.NotNil(updated)
	s.Equal(updateReq.Name, updated.Name)
}

func (s *MerchantServiceTestSuite) Test4_TrashAndRestore() {
	s.Require().NotZero(s.merchantID)
	ctx := context.Background()

	_, err := s.merchantService.MerchantCommandService().TrashedMerchant(ctx, s.merchantID)
	s.NoError(err)

	_, err = s.merchantService.MerchantCommandService().RestoreMerchant(ctx, s.merchantID)
	s.NoError(err)
}

func (s *MerchantServiceTestSuite) Test7_BulkOperations() {
	ctx := context.Background()

	// Restore All
	success, err := s.merchantService.MerchantCommandService().RestoreAllMerchant(ctx)
	s.NoError(err)
	s.True(success)

	// Delete All Permanent
	success, err = s.merchantService.MerchantCommandService().DeleteAllMerchantPermanent(ctx)
	s.NoError(err)
	s.True(success)
}

func (s *MerchantServiceTestSuite) Test5_DeletePermanent() {
	s.Require().NotZero(s.merchantID)
	ctx := context.Background()

	success, err := s.merchantService.MerchantCommandService().DeleteMerchantPermanent(ctx, s.merchantID)
	s.NoError(err)
	s.True(success)
}

func TestMerchantServiceSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(MerchantServiceTestSuite))
}
