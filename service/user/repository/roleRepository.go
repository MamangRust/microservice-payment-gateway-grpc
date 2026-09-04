package repository

import (
	"context"
	"database/sql"
	"errors"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/database/schema"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

// roleRepository implements RoleRepository.
type roleRepository struct {
	db *db.Queries
}

// NewRoleRepository creates a new RoleRepository.
func NewRoleRepository(db *db.Queries) RoleRepository {
	return &roleRepository{
		db: db,
	}
}

func (r *roleRepository) FindById(ctx context.Context, id int) (*db.Role, error) {
	res, err := r.db.GetRole(ctx, int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sharedErrors.ErrNotFound.WithMessage("role not found").WithInternal(err)
		}
		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}
	return res, nil
}

func (r *roleRepository) FindByName(ctx context.Context, name string) (*db.Role, error) {
	res, err := r.db.GetRoleByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sharedErrors.ErrNotFound.WithMessage("role not found").WithInternal(err)
		}

		return nil, sharedErrors.ErrInternal.WithInternal(err)
	}
	return res, nil
}
