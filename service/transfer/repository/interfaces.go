package repository

import (
	"context"
	"time"

	carddb "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	saldodb "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/database/schema"
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

type SaldoRepository interface {
	FindByCardNumber(ctx context.Context, card_number string) (*saldodb.Saldo, error)

	UpdateSaldoBalance(ctx context.Context, request *requests.UpdateSaldoBalance) (*saldodb.UpdateSaldoBalanceRow, error)
}

type CardRepository interface {
	FindUserCardByCardNumber(ctx context.Context, card_number string) (*carddb.GetUserEmailByCardNumberRow, error)

	FindCardByCardNumber(ctx context.Context, card_number string) (*carddb.GetCardByCardNumberRow, error)
}

type TransferQueryRepository interface {
	FindAll(ctx context.Context, req *requests.FindAllTransfers) ([]*db.GetTransfersRow, error)
	FindByActive(ctx context.Context, req *requests.FindAllTransfers) ([]*db.GetActiveTransfersRow, error)
	FindByTrashed(ctx context.Context, req *requests.FindAllTransfers) ([]*db.GetTrashedTransfersRow, error)
	FindById(ctx context.Context, id int) (*db.GetTransferByIDRow, error)
	FindTransferByTransferFrom(ctx context.Context, transferFrom string) ([]*db.GetTransfersBySourceCardRow, error)
	FindTransferByTransferTo(ctx context.Context, transferTo string) ([]*db.GetTransfersByDestinationCardRow, error)
}

type TransferCommandRepository interface {
	CreateTransfer(ctx context.Context, request *requests.CreateTransferRequest) (*db.CreateTransferRow, error)
	UpdateTransfer(ctx context.Context, request *requests.UpdateTransferRequest) (*db.UpdateTransferRow, error)
	UpdateTransferAmount(ctx context.Context, request *requests.UpdateTransferAmountRequest) (*db.UpdateTransferAmountRow, error)
	UpdateTransferStatus(ctx context.Context, request *requests.UpdateTransferStatus) (*db.UpdateTransferStatusRow, error)
	TransitionStatus(ctx context.Context, id int, fromStatus, toStatus, reason string) (*db.UpdateTransferStatusRow, error)
	GuardStatus(ctx context.Context, id int, fromStatus, toStatus, reason string) (bool, error)
	ListStuck(ctx context.Context, olderThan time.Duration, maxRows int32) ([]*db.StuckTransfer, error)

	TrashedTransfer(ctx context.Context, transferID int) (*db.Transfer, error)
	RestoreTransfer(ctx context.Context, transferID int) (*db.Transfer, error)
	DeleteTransferPermanent(ctx context.Context, transferID int) (bool, error)

	RestoreAllTransfer(ctx context.Context) (bool, error)
	DeleteAllTransferPermanent(ctx context.Context) (bool, error)
}
