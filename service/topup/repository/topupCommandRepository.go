package repository

import (
	"context"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
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
		TopupAmount: int64(request.TopupAmount),
		TopupMethod: request.TopupMethod,
	}

	res, err := r.db.CreateTopup(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrFailed("create topup").WithInternal(err)
	}

	return res, nil
}

func (r *topupCommandRepository) UpdateTopup(ctx context.Context, request *requests.UpdateTopupRequest) (*db.UpdateTopupRow, error) {
	req := db.UpdateTopupParams{
		TopupID:     int32(*request.TopupID),
		CardNumber:  request.CardNumber,
		TopupAmount: int64(request.TopupAmount),
		TopupMethod: request.TopupMethod,
	}

	res, err := r.db.UpdateTopup(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "topup", "update topup")
	}

	return res, nil
}

func (r *topupCommandRepository) UpdateTopupAmount(ctx context.Context, request *requests.UpdateTopupAmount) (*db.UpdateTopupAmountRow, error) {
	req := db.UpdateTopupAmountParams{
		TopupID:     int32(request.TopupID),
		TopupAmount: int64(request.TopupAmount),
	}

	res, err := r.db.UpdateTopupAmount(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "topup", "update topup amount")
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
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "topup", "update topup status")
	}

	return res, nil
}

func (r *topupCommandRepository) TrashedTopup(ctx context.Context, topup_id int) (*db.Topup, error) {
	res, err := r.db.TrashTopup(ctx, int32(topup_id))
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "topup", "trash topup")
	}
	return res, nil
}

func (r *topupCommandRepository) RestoreTopup(ctx context.Context, topup_id int) (*db.Topup, error) {
	res, err := r.db.RestoreTopup(ctx, int32(topup_id))
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "topup", "restore topup")
	}
	return res, nil
}

func (r *topupCommandRepository) DeleteTopupPermanent(ctx context.Context, topup_id int) (bool, error) {
	err := r.db.DeleteTopupPermanently(ctx, int32(topup_id))
	if err != nil {
		return false, sharedErrors.ErrNoRowsOrFailed(err, "topup", "delete topup permanently")
	}
	return true, nil
}

func (r *topupCommandRepository) RestoreAllTopup(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllTopups(ctx)

	if err != nil {
		return false, sharedErrors.ErrFailed("restore all topups").WithInternal(err)
	}

	return true, nil
}

func (r *topupCommandRepository) DeleteAllTopupPermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentTopups(ctx)

	if err != nil {
		return false, sharedErrors.ErrFailed("delete all topups permanently").WithInternal(err)
	}

	return true, nil
}
