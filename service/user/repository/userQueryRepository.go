package repository

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	user_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/user_errors/repository"
)

// userQueryRepository implements UserQueryRepository.
type userQueryRepository struct {
	db *db.Queries
}

// NewUserQueryRepository creates a new UserQueryRepository.
func NewUserQueryRepository(db *db.Queries) UserQueryRepository {
	return &userQueryRepository{
		db: db,
	}
}

func (r *userQueryRepository) FindAllUsers(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetUsersWithPaginationRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetUsersWithPaginationParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetUsersWithPagination(ctx, reqDb)

	if err != nil {
		return nil, user_errors.ErrFindAllUsers.WithInternal(err)
	}

	return res, nil
}

func (r *userQueryRepository) FindById(ctx context.Context, user_id int) (*db.GetUserByIDRow, error) {
	res, err := r.db.GetUserByID(ctx, int32(user_id))

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user_errors.ErrUserNotFound.WithInternal(err)
		}

		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	return res, nil
}

func (r *userQueryRepository) FindByActive(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetActiveUsersWithPaginationRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetActiveUsersWithPaginationParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetActiveUsersWithPagination(ctx, reqDb)

	if err != nil {
		return nil, user_errors.ErrFindActiveUsers.WithInternal(err)
	}

	return res, nil
}

func (r *userQueryRepository) FindByTrashed(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetTrashedUsersWithPaginationRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetTrashedUsersWithPaginationParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetTrashedUsersWithPagination(ctx, reqDb)

	if err != nil {
		return nil, user_errors.ErrFindTrashedUsers.WithInternal(err)
	}

	return res, nil
}

func (r *userQueryRepository) FindByEmail(ctx context.Context, email string) (*db.GetUserByEmailRow, error) {
	res, err := r.db.GetUserByEmail(ctx, email)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user_errors.ErrUserNotFound.WithInternal(err)
		}

		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	return res, nil
}

func (r *userQueryRepository) FindByVerificationCode(ctx context.Context, code string) (*db.GetUserByVerificationCodeRow, error) {
	res, err := r.db.GetUserByVerificationCode(ctx, code)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user_errors.ErrUserNotFound.WithInternal(err)
		}

		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}

	return res, nil
}
