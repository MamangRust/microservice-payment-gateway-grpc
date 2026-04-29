package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

type SaldoRepository interface {
	FindByCardNumber(ctx context.Context, card_number string) (*db.Saldo, error)

	UpdateSaldoBalance(ctx context.Context, request *requests.UpdateSaldoBalance) (*db.UpdateSaldoBalanceRow, error)
}

type CardRepository interface {
	FindUserCardByCardNumber(ctx context.Context, card_number string) (*db.GetUserEmailByCardNumberRow, error)

	FindCardByCardNumber(ctx context.Context, card_number string) (*db.GetCardByCardNumberRow, error)
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

	TrashedTransfer(ctx context.Context, transferID int) (*db.Transfer, error)
	RestoreTransfer(ctx context.Context, transferID int) (*db.Transfer, error)
	DeleteTransferPermanent(ctx context.Context, transferID int) (bool, error)

	RestoreAllTransfer(ctx context.Context) (bool, error)
	DeleteAllTransferPermanent(ctx context.Context) (bool, error)
}
