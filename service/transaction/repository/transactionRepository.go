package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	transaction_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/transaction_errors/repository"
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
		CardNumber:      request.CardNumber,
		Amount:          int32(request.Amount),
		PaymentMethod:   request.PaymentMethod,
		MerchantID:      int32(*request.MerchantID),
		TransactionTime: request.TransactionTime,
	}

	res, err := r.db.CreateTransaction(ctx, req)

	if err != nil {
		return nil, transaction_errors.ErrCreateTransactionFailed.WithInternal(err)
	}

	return res, nil
}

func (r *transactionCommandRepository) UpdateTransaction(ctx context.Context, request *requests.UpdateTransactionRequest) (*db.UpdateTransactionRow, error) {
	req := db.UpdateTransactionParams{
		TransactionID:   int32(*request.TransactionID),
		CardNumber:      request.CardNumber,
		Amount:          int32(request.Amount),
		PaymentMethod:   request.PaymentMethod,
		MerchantID:      int32(*request.MerchantID),
		TransactionTime: request.TransactionTime,
	}

	res, err := r.db.UpdateTransaction(ctx, req)

	if err != nil {
		return nil, transaction_errors.ErrUpdateTransactionFailed.WithInternal(err)
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
		return nil, transaction_errors.ErrUpdateTransactionStatusFailed.WithInternal(err)
	}

	return res, nil
}

func (r *transactionCommandRepository) TrashedTransaction(ctx context.Context, transaction_id int) (*db.Transaction, error) {
	res, err := r.db.TrashTransaction(ctx, int32(transaction_id))
	if err != nil {
		return nil, transaction_errors.ErrTrashedTransactionFailed.WithInternal(err)
	}
	return res, nil
}

func (r *transactionCommandRepository) RestoreTransaction(ctx context.Context, transaction_id int) (*db.Transaction, error) {
	res, err := r.db.RestoreTransaction(ctx, int32(transaction_id))
	if err != nil {
		return nil, transaction_errors.ErrRestoreTransactionFailed.WithInternal(err)
	}
	return res, nil
}

func (r *transactionCommandRepository) DeleteTransactionPermanent(ctx context.Context, transaction_id int) (bool, error) {
	err := r.db.DeleteTransactionPermanently(ctx, int32(transaction_id))
	if err != nil {

		return false, transaction_errors.ErrDeleteTransactionPermanentFailed.WithInternal(err)
	}
	return true, nil
}

func (r *transactionCommandRepository) RestoreAllTransaction(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllTransactions(ctx)

	if err != nil {
		return false, transaction_errors.ErrRestoreAllTransactionsFailed.WithInternal(err)
	}

	return true, nil
}

func (r *transactionCommandRepository) DeleteAllTransactionPermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentTransactions(ctx)

	if err != nil {
		return false, transaction_errors.ErrDeleteAllTransactionsPermanentFailed.WithInternal(err)
	}
	return true, nil
}
