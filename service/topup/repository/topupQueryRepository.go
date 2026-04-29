package repository

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	topup_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/topup_errors/repository"
)

type topupQueryRepository struct {
	db *db.Queries
}

func NewTopupQueryRepository(db *db.Queries) TopupQueryRepository {
	return &topupQueryRepository{
		db: db,
	}
}

func (r *topupQueryRepository) FindAllTopups(ctx context.Context, req *requests.FindAllTopups) ([]*db.GetTopupsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetTopupsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetTopups(ctx, reqDb)

	if err != nil {
		return nil, topup_errors.ErrFindAllTopupsFailed.WithInternal(err)
	}

	return res, nil
}

func (r *topupQueryRepository) FindByActive(ctx context.Context, req *requests.FindAllTopups) ([]*db.GetActiveTopupsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetActiveTopupsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetActiveTopups(ctx, reqDb)

	if err != nil {
		return nil, topup_errors.ErrFindTopupsByActiveFailed.WithInternal(err)
	}

	return res, nil
}

func (r *topupQueryRepository) FindByTrashed(ctx context.Context, req *requests.FindAllTopups) ([]*db.GetTrashedTopupsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetTrashedTopupsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetTrashedTopups(ctx, reqDb)

	if err != nil {
		return nil, topup_errors.ErrFindTopupsByTrashedFailed.WithInternal(err)
	}

	return res, nil
}

func (r *topupQueryRepository) FindAllTopupByCardNumber(ctx context.Context, req *requests.FindAllTopupsByCardNumber) ([]*db.GetTopupsByCardNumberRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetTopupsByCardNumberParams{
		CardNumber: req.CardNumber,
		Column2:    req.Search,
		Limit:      int32(req.PageSize),
		Offset:     int32(offset),
	}

	res, err := r.db.GetTopupsByCardNumber(ctx, reqDb)

	if err != nil {
		return nil, topup_errors.ErrFindTopupsByCardNumberFailed.WithInternal(err)
	}

	return res, nil
}

func (r *topupQueryRepository) FindById(ctx context.Context, topup_id int) (*db.GetTopupByIDRow, error) {
	res, err := r.db.GetTopupByID(ctx, int32(topup_id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, topup_errors.ErrFindTopupByIdFailed.WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}
	return res, nil
}
