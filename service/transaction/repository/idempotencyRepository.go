package repository

import (
	"context"
	"errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/idempotency"
	"github.com/jackc/pgx/v5"
)

// TransactionIdempotencyRepository adapts the shared idempotency.Store contract
// to the transaction service DB's idempotency_records table.
type TransactionIdempotencyRepository struct {
	db *db.Queries
}

func NewTransactionIdempotencyRepository(db *db.Queries) *TransactionIdempotencyRepository {
	return &TransactionIdempotencyRepository{db: db}
}

func (r *TransactionIdempotencyRepository) Claim(ctx context.Context, scope, key, requestHash string) (*idempotency.Record, error) {
	rec, err := r.db.ClaimIdempotency(ctx, scope, key, requestHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, idempotency.ErrKeyInUse
		}
		return nil, err
	}
	return mapTransactionIdempotencyRecord(rec), nil
}

func (r *TransactionIdempotencyRepository) Get(ctx context.Context, scope, key string) (*idempotency.Record, error) {
	rec, err := r.db.GetIdempotency(ctx, scope, key)
	if err != nil {
		return nil, err
	}
	return mapTransactionIdempotencyRecord(rec), nil
}

func (r *TransactionIdempotencyRepository) Complete(ctx context.Context, scope, key, requestHash string, outcome idempotency.Outcome) error {
	return r.db.CompleteIdempotency(ctx, db.CompleteIdempotencyParams{
		Scope:           scope,
		IdempotencyKey:  key,
		RequestHash:     requestHash,
		Status:          outcome.Status,
		ResponsePayload: outcome.ResponseJSON,
		ResourceID:      outcome.ResourceID,
	})
}

func (r *TransactionIdempotencyRepository) Release(ctx context.Context, scope, key, requestHash string) error {
	return r.db.ReleaseIdempotency(ctx, scope, key, requestHash)
}

func mapTransactionIdempotencyRecord(rec *db.IdempotencyRecord) *idempotency.Record {
	if rec == nil {
		return nil
	}
	return &idempotency.Record{
		ID:           rec.ID,
		Key:          rec.IdempotencyKey,
		RequestHash:  rec.RequestHash,
		OperationID:  rec.OperationID.String(),
		Status:       rec.Status,
		ResponseJSON: rec.ResponsePayload,
		ResourceID:   rec.ResourceID,
		CreatedAt:    rec.CreatedAt,
		UpdatedAt:    rec.UpdatedAt,
		ExpiresAt:    rec.ExpiresAt,
	}
}
