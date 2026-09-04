package service

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

type SaldoQueryService interface {
	ListReconciliationQueue(ctx context.Context, status string, limit int32) ([]*db.ReconciliationQueueRow, error)
	ListLedgerEntries(ctx context.Context, cardNumber string, limit int32) ([]*db.LedgerEntry, error)
	FindAll(ctx context.Context, req *requests.FindAllSaldos) ([]*db.GetSaldosRow, *int, error)
	FindById(ctx context.Context, saldo_id int) (*db.GetSaldoByIDRow, error)
	FindByCardNumber(ctx context.Context, card_number string) (*db.Saldo, error)
	FindByActive(ctx context.Context, req *requests.FindAllSaldos) ([]*db.GetActiveSaldosRow, *int, error)
	FindByTrashed(ctx context.Context, req *requests.FindAllSaldos) ([]*db.GetTrashedSaldosRow, *int, error)
}

type SaldoCommandService interface {
	CreateSaldo(ctx context.Context, request *requests.CreateSaldoRequest) (*db.CreateSaldoRow, error)
	CreateSaldoIfNotExists(ctx context.Context, request *requests.CreateSaldoRequest) error
	UpdateSaldo(ctx context.Context, request *requests.UpdateSaldoRequest) (*db.UpdateSaldoRow, error)
	UpdateSaldoWithdraw(ctx context.Context, request *requests.UpdateSaldoWithdraw) (*db.UpdateSaldoWithdrawRow, error)
	ApplySaldoAdjustment(ctx context.Context, request *requests.ApplySaldoAdjustmentRequest) (*db.SaldoAdjustmentRow, error)
	ResolveReconciliation(ctx context.Context, queueID int64, operationID, note string) error
	DebitSaldo(ctx context.Context, request *requests.DebitSaldoRequest) (*db.DebitSaldoRow, error)
	CreditSaldo(ctx context.Context, request *requests.CreditSaldoRequest) (*db.CreditSaldoRow, error)
	TrashSaldo(ctx context.Context, saldo_id int) (*db.Saldo, error)
	RestoreSaldo(ctx context.Context, saldo_id int) (*db.Saldo, error)
	DeleteSaldoPermanent(ctx context.Context, saldo_id int) (bool, error)

	RestoreAllSaldo(ctx context.Context) (bool, error)
	DeleteAllSaldoPermanent(ctx context.Context) (bool, error)
}
