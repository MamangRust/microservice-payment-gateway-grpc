package repository

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

type saldoQueryRepository struct {
	db *db.Queries
}

func NewSaldoQueryRepository(db *db.Queries) SaldoQueryRepository {
	return &saldoQueryRepository{
		db: db,
	}
}

func (r *saldoQueryRepository) ListReconciliationQueue(ctx context.Context, status string, limit int32) ([]*db.ReconciliationQueueRow, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	res, err := r.db.ListReconciliationQueue(ctx, status, limit)
	if err != nil {
		return nil, sharedErrors.ErrFailed("list reconciliation queue").WithInternal(err)
	}
	return res, nil
}

func (r *saldoQueryRepository) ListLedgerEntries(ctx context.Context, cardNumber string, limit int32) ([]*db.LedgerEntry, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	res, err := r.db.ListLedgerEntries(ctx, cardNumber, limit)
	if err != nil {
		return nil, sharedErrors.ErrFailed("list ledger entries").WithInternal(err)
	}
	return res, nil
}

func (r *saldoQueryRepository) FindAllSaldos(ctx context.Context, req *requests.FindAllSaldos) ([]*db.GetSaldosRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetSaldosParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	saldos, err := r.db.GetSaldos(ctx, reqDb)

	if err != nil {
		return nil, sharedErrors.ErrFailed("find all saldo records").WithInternal(err)
	}

	return saldos, nil
}

func (r *saldoQueryRepository) FindByActive(ctx context.Context, req *requests.FindAllSaldos) ([]*db.GetActiveSaldosRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetActiveSaldosParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetActiveSaldos(ctx, reqDb)

	if err != nil {
		return nil, sharedErrors.ErrFailed("find active saldo records").WithInternal(err)
	}

	return res, nil
}

func (r *saldoQueryRepository) FindByTrashed(ctx context.Context, req *requests.FindAllSaldos) ([]*db.GetTrashedSaldosRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetTrashedSaldosParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	saldos, err := r.db.GetTrashedSaldos(ctx, reqDb)

	if err != nil {
		return nil, sharedErrors.ErrFailed("find trashed saldo records").WithInternal(err)
	}

	return saldos, nil
}

func (r *saldoQueryRepository) FindByCardNumber(ctx context.Context, card_number string) (*db.Saldo, error) {
	res, err := r.db.GetSaldoByCardNumber(ctx, card_number)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sharedErrors.ErrNotFoundResponse("saldo record").WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	return res, nil
}

func (r *saldoQueryRepository) FindById(ctx context.Context, saldo_id int) (*db.GetSaldoByIDRow, error) {
	res, err := r.db.GetSaldoByID(ctx, int32(saldo_id))

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sharedErrors.ErrNotFoundResponse("saldo record").WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	return res, nil
}
