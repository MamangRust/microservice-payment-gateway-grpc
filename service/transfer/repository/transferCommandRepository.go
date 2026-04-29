package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	transfer_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/transfer_errors/repository"
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
		TransferAmount: int32(request.TransferAmount),
	}

	res, err := r.db.CreateTransfer(ctx, req)

	if err != nil {
		return nil, transfer_errors.ErrCreateTransferFailed.WithInternal(err)
	}

	return res, nil
}

func (r *transferCommandRepository) UpdateTransfer(ctx context.Context, request *requests.UpdateTransferRequest) (*db.UpdateTransferRow, error) {
	req := db.UpdateTransferParams{
		TransferID:     int32(*request.TransferID),
		TransferFrom:   request.TransferFrom,
		TransferTo:     request.TransferTo,
		TransferAmount: int32(request.TransferAmount),
	}

	res, err := r.db.UpdateTransfer(ctx, req)

	if err != nil {
		return nil, transfer_errors.ErrUpdateTransferFailed.WithInternal(err)
	}

	return res, nil

}

func (r *transferCommandRepository) UpdateTransferAmount(ctx context.Context, request *requests.UpdateTransferAmountRequest) (*db.UpdateTransferAmountRow, error) {
	req := db.UpdateTransferAmountParams{
		TransferID:     int32(request.TransferID),
		TransferAmount: int32(request.TransferAmount),
	}

	res, err := r.db.UpdateTransferAmount(ctx, req)

	if err != nil {
		return nil, transfer_errors.ErrUpdateTransferAmountFailed.WithInternal(err)
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
		return nil, transfer_errors.ErrUpdateTransferStatusFailed.WithInternal(err)
	}

	return res, nil
}

func (r *transferCommandRepository) TrashedTransfer(ctx context.Context, transfer_id int) (*db.Transfer, error) {
	res, err := r.db.TrashTransfer(ctx, int32(transfer_id))

	if err != nil {
		return nil, transfer_errors.ErrTrashedTransferFailed.WithInternal(err)
	}
	return res, nil
}

func (r *transferCommandRepository) RestoreTransfer(ctx context.Context, transfer_id int) (*db.Transfer, error) {
	res, err := r.db.RestoreTransfer(ctx, int32(transfer_id))
	if err != nil {
		return nil, transfer_errors.ErrRestoreTransferFailed.WithInternal(err)
	}
	return res, nil
}

func (r *transferCommandRepository) DeleteTransferPermanent(ctx context.Context, transfer_id int) (bool, error) {
	err := r.db.DeleteTransferPermanently(ctx, int32(transfer_id))
	if err != nil {
		return false, transfer_errors.ErrDeleteTransferPermanentFailed.WithInternal(err)
	}
	return true, nil
}

func (r *transferCommandRepository) RestoreAllTransfer(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllTransfers(ctx)

	if err != nil {
		return false, transfer_errors.ErrRestoreAllTransfersFailed.WithInternal(err)
	}

	return true, nil
}

func (r *transferCommandRepository) DeleteAllTransferPermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentTransfers(ctx)

	if err != nil {
		return false, transfer_errors.ErrDeleteAllTransfersPermanentFailed.WithInternal(err)
	}

	return true, nil
}
