package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/auth/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

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
