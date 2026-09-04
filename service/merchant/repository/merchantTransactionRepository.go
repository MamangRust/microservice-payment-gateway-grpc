package repository

import (
	"context"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

type merchantTransactionRepository struct {
	db *db.Queries
}

func NewMerchantTransactionRepository(db *db.Queries) MerchantTransactionRepository {
	return &merchantTransactionRepository{
		db: db,
	}
}

func (r *merchantTransactionRepository) FindAllTransactions(ctx context.Context, req *requests.FindAllMerchantTransactions) ([]*db.FindAllTransactionsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.FindAllTransactionsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	merchant, err := r.db.FindAllTransactions(ctx, reqDb)

	if err != nil {
		return nil, sharedErrors.ErrFailed("find all merchant transactions").WithInternal(err)
	}

	return merchant, nil
}

func (r *merchantTransactionRepository) FindAllTransactionsByMerchant(ctx context.Context, req *requests.FindAllMerchantTransactionsById) ([]*db.FindAllTransactionsByMerchantRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.FindAllTransactionsByMerchantParams{
		MerchantID: int32(req.MerchantID),
		Column2:    req.Search,
		Limit:      int32(req.PageSize),
		Offset:     int32(offset),
	}

	merchant, err := r.db.FindAllTransactionsByMerchant(ctx, reqDb)

	if err != nil {
		return nil, sharedErrors.ErrFailed("find merchant transactions by merchant").WithInternal(err)
	}

	return merchant, nil
}

func (r *merchantTransactionRepository) FindAllTransactionsByApikey(ctx context.Context, req *requests.FindAllMerchantTransactionsByApiKey) ([]*db.FindAllTransactionsByApikeyRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.FindAllTransactionsByApikeyParams{
		ApiKey:  req.ApiKey,
		Column2: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	merchant, err := r.db.FindAllTransactionsByApikey(ctx, reqDb)

	if err != nil {
		return nil, sharedErrors.ErrFailed("find merchant transactions by API key").WithInternal(err)
	}

	return merchant, nil
}
