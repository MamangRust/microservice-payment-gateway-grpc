package adapter

import (
	"context"

	pbuser "github.com/MamangRust/microservice-payment-gateway-grpc/pb/user"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/models"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/resilience"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

// UserAdapter is the read contract consumers depend on for user reads.
type UserAdapter interface {
	FindById(ctx context.Context, id int) (*models.User, error)
}

// AuthUserAdapter is the contract the auth service depends on for user reads and
// the writes it performs on behalf of authentication flows.
type AuthUserAdapter interface {
	FindById(ctx context.Context, id int) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByEmailAndVerify(ctx context.Context, email string) (*models.User, error)
	FindByVerificationCode(ctx context.Context, code string) (*models.User, error)
	CreateUser(ctx context.Context, request *requests.RegisterRequest) (*models.User, error)
	UpdateUserIsVerified(ctx context.Context, userID int, isVerified bool) error
	UpdateUserPassword(ctx context.Context, userID int, password string) error
	DeleteUserPermanent(ctx context.Context, userID int) error
}

type userGRPCAdapter struct {
	QueryClient   pbuser.UserQueryServiceClient
	CommandClient pbuser.UserCommandServiceClient
	guard         *resilience.DependencyGuard
}

func (a *userGRPCAdapter) SetGuard(g *resilience.DependencyGuard) {
	a.guard = g
}

// NewUserAdapter builds a read-only user adapter (used by services that only
// need to resolve a user by id).
func NewUserAdapter(queryClient pbuser.UserQueryServiceClient, opts ...GuardOption) UserAdapter {
	a := &userGRPCAdapter{QueryClient: queryClient}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// NewAuthUserAdapter builds a user adapter with both query and command clients
// for the auth service's register/login/reset flows.
func NewAuthUserAdapter(queryClient pbuser.UserQueryServiceClient, commandClient pbuser.UserCommandServiceClient, opts ...GuardOption) AuthUserAdapter {
	a := &userGRPCAdapter{QueryClient: queryClient, CommandClient: commandClient}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *userGRPCAdapter) FindById(ctx context.Context, id int) (*models.User, error) {
	var resp *pbuser.ApiResponseUser
	err := a.guard.Call(ctx, func(ctx context.Context) error {
		var callErr error
		resp, callErr = a.QueryClient.FindById(ctx, &pbuser.FindByIdUserRequest{Id: int32(id)})
		return callErr
	})
	if err != nil || resp == nil || resp.Data == nil {
		return nil, sharedErrors.ErrUserNotFound.WithInternal(err)
	}
	return userToModel(resp.Data), nil
}

func (a *userGRPCAdapter) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var resp *pbuser.ApiResponseUser
	err := a.guard.Call(ctx, func(ctx context.Context) error {
		var callErr error
		resp, callErr = a.QueryClient.FindByEmail(ctx, &pbuser.FindByEmailUserRequest{Email: email})
		return callErr
	})
	if err != nil || resp == nil || resp.Data == nil {
		return nil, sharedErrors.ErrUserNotFound.WithInternal(err)
	}
	return userToModel(resp.Data), nil
}

// FindByEmailAndVerify resolves a user by email for credential verification. The
// user service returns the password hash on the email lookup, so we reuse
// FindByEmail here.
func (a *userGRPCAdapter) FindByEmailAndVerify(ctx context.Context, email string) (*models.User, error) {
	return a.FindByEmail(ctx, email)
}

func (a *userGRPCAdapter) FindByVerificationCode(ctx context.Context, code string) (*models.User, error) {
	var resp *pbuser.ApiResponseUser
	err := a.guard.Call(ctx, func(ctx context.Context) error {
		var callErr error
		resp, callErr = a.QueryClient.FindByVerificationCode(ctx, &pbuser.FindByVerificationCodeUserRequest{VerificationCode: code})
		return callErr
	})
	if err != nil || resp == nil || resp.Data == nil {
		return nil, sharedErrors.ErrUserNotFound.WithInternal(err)
	}
	return userToModel(resp.Data), nil
}

func (a *userGRPCAdapter) CreateUser(ctx context.Context, request *requests.RegisterRequest) (*models.User, error) {
	var resp *pbuser.ApiResponseUser
	err := a.guard.Call(ctx, func(ctx context.Context) error {
		var callErr error
		resp, callErr = a.CommandClient.Create(ctx, &pbuser.CreateUserRequest{
			Firstname:       request.FirstName,
			Lastname:        request.LastName,
			Email:           request.Email,
			Password:        request.Password,
			ConfirmPassword: request.ConfirmPassword,
		})
		return callErr
	})
	if err != nil || resp == nil || resp.Data == nil {
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}
	return userToModel(resp.Data), nil
}

func (a *userGRPCAdapter) UpdateUserIsVerified(ctx context.Context, userID int, isVerified bool) error {
	err := a.guard.Call(ctx, func(ctx context.Context) error {
		_, callErr := a.CommandClient.UpdateIsVerified(ctx, &pbuser.UpdateUserIsVerifiedRequest{
			UserId:     int32(userID),
			IsVerified: isVerified,
		})
		return callErr
	})
	if err != nil {
		return sharedErrors.ErrInternal.WithInternal(err)
	}
	return nil
}

func (a *userGRPCAdapter) UpdateUserPassword(ctx context.Context, userID int, password string) error {
	err := a.guard.Call(ctx, func(ctx context.Context) error {
		_, callErr := a.CommandClient.UpdatePassword(ctx, &pbuser.UpdateUserPasswordRequest{
			UserId:   int32(userID),
			Password: password,
		})
		return callErr
	})
	if err != nil {
		return sharedErrors.ErrInternal.WithInternal(err)
	}
	return nil
}

func (a *userGRPCAdapter) DeleteUserPermanent(ctx context.Context, userID int) error {
	err := a.guard.Call(ctx, func(ctx context.Context) error {
		_, callErr := a.CommandClient.DeleteUserPermanent(ctx, &pbuser.FindByIdUserRequest{Id: int32(userID)})
		return callErr
	})
	if err != nil {
		return sharedErrors.ErrInternal.WithInternal(err)
	}
	return nil
}

func userToModel(u *pbuser.UserResponse) *models.User {
	if u == nil {
		return nil
	}
	return &models.User{
		UserID:    u.Id,
		Firstname: u.Firstname,
		Lastname:  u.Lastname,
		Email:     u.Email,
		Password:  u.Password,
		CreatedAt: parseTime(u.CreatedAt),
		UpdatedAt: parseTime(u.UpdatedAt),
	}
}
