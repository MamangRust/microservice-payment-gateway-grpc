package repository

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	role_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/role_errors/repository"
)

// roleQueryRepository is a struct that implements the RoleQueryRepository interface
type roleQueryRepository struct {
	db *db.Queries
}

func NewRoleQueryRepository(db *db.Queries) RoleQueryRepository {
	return &roleQueryRepository{
		db: db,
	}
}

func (r *roleQueryRepository) FindAllRoles(ctx context.Context, req *requests.FindAllRoles) ([]*db.GetRolesRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetRolesParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetRoles(ctx, reqDb)

	if err != nil {
		return nil, role_errors.ErrFindAllRoles.WithInternal(err)
	}

	return res, nil
}

func (r *roleQueryRepository) FindById(ctx context.Context, id int) (*db.Role, error) {
	res, err := r.db.GetRole(ctx, int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, role_errors.ErrRoleNotFound.WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}
	return res, nil
}

func (r *roleQueryRepository) FindByName(ctx context.Context, name string) (*db.Role, error) {
	res, err := r.db.GetRoleByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, role_errors.ErrRoleNotFound.WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}
	return res, nil
}

func (r *roleQueryRepository) FindByUserId(ctx context.Context, user_id int) ([]*db.Role, error) {
	res, err := r.db.GetUserRoles(ctx, int32(user_id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, role_errors.ErrRoleNotFound.WithInternal(err)
		}

		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}
	return res, nil
}

func (r *roleQueryRepository) FindByActiveRole(ctx context.Context, req *requests.FindAllRoles) ([]*db.GetActiveRolesRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetActiveRolesParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetActiveRoles(ctx, reqDb)

	if err != nil {
		return nil, role_errors.ErrFindActiveRoles.WithInternal(err)
	}

	return res, nil
}

func (r *roleQueryRepository) FindByTrashedRole(ctx context.Context, req *requests.FindAllRoles) ([]*db.GetTrashedRolesRow, error) {
	offset := (req.Page - 1) * req.PageSize

	reqDb := db.GetTrashedRolesParams{
		Column1: req.Search,
		Limit:   int32(req.PageSize),
		Offset:  int32(offset),
	}

	res, err := r.db.GetTrashedRoles(ctx, reqDb)

	if err != nil {
		return nil, role_errors.ErrFindTrashedRoles.WithInternal(err)
	}

	return res, nil
}
