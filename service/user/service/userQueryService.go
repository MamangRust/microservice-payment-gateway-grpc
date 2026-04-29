package service

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errorhandler"
	user_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/user_errors/service"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	mencache "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/redis"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// userQueryDeps defines dependencies for userQueryService.
type userQueryDeps struct {
	Cache         mencache.UserQueryCache
	Repository    repository.UserQueryRepository
	Logger        logger.LoggerInterface
	Observability observability.TraceLoggerObservability
}

// userQueryService implements user query operations.
type userQueryService struct {
	cache               mencache.UserQueryCache
	userQueryRepository repository.UserQueryRepository
	logger              logger.LoggerInterface
	observability       observability.TraceLoggerObservability
}

// NewUserQueryService creates a new UserQueryService.
func NewUserQueryService(
	params *userQueryDeps,
) UserQueryService {
	return &userQueryService{
		cache:               params.Cache,
		userQueryRepository: params.Repository,
		logger:              params.Logger,
		observability:       params.Observability,
	}
}

func (s *userQueryService) FindAll(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetUsersWithPaginationRow, *int, error) {
	const method = "FindAll"

	page, pageSize := s.normalizePagination(req.Page, req.PageSize)
	search := req.Search

	if page <= 0 {
		page = 1
	}

	if pageSize <= 0 {
		pageSize = 10
	}

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("page", page),
		attribute.Int("pageSize", pageSize),
		attribute.String("search", search))

	defer func() {
		end(status)
	}()

	if data, total, found := s.cache.GetCachedUsersCache(ctx, req); found {
		logSuccess("Successfully retrieved all user records from cache", zap.Int("totalRecords", *total), zap.Int("page", page), zap.Int("pageSize", pageSize))

		return data, total, nil
	}

	users, err := s.userQueryRepository.FindAllUsers(ctx, req)
	if err != nil {
		status = "error"
		return errorhandler.HandlerErrorPagination[[]*db.GetUsersWithPaginationRow](
			s.logger,
			user_errors.ErrFailedFindAll,
			method,
			span,

			zap.Int("page", page),
			zap.Int("pageSize", pageSize),
			zap.String("search", search),
		)
	}

	var totalCount int

	if len(users) > 0 {
		totalCount = int(users[0].TotalCount)
	} else {
		totalCount = 0
	}

	s.cache.SetCachedUsersCache(ctx, req, users, &totalCount)

	logSuccess("Successfully fetched user",
		zap.Int("totalRecords", totalCount),
		zap.Int("page", page),
		zap.Int("pageSize", pageSize))

	return users, &totalCount, nil
}

func (s *userQueryService) FindByID(ctx context.Context, id int) (*db.GetUserByIDRow, error) {
	const method = "FindByID"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("user_id", id))

	defer func() {
		end(status)
	}()

	if data, found := s.cache.GetCachedUserCache(ctx, id); found {
		logSuccess("Successfully retrieved user record from cache", zap.Int("user.id", id))
		return data, nil
	}

	user, err := s.userQueryRepository.FindById(ctx, id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.GetUserByIDRow](
			s.logger,
			user_errors.ErrUserNotFoundRes,
			method,
			span,

			zap.Int("user_id", id),
		)
	}

	s.cache.SetCachedUserCache(ctx, user)

	logSuccess("Successfully fetched user", zap.Int("user_id", id))

	return user, nil
}

func (s *userQueryService) FindByActive(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetActiveUsersWithPaginationRow, *int, error) {
	const method = "FindByActive"

	page, pageSize := s.normalizePagination(req.Page, req.PageSize)
	search := req.Search

	if page <= 0 {
		page = 1
	}

	if pageSize <= 0 {
		pageSize = 10
	}

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("page", page),
		attribute.Int("pageSize", pageSize),
		attribute.String("search", search))

	defer func() {
		end(status)
	}()

	if data, total, found := s.cache.GetCachedUserActiveCache(ctx, req); found {
		logSuccess("Successfully retrieved active user records from cache", zap.Int("totalRecords", *total), zap.Int("page", page), zap.Int("pageSize", pageSize))
		return data, total, nil
	}

	users, err := s.userQueryRepository.FindByActive(ctx, req)
	if err != nil {
		status = "error"
		return errorhandler.HandlerErrorPagination[[]*db.GetActiveUsersWithPaginationRow](
			s.logger,
			user_errors.ErrFailedFindActive,
			method,
			span,

			zap.Int("page", page),
			zap.Int("pageSize", pageSize),
			zap.String("search", search),
		)
	}

	var totalCount int

	if len(users) > 0 {
		totalCount = int(users[0].TotalCount)
	} else {
		totalCount = 0
	}

	s.cache.SetCachedUserActiveCache(ctx, req, users, &totalCount)

	logSuccess("Successfully fetched active user",
		zap.Int("totalRecords", totalCount),
		zap.Int("page", page),
		zap.Int("pageSize", pageSize))

	return users, &totalCount, nil
}

func (s *userQueryService) FindByTrashed(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetTrashedUsersWithPaginationRow, *int, error) {
	const method = "FindByTrashed"

	page, pageSize := s.normalizePagination(req.Page, req.PageSize)
	search := req.Search

	if page <= 0 {
		page = 1
	}

	if pageSize <= 0 {
		pageSize = 10
	}

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("page", page),
		attribute.Int("pageSize", pageSize),
		attribute.String("search", search))

	defer func() {
		end(status)
	}()

	if data, total, found := s.cache.GetCachedUserTrashedCache(ctx, req); found {
		return data, total, nil
	}

	users, err := s.userQueryRepository.FindByTrashed(ctx, req)
	if err != nil {
		status = "error"
		return errorhandler.HandlerErrorPagination[[]*db.GetTrashedUsersWithPaginationRow](
			s.logger,
			user_errors.ErrFailedFindTrashed,
			method,
			span,

			zap.Int("page", page),
			zap.Int("pageSize", pageSize),
			zap.String("search", search),
		)
	}

	var totalCount int

	if len(users) > 0 {
		totalCount = int(users[0].TotalCount)
	} else {
		totalCount = 0
	}

	s.cache.SetCachedUserTrashedCache(ctx, req, users, &totalCount)

	logSuccess("Successfully fetched trashed user",
		zap.Int("totalRecords", totalCount),
		zap.Int("page", page),
		zap.Int("pageSize", pageSize))

	return users, &totalCount, nil
}

func (s *userQueryService) FindByEmail(ctx context.Context, email string) (*db.GetUserByEmailRow, error) {
	const method = "FindByEmail"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.String("email", email))

	defer func() {
		end(status)
	}()

	user, err := s.userQueryRepository.FindByEmail(ctx, email)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.GetUserByEmailRow](
			s.logger,
			user_errors.ErrUserNotFoundRes,
			method,
			span,

			zap.String("email", email),
		)
	}

	logSuccess("Successfully fetched user by email", zap.String("email", email))

	return user, nil
}

func (s *userQueryService) FindByVerificationCode(ctx context.Context, code string) (*db.GetUserByVerificationCodeRow, error) {
	const method = "FindByVerificationCode"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.String("verification_code", code))

	defer func() {
		end(status)
	}()

	user, err := s.userQueryRepository.FindByVerificationCode(ctx, code)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.GetUserByVerificationCodeRow](
			s.logger,
			user_errors.ErrUserNotFoundRes,
			method,
			span,

			zap.String("verification_code", code),
		)
	}

	logSuccess("Successfully fetched user by verification code", zap.String("verification_code", code))

	return user, nil
}

// normalizePagination validates and normalizes pagination parameters.
func (s *userQueryService) normalizePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return page, pageSize
}
