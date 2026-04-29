package service

import (
	"context"
	"errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/hash"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errorhandler"
	user_repo_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/user_errors/repository"
	user_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/user_errors/service"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"

	mencache "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/redis"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/user/repository"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// userCommandDeps defines dependencies for userCommandService.
type userCommandDeps struct {
	Cache                 mencache.UserCommandCache
	UserQueryRepository   repository.UserQueryRepository
	UserCommandRepository repository.UserCommandRepository
	RoleRepository        repository.RoleRepository
	Logger                logger.LoggerInterface
	Hashing               hash.HashPassword
	Observability         observability.TraceLoggerObservability
}

// userCommandService implements user command operations.
type userCommandService struct {
	cache                 mencache.UserCommandCache
	userQueryRepository   repository.UserQueryRepository
	userCommandRepository repository.UserCommandRepository
	roleRepository        repository.RoleRepository
	logger                logger.LoggerInterface
	hashing               hash.HashPassword
	observability         observability.TraceLoggerObservability
}

// NewUserCommandService creates a new UserCommandService.
func NewUserCommandService(
	params *userCommandDeps,
) UserCommandService {
	return &userCommandService{
		cache:                 params.Cache,
		userQueryRepository:   params.UserQueryRepository,
		userCommandRepository: params.UserCommandRepository,
		roleRepository:        params.RoleRepository,
		logger:                params.Logger,
		hashing:               params.Hashing,
		observability:         params.Observability,
	}
}

func (s *userCommandService) CreateUser(ctx context.Context, request *requests.CreateUserRequest) (*db.CreateUserRow, error) {
	const method = "CreateUser"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.String("email", request.Email))

	defer func() {
		end(status)
	}()

	s.logger.Debug("Creating new user", zap.String("email", request.Email), zap.Any("request", request))

	existingUser, err := s.userQueryRepository.FindByEmail(ctx, request.Email)
	if err != nil {
		if errors.Is(err, user_repo_errors.ErrUserNotFound) {
			s.logger.Debug("Email is available, proceeding to create user", zap.String("email", request.Email))
		} else {
			status = "error"
			return errorhandler.HandleError[*db.CreateUserRow](
				s.logger,
				err,
				method,
				span,
				zap.String("email", request.Email),
			)
		}
	} else if existingUser != nil {
		status = "error"
		return errorhandler.HandleError[*db.CreateUserRow](
			s.logger,
			user_errors.ErrUserEmailAlready,
			method,
			span,
			zap.String("email", request.Email),
		)
	}

	hash, err := s.hashing.HashPassword(request.Password)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.CreateUserRow](
			s.logger,
			user_errors.ErrUserPassword,
			method,
			span,
		)
	}

	request.Password = hash

	res, err := s.userCommandRepository.CreateUser(ctx, request)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.CreateUserRow](
			s.logger,
			user_errors.ErrFailedCreateUser,
			method,
			span,
		)
	}

	logSuccess("Successfully created new user", zap.String("email", res.Email), zap.Int("user_id", int(res.UserID)))

	return res, nil
}

func (s *userCommandService) UpdateUser(ctx context.Context, request *requests.UpdateUserRequest) (*db.UpdateUserRow, error) {
	const method = "UpdateUser"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("user_id", *request.UserID))

	defer func() {
		end(status)
	}()

	s.logger.Debug("Updating user", zap.Int("user_id", *request.UserID), zap.Any("request", request))

	existingUser, err := s.userQueryRepository.FindById(ctx, *request.UserID)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateUserRow](
			s.logger,
			user_errors.ErrUserNotFoundRes,
			method,
			span,

			zap.Int("user_id", *request.UserID),
		)
	}

	if request.Email != "" && request.Email != existingUser.Email {
		duplicateUser, _ := s.userQueryRepository.FindByEmail(ctx, request.Email)
		if duplicateUser != nil {
			status = "error"
			return errorhandler.HandleError[*db.UpdateUserRow](
				s.logger,
				user_errors.ErrUserEmailAlready,
				method,
				span,
				zap.String("email", request.Email),
			)
		}
		existingUser.Email = request.Email
	}

	if request.Password != "" {
		hash, err := s.hashing.HashPassword(request.Password)
		if err != nil {
			status = "error"
			return errorhandler.HandleError[*db.UpdateUserRow](
				s.logger,
				user_errors.ErrUserPassword,
				method,
				span,
			)
		}
		existingUser.Password = hash
	}

	res, err := s.userCommandRepository.UpdateUser(ctx, request)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateUserRow](
			s.logger,
			user_errors.ErrFailedUpdateUser,
			method,
			span,

			zap.Int("user_id", *request.UserID),
		)
	}

	logSuccess("Successfully updated user", zap.Int("user_id", int(res.UserID)))

	return res, nil
}

func (s *userCommandService) TrashedUser(ctx context.Context, user_id int) (*db.TrashUserRow, error) {
	const method = "TrashedUser"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("user_id", user_id))

	defer func() {
		end(status)
	}()

	s.logger.Debug("Trashing user", zap.Int("user_id", user_id))

	res, err := s.userCommandRepository.TrashedUser(ctx, user_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.TrashUserRow](
			s.logger,
			user_errors.ErrFailedTrashedUser,
			method,
			span,

			zap.Int("user_id", user_id),
		)
	}

	logSuccess("Successfully trashed user", zap.Int("user_id", user_id))

	return res, nil
}

func (s *userCommandService) RestoreUser(ctx context.Context, user_id int) (*db.RestoreUserRow, error) {
	const method = "RestoreUser"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("user_id", user_id))

	defer func() {
		end(status)
	}()

	s.logger.Debug("Restoring user", zap.Int("user_id", user_id))

	res, err := s.userCommandRepository.RestoreUser(ctx, user_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.RestoreUserRow](
			s.logger,
			user_errors.ErrFailedRestoreUser,
			method,
			span,

			zap.Int("user_id", user_id),
		)
	}

	logSuccess("Successfully restored user", zap.Int("user_id", user_id))

	return res, nil
}

func (s *userCommandService) DeleteUserPermanent(ctx context.Context, user_id int) (bool, error) {
	const method = "DeleteUserPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("user_id", user_id))

	defer func() {
		end(status)
	}()

	s.logger.Debug("Deleting user permanently", zap.Int("user_id", user_id))

	_, err := s.userCommandRepository.DeleteUserPermanent(ctx, user_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			user_errors.ErrFailedDeletePermanent,
			method,
			span,

			zap.Int("user_id", user_id),
		)
	}

	logSuccess("Successfully deleted user permanently", zap.Int("user_id", user_id))

	return true, nil
}

func (s *userCommandService) RestoreAllUser(ctx context.Context) (bool, error) {
	const method = "RestoreAllUser"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status)
	}()

	s.logger.Debug("Restoring all users")

	_, err := s.userCommandRepository.RestoreAllUser(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			user_errors.ErrFailedRestoreAll,
			method,
			span,
		)
	}

	logSuccess("Successfully restored all users")

	return true, nil
}

func (s *userCommandService) DeleteAllUserPermanent(ctx context.Context) (bool, error) {
	const method = "DeleteAllUserPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status)
	}()

	s.logger.Debug("Permanently deleting all users")

	_, err := s.userCommandRepository.DeleteAllUserPermanent(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			user_errors.ErrFailedDeleteAll,
			method,
			span,
		)
	}

	logSuccess("Successfully deleted all users permanently")

	return true, nil
}
func (s *userCommandService) UpdateIsVerified(ctx context.Context, userID int, isVerified bool) (*db.UpdateUserIsVerifiedRow, error) {
	const method = "UpdateIsVerified"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("user_id", userID),
		attribute.Bool("is_verified", isVerified))

	defer func() {
		end(status)
	}()

	res, err := s.userCommandRepository.UpdateIsVerified(ctx, userID, isVerified)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateUserIsVerifiedRow](
			s.logger,
			user_errors.ErrFailedUpdateUser,
			method,
			span,
			zap.Int("user_id", userID),
		)
	}

	logSuccess("Successfully updated user verification status", zap.Int("user_id", userID))

	return res, nil
}

func (s *userCommandService) UpdatePassword(ctx context.Context, userID int, password string) (*db.UpdateUserPasswordRow, error) {
	const method = "UpdatePassword"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("user_id", userID))

	defer func() {
		end(status)
	}()

	hash, err := s.hashing.HashPassword(password)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateUserPasswordRow](
			s.logger,
			user_errors.ErrUserPassword,
			method,
			span,
		)
	}

	res, err := s.userCommandRepository.UpdatePassword(ctx, userID, hash)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateUserPasswordRow](
			s.logger,
			user_errors.ErrFailedUpdateUser,
			method,
			span,
			zap.Int("user_id", userID),
		)
	}

	logSuccess("Successfully updated user password", zap.Int("user_id", userID))

	return res, nil
}
