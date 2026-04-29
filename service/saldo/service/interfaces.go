package service

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

type SaldoQueryService interface {
	FindAll(ctx context.Context, req *requests.FindAllSaldos) ([]*db.GetSaldosRow, *int, error)
	FindById(ctx context.Context, saldo_id int) (*db.GetSaldoByIDRow, error)
	FindByCardNumber(ctx context.Context, card_number string) (*db.Saldo, error)
	FindByActive(ctx context.Context, req *requests.FindAllSaldos) ([]*db.GetActiveSaldosRow, *int, error)
	FindByTrashed(ctx context.Context, req *requests.FindAllSaldos) ([]*db.GetTrashedSaldosRow, *int, error)
}

type SaldoCommandService interface {
	CreateSaldo(ctx context.Context, request *requests.CreateSaldoRequest) (*db.CreateSaldoRow, error)
	UpdateSaldo(ctx context.Context, request *requests.UpdateSaldoRequest) (*db.UpdateSaldoRow, error)
	UpdateSaldoWithdraw(ctx context.Context, request *requests.UpdateSaldoWithdraw) (*db.UpdateSaldoWithdrawRow, error)
	TrashSaldo(ctx context.Context, saldo_id int) (*db.Saldo, error)
	RestoreSaldo(ctx context.Context, saldo_id int) (*db.Saldo, error)
	DeleteSaldoPermanent(ctx context.Context, saldo_id int) (bool, error)

	RestoreAllSaldo(ctx context.Context) (bool, error)
	DeleteAllSaldoPermanent(ctx context.Context) (bool, error)
}
