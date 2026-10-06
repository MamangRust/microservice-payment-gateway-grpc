package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/database/schema"
)

// MerchantTransactionRepository exposes merchant-owned transaction reads through
// the merchant database schema.
type MerchantTransactionRepository interface {
	FindAllTransactions(ctx context.Context, arg db.FindAllTransactionsParams) ([]*db.FindAllTransactionsRow, error)
	FindAllTransactionsByApikey(ctx context.Context, arg db.FindAllTransactionsByApikeyParams) ([]*db.FindAllTransactionsByApikeyRow, error)
	FindAllTransactionsByMerchant(ctx context.Context, arg db.FindAllTransactionsByMerchantParams) ([]*db.FindAllTransactionsByMerchantRow, error)
}

type merchantTransactionRepository struct {
	Queries *db.Queries
}

// NewMerchantTransactionRepository builds a MerchantTransactionRepository backed
// by the merchant query store.
func NewMerchantTransactionRepository(queries *db.Queries) MerchantTransactionRepository {
	return &merchantTransactionRepository{Queries: queries}
}

func (r *merchantTransactionRepository) FindAllTransactions(ctx context.Context, arg db.FindAllTransactionsParams) ([]*db.FindAllTransactionsRow, error) {
	return r.Queries.FindAllTransactions(ctx, arg)
}

func (r *merchantTransactionRepository) FindAllTransactionsByApikey(ctx context.Context, arg db.FindAllTransactionsByApikeyParams) ([]*db.FindAllTransactionsByApikeyRow, error) {
	return r.Queries.FindAllTransactionsByApikey(ctx, arg)
}

func (r *merchantTransactionRepository) FindAllTransactionsByMerchant(ctx context.Context, arg db.FindAllTransactionsByMerchantParams) ([]*db.FindAllTransactionsByMerchantRow, error) {
	return r.Queries.FindAllTransactionsByMerchant(ctx, arg)
}
