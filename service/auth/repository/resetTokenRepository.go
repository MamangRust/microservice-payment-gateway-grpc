package repository

import (
	"context"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	refresh_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/refresh_token_errors/repository"
)

// resetTokenRepository is a struct that implements the ResetTokenRepository interface
type resetTokenRepository struct {
	db *db.Queries
}

// NewResetTokenRepository creates a new instance of resetTokenRepository.
func NewResetTokenRepository(db *db.Queries) *resetTokenRepository {
	return &resetTokenRepository{
		db: db,
	}
}

// FindByToken retrieves a reset token record by token string.
func (r *resetTokenRepository) FindByToken(ctx context.Context, code string) (*db.GetResetTokenRow, error) {
	res, err := r.db.GetResetToken(ctx, code)
	if err != nil {
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}
	return res, nil
}

// CreateResetToken inserts a new reset token into the database.
func (r *resetTokenRepository) CreateResetToken(ctx context.Context, req *requests.CreateResetTokenRequest) (*db.CreateResetTokenRow, error) {
	expiryDate, err := time.Parse("2006-01-02 15:04:05", req.ExpiredAt)
	if err != nil {
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}
	res, err := r.db.CreateResetToken(ctx, db.CreateResetTokenParams{
		UserID:     int64(req.UserID),
		Token:      req.ResetToken,
		ExpiryDate: expiryDate,
	})
	if err != nil {
		return nil, refresh_errors.ErrCreateRefreshToken.WithInternal(err)
	}
	return res, nil
}

// DeleteResetToken removes the reset token associated with the given user ID.
func (r *resetTokenRepository) DeleteResetToken(ctx context.Context, user_id int) error {
	err := r.db.DeleteResetToken(ctx, int64(user_id))
	if err != nil {
		return refresh_errors.ErrDeleteByUserID.WithInternal(err)
	}
	return nil
}
