package repository

import (
	"context"
	"errors"
	"time"

	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/state"
	"github.com/jackc/pgx/v5"
)

type transactionCommandRepository struct {
	db *db.Queries
}

func NewTransactionCommandRepository(db *db.Queries) TransactionCommandRepository {
	return &transactionCommandRepository{
		db: db,
	}
}

func (r *transactionCommandRepository) CreateTransaction(ctx context.Context, request *requests.CreateTransactionRequest) (*db.CreateTransactionRow, error) {
	req := db.CreateTransactionParams{
		CardNumber: request.CardNumber, Amount: int64(request.Amount),
		PaymentMethod:   request.PaymentMethod,
		MerchantID:      int32(*request.MerchantID),
		TransactionTime: request.TransactionTime,
	}

	res, err := r.db.CreateTransaction(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrFailed("create transaction").WithInternal(err)
	}

	return res, nil
}

func (r *transactionCommandRepository) UpdateTransaction(ctx context.Context, request *requests.UpdateTransactionRequest) (*db.UpdateTransactionRow, error) {
	req := db.UpdateTransactionParams{
		TransactionID: int32(*request.TransactionID),
		CardNumber:    request.CardNumber, Amount: int64(request.Amount),
		PaymentMethod:   request.PaymentMethod,
		MerchantID:      int32(*request.MerchantID),
		TransactionTime: request.TransactionTime,
	}

	res, err := r.db.UpdateTransaction(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "transaction", "update transaction")
	}

	return res, nil
}

func (r *transactionCommandRepository) UpdateTransactionStatus(ctx context.Context, request *requests.UpdateTransactionStatus) (*db.UpdateTransactionStatusRow, error) {
	req := db.UpdateTransactionStatusParams{
		TransactionID: int32(request.TransactionID),
		Status:        request.Status,
	}

	res, err := r.db.UpdateTransactionStatus(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "transaction", "update transaction status")
	}

	return res, nil
}

// GuardStatus atomically transitions a transaction from an expected status to a
// new one. It reports false (with nil error) when the row is not in the expected
// status, i.e. another actor already moved it — callers use this to make
// compensation exactly-once under concurrency.
func (r *transactionCommandRepository) GuardStatus(ctx context.Context, id int, fromStatus, toStatus, reason string) (bool, error) {
	if err := state.CheckTransition(fromStatus, toStatus); err != nil {
		return false, err
	}
	_, err := r.db.GuardTransactionStatus(ctx, int32(id), fromStatus, toStatus, reason)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListStuck returns transactions stuck in a recoverable state for the recovery worker.
func (r *transactionCommandRepository) TransitionStatus(ctx context.Context, id int, fromStatus, toStatus, reason string) (*db.UpdateTransactionStatusRow, error) {
	if err := state.CheckTransition(fromStatus, toStatus); err != nil {
		return nil, err
	}
	return r.db.GuardTransactionStatus(ctx, int32(id), fromStatus, toStatus, reason)
}

func (r *transactionCommandRepository) ListStuck(ctx context.Context, olderThan time.Duration, maxRows int32) ([]*db.StuckTransaction, error) {
	return r.db.ListStuckTransactions(ctx, olderThan, maxRows)
}

func (r *transactionCommandRepository) TrashedTransaction(ctx context.Context, transaction_id int) (*db.Transaction, error) {
	res, err := r.db.TrashTransaction(ctx, int32(transaction_id))
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "transaction", "trash transaction")
	}
	return res, nil
}

func (r *transactionCommandRepository) RestoreTransaction(ctx context.Context, transaction_id int) (*db.Transaction, error) {
	res, err := r.db.RestoreTransaction(ctx, int32(transaction_id))
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "transaction", "restore transaction")
	}
	return res, nil
}

func (r *transactionCommandRepository) DeleteTransactionPermanent(ctx context.Context, transaction_id int) (bool, error) {
	err := r.db.DeleteTransactionPermanently(ctx, int32(transaction_id))
	if err != nil {

		return false, sharedErrors.ErrNoRowsOrFailed(err, "transaction", "delete transaction permanently")
	}
	return true, nil
}

func (r *transactionCommandRepository) RestoreAllTransaction(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllTransactions(ctx)

	if err != nil {
		return false, sharedErrors.ErrFailed("restore all transactions").WithInternal(err)
	}

	return true, nil
}

func (r *transactionCommandRepository) DeleteAllTransactionPermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentTransactions(ctx)

	if err != nil {
		return false, sharedErrors.ErrFailed("delete all transactions permanently").WithInternal(err)
	}
	return true, nil
}
