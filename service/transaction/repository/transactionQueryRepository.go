package repository

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	transaction_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/transaction_errors/repository"
)

type transactionQueryRepository struct {
	db *db.Queries
}

func NewTransactionQueryRepository(db *db.Queries) TransactionQueryRepository {
	return &transactionQueryRepository{
		db: db,
	}
}

func (r *transactionQueryRepository) FindAllTransactions(ctx context.Context, req *requests.FindAllTransactions) ([]*db.GetTransactionsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetTransactionsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	transactions, err := r.db.GetTransactions(ctx, reqDb)

	if err != nil {
		return nil, transaction_errors.ErrFindAllTransactionsFailed.WithInternal(err)
	}

	return transactions, nil
}

func (r *transactionQueryRepository) FindAllTransactionByCardNumber(ctx context.Context, req *requests.FindAllTransactionCardNumber) ([]*db.GetTransactionsByCardNumberRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetTransactionsByCardNumberParams{
		CardNumber: req.CardNumber,
		Column2:    req.Search,
		Limit:      int32(req.PageSize),
		Offset:     int32(offset),
	}

	transactions, err := r.db.GetTransactionsByCardNumber(ctx, reqDb)

	if err != nil {
		return nil, transaction_errors.ErrFindTransactionsByCardNumberFailed.WithInternal(err)
	}

	return transactions, nil
}

func (r *transactionQueryRepository) FindByActive(ctx context.Context, req *requests.FindAllTransactions) ([]*db.GetActiveTransactionsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetActiveTransactionsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetActiveTransactions(ctx, reqDb)

	if err != nil {
		return nil, transaction_errors.ErrFindActiveTransactionsFailed.WithInternal(err)
	}

	return res, nil
}

func (r *transactionQueryRepository) FindByTrashed(ctx context.Context, req *requests.FindAllTransactions) ([]*db.GetTrashedTransactionsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetTrashedTransactionsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetTrashedTransactions(ctx, reqDb)

	if err != nil {
		return nil, transaction_errors.ErrFindTrashedTransactionsFailed.WithInternal(err)
	}

	return res, nil
}

func (r *transactionQueryRepository) FindById(ctx context.Context, transaction_id int) (*db.GetTransactionByIDRow, error) {
	res, err := r.db.GetTransactionByID(ctx, int32(transaction_id))

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, transaction_errors.ErrFindTransactionByIdFailed.WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	return res, nil
}

func (r *transactionQueryRepository) FindTransactionByMerchantId(ctx context.Context, merchant_id int) ([]*db.GetTransactionsByMerchantIDRow, error) {
	res, err := r.db.GetTransactionsByMerchantID(ctx, int32(merchant_id))

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, transaction_errors.ErrFindTransactionByMerchantIdFailed.WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	return res, nil
}
