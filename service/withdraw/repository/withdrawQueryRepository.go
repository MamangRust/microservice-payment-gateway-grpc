package repository

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	withdraw_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/withdraw_errors/repository"
)

type withdrawQueryRepository struct {
	db *db.Queries
}

func NewWithdrawQueryRepository(db *db.Queries) WithdrawQueryRepository {
	return &withdrawQueryRepository{
		db: db,
	}
}

func (r *withdrawQueryRepository) FindAll(ctx context.Context, req *requests.FindAllWithdraws) ([]*db.GetWithdrawsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetWithdrawsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	withdraw, err := r.db.GetWithdraws(ctx, reqDb)

	if err != nil {
		return nil, withdraw_errors.ErrFindAllWithdrawsFailed.WithInternal(err)
	}

	return withdraw, nil

}

func (r *withdrawQueryRepository) FindByActive(ctx context.Context, req *requests.FindAllWithdraws) ([]*db.GetActiveWithdrawsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetActiveWithdrawsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetActiveWithdraws(ctx, reqDb)

	if err != nil {
		return nil, withdraw_errors.ErrFindActiveWithdrawsFailed.WithInternal(err)
	}

	return res, nil
}

func (r *withdrawQueryRepository) FindByTrashed(ctx context.Context, req *requests.FindAllWithdraws) ([]*db.GetTrashedWithdrawsRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetTrashedWithdrawsParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetTrashedWithdraws(ctx, reqDb)

	if err != nil {
		return nil, withdraw_errors.ErrFindTrashedWithdrawsFailed.WithInternal(err)
	}

	return res, nil
}

func (r *withdrawQueryRepository) FindAllByCardNumber(ctx context.Context, req *requests.FindAllWithdrawCardNumber) ([]*db.GetWithdrawsByCardNumberRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetWithdrawsByCardNumberParams{
		CardNumber: req.CardNumber,
		Column2:    req.Search,
		Limit:      int32(req.PageSize),
		Offset:     int32(offset),
	}

	withdraw, err := r.db.GetWithdrawsByCardNumber(ctx, reqDb)

	if err != nil {
		return nil, withdraw_errors.ErrFindWithdrawsByCardNumberFailed.WithInternal(err)
	}

	return withdraw, nil

}

func (r *withdrawQueryRepository) FindById(ctx context.Context, id int) (*db.GetWithdrawByIDRow, error) {
	withdraw, err := r.db.GetWithdrawByID(ctx, int32(id))

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, withdraw_errors.ErrFindWithdrawByIdFailed.WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	return withdraw, nil
}
