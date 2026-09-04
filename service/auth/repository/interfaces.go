package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/auth/database/schema"
	userdb "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

// UserRepository defines the data access layer for user-related operations.
//
//go:generate mockgen -source=interfaces.go -destination=mocks/mock.go
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*userdb.GetUserByEmailRow, error)

	FindByEmailAndVerify(ctx context.Context, email string) (*userdb.GetUserByEmailAndVerifiedRow, error)

	FindById(ctx context.Context, user_id int) (*userdb.GetUserByIDRow, error)

	CreateUser(ctx context.Context, request *requests.RegisterRequest) (*userdb.CreateUserRow, error)

	UpdateUserIsVerified(ctx context.Context, user_id int, is_verified bool) (*userdb.UpdateUserIsVerifiedRow, error)

	UpdateUserPassword(ctx context.Context, user_id int, password string) (*userdb.UpdateUserPasswordRow, error)

	FindByVerificationCode(ctx context.Context, verification_code string) (*userdb.GetUserByVerificationCodeRow, error)
}

type ResetTokenRepository interface {
	FindByToken(ctx context.Context, code string) (*db.GetResetTokenRow, error)

	CreateResetToken(ctx context.Context, req *requests.CreateResetTokenRequest) (*db.CreateResetTokenRow, error)

	DeleteResetToken(ctx context.Context, user_id int32) error
}

type RefreshTokenRepository interface {
	FindByToken(ctx context.Context, token string) (*db.RefreshToken, error)

	FindByUserId(ctx context.Context, user_id int) (*db.RefreshToken, error)

	CreateRefreshToken(ctx context.Context, req *requests.CreateRefreshToken) (*db.RefreshToken, error)

	UpdateRefreshToken(ctx context.Context, req *requests.UpdateRefreshToken) (*db.RefreshToken, error)

	DeleteRefreshToken(ctx context.Context, token string) error

	DeleteRefreshTokenByUserId(ctx context.Context, user_id int) error
}

type UserRoleRepository interface {
	AssignRoleToUser(ctx context.Context, req *requests.CreateUserRoleRequest) (*db.UserRole, error)

	RemoveRoleFromUser(ctx context.Context, req *requests.RemoveUserRoleRequest) error
}

type RoleRepository interface {
	FindById(ctx context.Context, id int) (*db.Role, error)

	FindByName(ctx context.Context, name string) (*db.Role, error)
}
