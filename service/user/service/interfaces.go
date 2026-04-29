package service

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

// UserQueryService handles query operations related to user data.
type UserQueryService interface {
	FindAll(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetUsersWithPaginationRow, *int, error)
	FindByID(ctx context.Context, id int) (*db.GetUserByIDRow, error)
	FindByEmail(ctx context.Context, email string) (*db.GetUserByEmailRow, error)
	FindByVerificationCode(ctx context.Context, verificationCode string) (*db.GetUserByVerificationCodeRow, error)
	FindByActive(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetActiveUsersWithPaginationRow, *int, error)
	FindByTrashed(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetTrashedUsersWithPaginationRow, *int, error)
}

// UserCommandService handles command operations related to user management.
type UserCommandService interface {
	CreateUser(ctx context.Context, request *requests.CreateUserRequest) (*db.CreateUserRow, error)
	UpdateUser(ctx context.Context, request *requests.UpdateUserRequest) (*db.UpdateUserRow, error)
	UpdateIsVerified(ctx context.Context, userID int, isVerified bool) (*db.UpdateUserIsVerifiedRow, error)
	UpdatePassword(ctx context.Context, userID int, password string) (*db.UpdateUserPasswordRow, error)
	TrashedUser(ctx context.Context, user_id int) (*db.TrashUserRow, error)
	RestoreUser(ctx context.Context, user_id int) (*db.RestoreUserRow, error)
	DeleteUserPermanent(ctx context.Context, user_id int) (bool, error)

	RestoreAllUser(ctx context.Context) (bool, error)
	DeleteAllUserPermanent(ctx context.Context) (bool, error)
}
