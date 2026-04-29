package repository

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	merchant_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/merchant_errors/repository"
)

type merchantQueryRepository struct {
	db *db.Queries
}

func NewMerchantQueryRepository(db *db.Queries) MerchantQueryRepository {
	return &merchantQueryRepository{
		db: db,
	}
}

func (r *merchantQueryRepository) FindAllMerchants(ctx context.Context, req *requests.FindAllMerchants) ([]*db.GetMerchantsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetMerchantsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	merchant, err := r.db.GetMerchants(ctx, reqDb)

	if err != nil {
		return nil, merchant_errors.ErrFindAllMerchantsFailed.WithInternal(err)
	}

	return merchant, nil
}

func (r *merchantQueryRepository) FindByActive(ctx context.Context, req *requests.FindAllMerchants) ([]*db.GetActiveMerchantsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetActiveMerchantsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetActiveMerchants(ctx, reqDb)

	if err != nil {
		return nil, merchant_errors.ErrFindActiveMerchantsFailed.WithInternal(err)
	}

	return res, nil
}

func (r *merchantQueryRepository) FindByTrashed(ctx context.Context, req *requests.FindAllMerchants) ([]*db.GetTrashedMerchantsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetTrashedMerchantsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetTrashedMerchants(ctx, reqDb)

	if err != nil {
		return nil, merchant_errors.ErrFindTrashedMerchantsFailed.WithInternal(err)
	}

	return res, nil
}

func (r *merchantQueryRepository) FindByMerchantId(ctx context.Context, merchant_id int) (*db.GetMerchantByIDRow, error) {
	res, err := r.db.GetMerchantByID(ctx, int32(merchant_id))

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, merchant_errors.ErrFindMerchantByIdFailed.WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	return res, nil
}

func (r *merchantQueryRepository) FindByApiKey(ctx context.Context, api_key string) (*db.GetMerchantByApiKeyRow, error) {
	res, err := r.db.GetMerchantByApiKey(ctx, api_key)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, merchant_errors.ErrFindMerchantByApiKeyFailed.WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	return res, nil
}

func (r *merchantQueryRepository) FindByName(ctx context.Context, name string) (*db.GetMerchantByNameRow, error) {
	res, err := r.db.GetMerchantByName(ctx, name)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, merchant_errors.ErrFindMerchantByNameFailed.WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	return res, nil
}

func (r *merchantQueryRepository) FindByMerchantUserId(ctx context.Context, user_id int) ([]*db.GetMerchantsByUserIDRow, error) {
	res, err := r.db.GetMerchantsByUserID(ctx, int32(user_id))

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, merchant_errors.ErrFindMerchantByUserIdFailed.WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	return res, nil
}
