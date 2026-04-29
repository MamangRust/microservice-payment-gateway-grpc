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

type TopupQueryRepository interface {
	FindAllTopups(ctx context.Context, req *requests.FindAllTopups) ([]*db.GetTopupsRow, error)
	FindByActive(ctx context.Context, req *requests.FindAllTopups) ([]*db.GetActiveTopupsRow, error)
	FindByTrashed(ctx context.Context, req *requests.FindAllTopups) ([]*db.GetTrashedTopupsRow, error)
	FindAllTopupByCardNumber(ctx context.Context, req *requests.FindAllTopupsByCardNumber) ([]*db.GetTopupsByCardNumberRow, error)

	FindById(ctx context.Context, topup_id int) (*db.GetTopupByIDRow, error)
}

type TopupCommandRepository interface {
	CreateTopup(ctx context.Context, request *requests.CreateTopupRequest) (*db.CreateTopupRow, error)
	UpdateTopup(ctx context.Context, request *requests.UpdateTopupRequest) (*db.UpdateTopupRow, error)

	UpdateTopupAmount(ctx context.Context, request *requests.UpdateTopupAmount) (*db.UpdateTopupAmountRow, error)
	UpdateTopupStatus(ctx context.Context, request *requests.UpdateTopupStatus) (*db.UpdateTopupStatusRow, error)

	TrashedTopup(ctx context.Context, topup_id int) (*db.Topup, error)
	RestoreTopup(ctx context.Context, topup_id int) (*db.Topup, error)
	DeleteTopupPermanent(ctx context.Context, topup_id int) (bool, error)

	RestoreAllTopup(ctx context.Context) (bool, error)
	DeleteAllTopupPermanent(ctx context.Context) (bool, error)
}

type CardRepository interface {
	FindUserCardByCardNumber(ctx context.Context, card_number string) (*db.GetUserEmailByCardNumberRow, error)
	FindCardByCardNumber(ctx context.Context, card_number string) (*db.GetCardByCardNumberRow, error)
	UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*db.UpdateCardRow, error)
}
