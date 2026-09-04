package repository

import (
	"context"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

type withdrawCommandRepository struct {
	db *db.Queries
}

func NewWithdrawCommandRepository(db *db.Queries) WithdrawCommandRepository {
	return &withdrawCommandRepository{
		db: db,
	}
}

func (r *withdrawCommandRepository) CreateWithdraw(ctx context.Context, request *requests.CreateWithdrawRequest) (*db.CreateWithdrawRow, error) {
	req := db.CreateWithdrawParams{
		CardNumber:     request.CardNumber,
		WithdrawAmount: int64(request.WithdrawAmount),
		WithdrawTime:   request.WithdrawTime,
	}

	res, err := r.db.CreateWithdraw(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrFailed("create withdraw").WithInternal(err)
	}

	return res, nil
}

func (r *withdrawCommandRepository) UpdateWithdraw(ctx context.Context, request *requests.UpdateWithdrawRequest) (*db.UpdateWithdrawRow, error) {
	req := db.UpdateWithdrawParams{
		WithdrawID:     int32(*request.WithdrawID),
		CardNumber:     request.CardNumber,
		WithdrawAmount: int64(request.WithdrawAmount),
		WithdrawTime:   request.WithdrawTime,
	}

	res, err := r.db.UpdateWithdraw(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "withdraw", "update withdraw")
	}

	return res, nil
}

func (r *withdrawCommandRepository) UpdateWithdrawStatus(ctx context.Context, request *requests.UpdateWithdrawStatus) (*db.UpdateWithdrawStatusRow, error) {
	req := db.UpdateWithdrawStatusParams{
		WithdrawID: int32(request.WithdrawID),
		Status:     request.Status,
	}

	res, err := r.db.UpdateWithdrawStatus(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "withdraw", "update withdraw status")
	}

	return res, nil
}

func (r *withdrawCommandRepository) TrashedWithdraw(ctx context.Context, withdraw_id int) (*db.Withdraw, error) {
	res, err := r.db.TrashWithdraw(ctx, int32(withdraw_id))

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "withdraw", "trash withdraw")
	}

	return res, nil
}

func (r *withdrawCommandRepository) RestoreWithdraw(ctx context.Context, withdraw_id int) (*db.Withdraw, error) {
	res, err := r.db.RestoreWithdraw(ctx, int32(withdraw_id))

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "withdraw", "restore withdraw")
	}

	return res, nil
}

func (r *withdrawCommandRepository) DeleteWithdrawPermanent(ctx context.Context, withdraw_id int) (bool, error) {
	err := r.db.DeleteWithdrawPermanently(ctx, int32(withdraw_id))

	if err != nil {
		return false, sharedErrors.ErrNoRowsOrFailed(err, "withdraw", "delete withdraw permanently")
	}

	return true, nil
}

func (r *withdrawCommandRepository) RestoreAllWithdraw(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllWithdraws(ctx)

	if err != nil {
		return false, sharedErrors.ErrFailed("restore all withdraws").WithInternal(err)
	}

	return true, nil
}

func (r *withdrawCommandRepository) DeleteAllWithdrawPermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentWithdraws(ctx)

	if err != nil {
		return false, sharedErrors.ErrFailed("delete all withdraws permanently").WithInternal(err)
	}

	return true, nil
}
