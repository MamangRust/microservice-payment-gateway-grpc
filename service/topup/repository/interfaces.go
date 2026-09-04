package repository

import (
	"context"

	carddb "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	saldodb "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/database/schema"
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
	FindUserCardByCardNumber(ctx context.Context, card_number string) (*carddb.GetUserEmailByCardNumberRow, error)
	FindCardByCardNumber(ctx context.Context, card_number string) (*carddb.GetCardByCardNumberRow, error)
	UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*carddb.UpdateCardRow, error)
}
