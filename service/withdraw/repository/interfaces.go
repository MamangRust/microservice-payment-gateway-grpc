package repository

import (
	"context"

	carddb "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	saldodb "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/database/schema"
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
	UpdateSaldoWithdraw(ctx context.Context, request *requests.UpdateSaldoWithdraw) (*saldodb.UpdateSaldoWithdrawRow, error)
}

type WithdrawQueryRepository interface {
	FindAll(ctx context.Context, req *requests.FindAllWithdraws) ([]*db.GetWithdrawsRow, error)
	FindByActive(ctx context.Context, req *requests.FindAllWithdraws) ([]*db.GetActiveWithdrawsRow, error)
	FindByTrashed(ctx context.Context, req *requests.FindAllWithdraws) ([]*db.GetTrashedWithdrawsRow, error)
	FindAllByCardNumber(ctx context.Context, req *requests.FindAllWithdrawCardNumber) ([]*db.GetWithdrawsByCardNumberRow, error)
	FindById(ctx context.Context, id int) (*db.GetWithdrawByIDRow, error)
	GetTodayWithdrawSumByCardNumber(ctx context.Context, cardNumber string) (int64, error)
}

type WithdrawCommandRepository interface {
	CreateWithdraw(ctx context.Context, request *requests.CreateWithdrawRequest) (*db.CreateWithdrawRow, error)
	UpdateWithdraw(ctx context.Context, request *requests.UpdateWithdrawRequest) (*db.UpdateWithdrawRow, error)
	UpdateWithdrawStatus(ctx context.Context, request *requests.UpdateWithdrawStatus) (*db.UpdateWithdrawStatusRow, error)

	TrashedWithdraw(ctx context.Context, withdrawID int) (*db.Withdraw, error)
	RestoreWithdraw(ctx context.Context, withdrawID int) (*db.Withdraw, error)
	DeleteWithdrawPermanent(ctx context.Context, withdrawID int) (bool, error)

	RestoreAllWithdraw(ctx context.Context) (bool, error)
	DeleteAllWithdrawPermanent(ctx context.Context) (bool, error)
}

type CardRepository interface {
	FindUserCardByCardNumber(ctx context.Context, card_number string) (*carddb.GetUserEmailByCardNumberRow, error)
}
