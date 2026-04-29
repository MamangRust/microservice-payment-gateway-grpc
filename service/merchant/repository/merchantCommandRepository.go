package repository

import (
	"context"

	apikey "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/api-key"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	merchant_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/merchant_errors/repository"
)

type merchantCommandRepository struct {
	db *db.Queries
}

func NewMerchantCommandRepository(db *db.Queries) MerchantCommandRepository {
	return &merchantCommandRepository{
		db: db,
	}
}

func (r *merchantCommandRepository) CreateMerchant(ctx context.Context, request *requests.CreateMerchantRequest) (*db.CreateMerchantRow, error) {
	apiKey, err := apikey.GenerateApiKey()

	if err != nil {
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	req := db.CreateMerchantParams{
		Name:   request.Name,
		ApiKey: apiKey,
		UserID: int32(request.UserID),
		Status: "inactive",
	}

	res, err := r.db.CreateMerchant(ctx, req)

	if err != nil {
		return nil, merchant_errors.ErrCreateMerchantFailed.WithInternal(err)
	}

	return res, nil
}

func (r *merchantCommandRepository) UpdateMerchant(ctx context.Context, request *requests.UpdateMerchantRequest) (*db.UpdateMerchantRow, error) {
	req := db.UpdateMerchantParams{
		MerchantID: int32(*request.MerchantID),
		Name:       request.Name,
		UserID:     int32(request.UserID),
		Status:     request.Status,
	}

	res, err := r.db.UpdateMerchant(ctx, req)

	if err != nil {
		return nil, merchant_errors.ErrUpdateMerchantFailed.WithInternal(err)
	}

	return res, nil
}

func (r *merchantCommandRepository) UpdateMerchantStatus(ctx context.Context, request *requests.UpdateMerchantStatusRequest) (*db.UpdateMerchantStatusRow, error) {
	req := db.UpdateMerchantStatusParams{
		MerchantID: int32(*request.MerchantID),
		Status:     request.Status,
	}

	res, err := r.db.UpdateMerchantStatus(ctx, req)

	if err != nil {
		return nil, merchant_errors.ErrUpdateMerchantStatusFailed.WithInternal(err)
	}

	return res, nil
}

func (r *merchantCommandRepository) TrashedMerchant(ctx context.Context, merchant_id int) (*db.Merchant, error) {
	res, err := r.db.TrashMerchant(ctx, int32(merchant_id))

	if err != nil {
		return nil, merchant_errors.ErrTrashedMerchantFailed.WithInternal(err)
	}

	return res, nil
}

func (r *merchantCommandRepository) RestoreMerchant(ctx context.Context, merchant_id int) (*db.Merchant, error) {
	res, err := r.db.RestoreMerchant(ctx, int32(merchant_id))

	if err != nil {
		return nil, merchant_errors.ErrRestoreMerchantFailed.WithInternal(err)
	}

	return res, nil
}

func (r *merchantCommandRepository) DeleteMerchantPermanent(ctx context.Context, merchant_id int) (bool, error) {
	err := r.db.DeleteMerchantPermanently(ctx, int32(merchant_id))

	if err != nil {
		return false, merchant_errors.ErrDeleteMerchantPermanentFailed.WithInternal(err)
	}

	return true, nil
}

func (r *merchantCommandRepository) RestoreAllMerchant(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllMerchants(ctx)

	if err != nil {
		return false, merchant_errors.ErrRestoreAllMerchantFailed.WithInternal(err)
	}

	return true, nil
}

func (r *merchantCommandRepository) DeleteAllMerchantPermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentMerchants(ctx)

	if err != nil {
		return false, merchant_errors.ErrDeleteAllMerchantPermanentFailed.WithInternal(err)
	}

	return true, nil
}
