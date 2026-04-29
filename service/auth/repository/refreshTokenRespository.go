package repository

import (
	"context"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	refreshtoken_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/refresh_token_errors/repository"
)

// refreshTokenRepository is a struct that implements the RefreshTokenRepository interface
type refreshTokenRepository struct {
	db *db.Queries
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository instance
func NewRefreshTokenRepository(db *db.Queries) *refreshTokenRepository {
	return &refreshTokenRepository{
		db: db,
	}
}

// FindByToken retrieves a refresh token record by the token string.
func (r *refreshTokenRepository) FindByToken(ctx context.Context, token string) (*db.RefreshToken, error) {
	res, err := r.db.FindRefreshTokenByToken(ctx, token)

	if err != nil {
		return nil, refreshtoken_errors.ErrTokenNotFound.WithInternal(err)
	}

	return res, nil
}

// FindByUserId retrieves a refresh token record by the associated user ID.
func (r *refreshTokenRepository) FindByUserId(ctx context.Context, user_id int) (*db.RefreshToken, error) {
	res, err := r.db.FindRefreshTokenByUserId(ctx, int32(user_id))

	if err != nil {
		return nil, refreshtoken_errors.ErrFindByUserID.WithInternal(err)
	}

	return res, nil
}

// CreateRefreshToken inserts a new refresh token record into the database.
func (r *refreshTokenRepository) CreateRefreshToken(ctx context.Context, req *requests.CreateRefreshToken) (*db.RefreshToken, error) {
	layout := "2006-01-02 15:04:05"
	expirationTime, err := time.Parse(layout, req.ExpiresAt)
	if err != nil {
		return nil, refreshtoken_errors.ErrParseDate.WithInternal(err)
	}

	res, err := r.db.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:     int32(req.UserId),
		Token:      req.Token,
		Expiration: expirationTime,
	})

	if err != nil {
		return nil, refreshtoken_errors.ErrCreateRefreshToken.WithInternal(err)
	}

	return res, nil
}

// UpdateRefreshToken updates an existing refresh token record.
func (r *refreshTokenRepository) UpdateRefreshToken(ctx context.Context, req *requests.UpdateRefreshToken) (*db.RefreshToken, error) {
	layout := "2006-01-02 15:04:05"
	expirationTime, err := time.Parse(layout, req.ExpiresAt)
	if err != nil {
		return nil, refreshtoken_errors.ErrParseDate.WithInternal(err)
	}

	res, err := r.db.UpdateRefreshTokenByUserId(ctx, db.UpdateRefreshTokenByUserIdParams{
		UserID:     int32(req.UserId),
		Token:      req.Token,
		Expiration: expirationTime,
	})
	if err != nil {
		return nil, refreshtoken_errors.ErrUpdateRefreshToken.WithInternal(err)
	}

	return res, nil
}

// DeleteRefreshToken removes a refresh token by its token string.
func (r *refreshTokenRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	err := r.db.DeleteRefreshToken(ctx, token)

	if err != nil {
		return refreshtoken_errors.ErrDeleteRefreshToken.WithInternal(err)
	}

	return nil
}

// DeleteRefreshTokenByUserId removes a refresh token by the associated user ID.
func (r *refreshTokenRepository) DeleteRefreshTokenByUserId(ctx context.Context, user_id int) error {
	err := r.db.DeleteRefreshTokenByUserId(ctx, int32(user_id))

	if err != nil {
		return refreshtoken_errors.ErrDeleteByUserID.WithInternal(err)
	}

	return nil
}
