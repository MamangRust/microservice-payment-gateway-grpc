package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

type MerchantRepository interface {
	FindByApiKey(ctx context.Context, api_key string) (*db.GetMerchantByApiKeyRow, error)
}

type SaldoRepository interface {
	FindByCardNumber(ctx context.Context, card_number string) (*db.Saldo, error)

	UpdateSaldoBalance(ctx context.Context, request *requests.UpdateSaldoBalance) (*db.UpdateSaldoBalanceRow, error)
}

type CardRepository interface {
	FindCardByUserId(ctx context.Context, user_id int) (*db.GetCardByUserIDRow, error)

	FindUserCardByCardNumber(ctx context.Context, card_number string) (*db.GetUserEmailByCardNumberRow, error)

	FindCardByCardNumber(ctx context.Context, card_number string) (*db.GetCardByCardNumberRow, error)

	UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*db.UpdateCardRow, error)
}

type TransactionQueryRepository interface {
	FindAllTransactions(ctx context.Context, req *requests.FindAllTransactions) ([]*db.GetTransactionsRow, error)
	FindByActive(ctx context.Context, req *requests.FindAllTransactions) ([]*db.GetActiveTransactionsRow, error)
	FindByTrashed(ctx context.Context, req *requests.FindAllTransactions) ([]*db.GetTrashedTransactionsRow, error)
	FindAllTransactionByCardNumber(ctx context.Context, req *requests.FindAllTransactionCardNumber) ([]*db.GetTransactionsByCardNumberRow, error)
	FindById(ctx context.Context, transaction_id int) (*db.GetTransactionByIDRow, error)
	FindTransactionByMerchantId(ctx context.Context, merchant_id int) ([]*db.GetTransactionsByMerchantIDRow, error)
}

type TransactionCommandRepository interface {
	CreateTransaction(ctx context.Context, request *requests.CreateTransactionRequest) (*db.CreateTransactionRow, error)
	UpdateTransaction(ctx context.Context, request *requests.UpdateTransactionRequest) (*db.UpdateTransactionRow, error)
	UpdateTransactionStatus(ctx context.Context, request *requests.UpdateTransactionStatus) (*db.UpdateTransactionStatusRow, error)
	TrashedTransaction(ctx context.Context, transaction_id int) (*db.Transaction, error)
	RestoreTransaction(ctx context.Context, topup_id int) (*db.Transaction, error)
	DeleteTransactionPermanent(ctx context.Context, topup_id int) (bool, error)

	RestoreAllTransaction(ctx context.Context) (bool, error)
	DeleteAllTransactionPermanent(ctx context.Context) (bool, error)
}
