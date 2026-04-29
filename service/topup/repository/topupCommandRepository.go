package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	topup_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/topup_errors/repository"
)

type topupCommandRepository struct {
	db *db.Queries
}

func NewTopupCommandRepository(db *db.Queries) TopupCommandRepository {
	return &topupCommandRepository{
		db: db,
	}
}

func (r *topupCommandRepository) CreateTopup(ctx context.Context, request *requests.CreateTopupRequest) (*db.CreateTopupRow, error) {
	req := db.CreateTopupParams{
		CardNumber:  request.CardNumber,
		TopupAmount: int32(request.TopupAmount),
		TopupMethod: request.TopupMethod,
	}

	res, err := r.db.CreateTopup(ctx, req)

	if err != nil {
		return nil, topup_errors.ErrCreateTopupFailed.WithInternal(err)
	}

	return res, nil
}

func (r *topupCommandRepository) UpdateTopup(ctx context.Context, request *requests.UpdateTopupRequest) (*db.UpdateTopupRow, error) {
	req := db.UpdateTopupParams{
		TopupID:     int32(*request.TopupID),
		CardNumber:  request.CardNumber,
		TopupAmount: int32(request.TopupAmount),
		TopupMethod: request.TopupMethod,
	}

	res, err := r.db.UpdateTopup(ctx, req)

	if err != nil {
		return nil, topup_errors.ErrUpdateTopupFailed.WithInternal(err)
	}

	return res, nil
}

func (r *topupCommandRepository) UpdateTopupAmount(ctx context.Context, request *requests.UpdateTopupAmount) (*db.UpdateTopupAmountRow, error) {
	req := db.UpdateTopupAmountParams{
		TopupID:     int32(request.TopupID),
		TopupAmount: int32(request.TopupAmount),
	}

	res, err := r.db.UpdateTopupAmount(ctx, req)

	if err != nil {
		return nil, topup_errors.ErrUpdateTopupAmountFailed.WithInternal(err)
	}

	return res, nil
}

func (r *topupCommandRepository) UpdateTopupStatus(ctx context.Context, request *requests.UpdateTopupStatus) (*db.UpdateTopupStatusRow, error) {
	req := db.UpdateTopupStatusParams{
		TopupID: int32(request.TopupID),
		Status:  request.Status,
	}

	res, err := r.db.UpdateTopupStatus(ctx, req)

	if err != nil {
		return nil, topup_errors.ErrUpdateTopupStatusFailed.WithInternal(err)
	}

	return res, nil
}

func (r *topupCommandRepository) TrashedTopup(ctx context.Context, topup_id int) (*db.Topup, error) {
	res, err := r.db.TrashTopup(ctx, int32(topup_id))
	if err != nil {
		return nil, topup_errors.ErrTrashedTopupFailed.WithInternal(err)
	}
	return res, nil
}

func (r *topupCommandRepository) RestoreTopup(ctx context.Context, topup_id int) (*db.Topup, error) {
	res, err := r.db.RestoreTopup(ctx, int32(topup_id))
	if err != nil {
		return nil, topup_errors.ErrRestoreTopupFailed.WithInternal(err)
	}
	return res, nil
}

func (r *topupCommandRepository) DeleteTopupPermanent(ctx context.Context, topup_id int) (bool, error) {
	err := r.db.DeleteTopupPermanently(ctx, int32(topup_id))
	if err != nil {
		return false, topup_errors.ErrDeleteTopupPermanentFailed.WithInternal(err)
	}
	return true, nil
}

func (r *topupCommandRepository) RestoreAllTopup(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllTopups(ctx)

	if err != nil {
		return false, topup_errors.ErrRestoreAllTopupFailed.WithInternal(err)
	}

	return true, nil
}

func (r *topupCommandRepository) DeleteAllTopupPermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentTopups(ctx)

	if err != nil {
		return false, topup_errors.ErrDeleteAllTopupPermanentFailed.WithInternal(err)
	}

	return true, nil
}
