package repository

import (
	"context"
	"errors"
	"time"

	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/state"
	"github.com/jackc/pgx/v5"
)

type transferCommandRepository struct {
	db *db.Queries
}

func NewTransferCommandRepository(db *db.Queries) TransferCommandRepository {
	return &transferCommandRepository{
		db: db,
	}
}

func (r *transferCommandRepository) CreateTransfer(ctx context.Context, request *requests.CreateTransferRequest) (*db.CreateTransferRow, error) {
	req := db.CreateTransferParams{
		TransferFrom:   request.TransferFrom,
		TransferTo:     request.TransferTo,
		TransferAmount: int64(request.TransferAmount),
		TransferTime:   time.Now(),
		Status:         state.Pending,
	}

	res, err := r.db.CreateTransfer(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrFailed("create transfer").WithInternal(err)
	}

	return res, nil
}

func (r *transferCommandRepository) UpdateTransfer(ctx context.Context, request *requests.UpdateTransferRequest) (*db.UpdateTransferRow, error) {
	req := db.UpdateTransferParams{
		TransferID:     int32(*request.TransferID),
		TransferFrom:   request.TransferFrom,
		TransferTo:     request.TransferTo,
		TransferAmount: int64(request.TransferAmount),
	}

	res, err := r.db.UpdateTransfer(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "transfer", "update transfer")
	}

	return res, nil

}

func (r *transferCommandRepository) UpdateTransferAmount(ctx context.Context, request *requests.UpdateTransferAmountRequest) (*db.UpdateTransferAmountRow, error) {
	req := db.UpdateTransferAmountParams{
		TransferID:     int32(request.TransferID),
		TransferAmount: int64(request.TransferAmount),
	}

	res, err := r.db.UpdateTransferAmount(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "transfer", "update transfer amount")
	}

	return res, nil
}

func (r *transferCommandRepository) UpdateTransferStatus(ctx context.Context, request *requests.UpdateTransferStatus) (*db.UpdateTransferStatusRow, error) {
	req := db.UpdateTransferStatusParams{
		TransferID: int32(request.TransferID),
		Status:     request.Status,
	}

	res, err := r.db.UpdateTransferStatus(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "transfer", "update transfer status")
	}

	return res, nil
}

// GuardStatus atomically transitions a transfer from an expected status to a new
// one. It reports false (with nil error) when the row is not in the expected
// status, i.e. another actor already moved it.
func (r *transferCommandRepository) GuardStatus(ctx context.Context, id int, fromStatus, toStatus, reason string) (bool, error) {
	if err := state.CheckTransition(fromStatus, toStatus); err != nil {
		return false, err
	}
	_, err := r.db.GuardTransferStatus(ctx, int32(id), fromStatus, toStatus, reason)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListStuck returns transfers stuck in a recoverable state for the recovery worker.
func (r *transferCommandRepository) TransitionStatus(ctx context.Context, id int, fromStatus, toStatus, reason string) (*db.UpdateTransferStatusRow, error) {
	if err := state.CheckTransition(fromStatus, toStatus); err != nil {
		return nil, err
	}
	return r.db.GuardTransferStatus(ctx, int32(id), fromStatus, toStatus, reason)
}

func (r *transferCommandRepository) ListStuck(ctx context.Context, olderThan time.Duration, maxRows int32) ([]*db.StuckTransfer, error) {
	return r.db.ListStuckTransfers(ctx, olderThan, maxRows)
}

func (r *transferCommandRepository) TrashedTransfer(ctx context.Context, transfer_id int) (*db.Transfer, error) {
	res, err := r.db.TrashTransfer(ctx, int32(transfer_id))

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "transfer", "trash transfer")
	}
	return res, nil
}

func (r *transferCommandRepository) RestoreTransfer(ctx context.Context, transfer_id int) (*db.Transfer, error) {
	res, err := r.db.RestoreTransfer(ctx, int32(transfer_id))

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "transfer", "restore transfer")
	}
	return res, nil
}

func (r *transferCommandRepository) DeleteTransferPermanent(ctx context.Context, transfer_id int) (bool, error) {
	err := r.db.DeleteTransferPermanently(ctx, int32(transfer_id))
	if err != nil {
		return false, sharedErrors.ErrNoRowsOrFailed(err, "transfer", "delete transfer permanently")
	}
	return true, nil
}

func (r *transferCommandRepository) RestoreAllTransfer(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllTransfers(ctx)

	if err != nil {
		return false, sharedErrors.ErrFailed("restore all transfers").WithInternal(err)
	}

	return true, nil
}

func (r *transferCommandRepository) DeleteAllTransferPermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentTransfers(ctx)

	if err != nil {
		return false, sharedErrors.ErrFailed("delete all transfers permanently").WithInternal(err)
	}

	return true, nil
}
