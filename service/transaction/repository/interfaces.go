package repository

import (
	"context"
	"time"

	carddb "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	merchantdb "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/database/schema"
	saldodb "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/idempotency"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/outbox"
)

type IdempotencyRepository interface {
	idempotency.Store
}

type OutboxRepository interface {
	outbox.Store[db.OutboxRecord]
}

type MerchantRepository interface {
	FindByApiKey(ctx context.Context, api_key string) (*merchantdb.GetMerchantByApiKeyRow, error)
}

type SaldoRepository interface {
	FindByCardNumber(ctx context.Context, card_number string) (*saldodb.Saldo, error)

	UpdateSaldoBalance(ctx context.Context, request *requests.UpdateSaldoBalance) (*saldodb.UpdateSaldoBalanceRow, error)
}

type CardRepository interface {
	FindCardByUserId(ctx context.Context, user_id int) (*carddb.GetCardByUserIDRow, error)

	FindUserCardByCardNumber(ctx context.Context, card_number string) (*carddb.GetUserEmailByCardNumberRow, error)

	FindCardByCardNumber(ctx context.Context, card_number string) (*carddb.GetCardByCardNumberRow, error)

	UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*carddb.UpdateCardRow, error)
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
	TransitionStatus(ctx context.Context, id int, fromStatus, toStatus, reason string) (*db.UpdateTransactionStatusRow, error)
	GuardStatus(ctx context.Context, id int, fromStatus, toStatus, reason string) (bool, error)
	ListStuck(ctx context.Context, olderThan time.Duration, maxRows int32) ([]*db.StuckTransaction, error)
	TrashedTransaction(ctx context.Context, transaction_id int) (*db.Transaction, error)
	RestoreTransaction(ctx context.Context, topup_id int) (*db.Transaction, error)
	DeleteTransactionPermanent(ctx context.Context, topup_id int) (bool, error)

	RestoreAllTransaction(ctx context.Context) (bool, error)
	DeleteAllTransactionPermanent(ctx context.Context) (bool, error)
}
