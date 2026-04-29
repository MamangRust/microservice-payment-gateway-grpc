package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

type UserQueryRepository interface {
	FindAllUsers(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetUsersWithPaginationRow, error)
	FindByActive(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetActiveUsersWithPaginationRow, error)
	FindByTrashed(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetTrashedUsersWithPaginationRow, error)
	FindById(ctx context.Context, user_id int) (*db.GetUserByIDRow, error)
	FindByEmail(ctx context.Context, email string) (*db.GetUserByEmailRow, error)
	FindByVerificationCode(ctx context.Context, code string) (*db.GetUserByVerificationCodeRow, error)
}

type UserCommandRepository interface {
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

type RoleRepository interface {
	FindById(ctx context.Context, role_id int) (*db.Role, error)
	FindByName(ctx context.Context, name string) (*db.Role, error)
}
