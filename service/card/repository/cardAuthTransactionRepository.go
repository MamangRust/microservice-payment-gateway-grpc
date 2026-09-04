package repository

import (
	"context"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	"github.com/jackc/pgx/v5/pgtype"
)

type cardAuthTransactionRepository struct {
	db *db.Queries
}

func NewCardAuthTransactionRepository(q *db.Queries) CardAuthTransactionRepository {
	return &cardAuthTransactionRepository{db: q}
}

func (r *cardAuthTransactionRepository) InsertPending(ctx context.Context, req *requests.AuthorizeCardRequest) (*db.CardAuthTransaction, error) {
	res, err := r.db.InsertAuthTransaction(ctx, db.InsertAuthTransactionParams{
		TxnID:          req.IdempotencyKey + "-txn",
		CardNumber:     req.CardNumber,
		MerchantID:     int32(req.MerchantID),
		Amount:         req.Amount,
		Currency:       req.Currency,
		Mcc:            req.Mcc,
		PosEntryMode:   req.PosEntryMode,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, sharedErrors.ErrFailed("insert authorization transaction").WithInternal(err)
	}
	return res, nil
}

func (r *cardAuthTransactionRepository) Approve(ctx context.Context, txnID string) (*db.CardAuthTransaction, error) {
	res, err := r.db.ApproveAuthTransaction(ctx, txnID)
	if err != nil {
		return nil, sharedErrors.ErrFailed("approve authorization transaction").WithInternal(err)
	}
	return res, nil
}

func (r *cardAuthTransactionRepository) Decline(ctx context.Context, txnID string) (*db.CardAuthTransaction, error) {
	res, err := r.db.DeclineAuthTransaction(ctx, txnID)
	if err != nil {
		return nil, sharedErrors.ErrFailed("decline authorization transaction").WithInternal(err)
	}
	return res, nil
}

func (r *cardAuthTransactionRepository) Reverse(ctx context.Context, txnID string) (*db.CardAuthTransaction, error) {
	res, err := r.db.ReverseAuthTransaction(ctx, txnID)
	if err != nil {
		return nil, sharedErrors.ErrFailed("reverse authorization transaction").WithInternal(err)
	}
	return res, nil
}

func (r *cardAuthTransactionRepository) FindByIdempotencyKey(ctx context.Context, key string) (*db.CardAuthTransaction, error) {
	res, err := r.db.GetAuthTransactionByIdempotencyKey(ctx, key)
	if err != nil {
		return nil, sharedErrors.ErrFailed("get authorization transaction").WithInternal(err)
	}
	return res, nil
}

func (r *cardAuthTransactionRepository) FindByTxnID(ctx context.Context, txnID string) (*db.CardAuthTransaction, error) {
	res, err := r.db.GetAuthTransactionByTxnId(ctx, txnID)
	if err != nil {
		return nil, sharedErrors.ErrFailed("get authorization transaction").WithInternal(err)
	}
	return res, nil
}

func (r *cardAuthTransactionRepository) FindByCardNumber(ctx context.Context, cardNumber string, page, pageSize int) ([]*db.GetAuthTransactionsByCardNumberRow, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	res, err := r.db.GetAuthTransactionsByCardNumber(ctx, db.GetAuthTransactionsByCardNumberParams{
		CardNumber: cardNumber,
		Limit:      int32(pageSize),
		Offset:     int32((page - 1) * pageSize),
	})
	if err != nil {
		return nil, sharedErrors.ErrFailed("get authorization transactions").WithInternal(err)
	}
	return res, nil
}

func (r *cardAuthTransactionRepository) CountRecentByCardNumber(ctx context.Context, cardNumber string, since time.Time) (int, error) {
	count, err := r.db.CountRecentAuthByCardNumber(ctx, db.CountRecentAuthByCardNumberParams{
		CardNumber: cardNumber,
		CreatedAt:  pgtype.Timestamp{Time: since, Valid: true},
	})
	if err != nil {
		return 0, sharedErrors.ErrFailed("count recent authorization transactions").WithInternal(err)
	}
	return int(count), nil
}

func (r *cardAuthTransactionRepository) UpdateRiskScore(ctx context.Context, txnID string, score int) error {
	if err := r.db.UpdateAuthTransactionRiskScore(ctx, db.UpdateAuthTransactionRiskScoreParams{
		TxnID:     txnID,
		RiskScore: int32(score),
	}); err != nil {
		return sharedErrors.ErrFailed("update authorization risk score").WithInternal(err)
	}
	return nil
}
