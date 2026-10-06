package repository

import (
	"context"
	"time"

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
