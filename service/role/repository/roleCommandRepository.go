package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	role_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/role_errors/repository"
)

// roleCommandRepository is a struct that implements the RoleCommandRepository interface
type roleCommandRepository struct {
	db *db.Queries
}

// NewRoleCommandRepository creates a new RoleCommandRepository instance with the provided
// database queries, context, and role record mapper. This repository is responsible for
// executing command operations related to role records in the database.
//
// Parameters:
//   - db: A pointer to the db.Queries object for executing database queries.
//   - mapper: A RoleRecordMapping that provides methods to map database rows to Role domain models.
//
// Returns:
//   - A pointer to the newly created roleCommandRepository instance.
func NewRoleCommandRepository(db *db.Queries) RoleCommandRepository {
	return &roleCommandRepository{
		db: db,
	}
}

func (r *roleCommandRepository) CreateRole(ctx context.Context, req *requests.CreateRoleRequest) (*db.Role, error) {
	res, err := r.db.CreateRole(ctx, req.Name)

	if err != nil {
		return nil, role_errors.ErrCreateRole.WithInternal(err)
	}

	return res, nil
}

func (r *roleCommandRepository) UpdateRole(ctx context.Context, req *requests.UpdateRoleRequest) (*db.Role, error) {
	res, err := r.db.UpdateRole(ctx, db.UpdateRoleParams{
		RoleID:   int32(*req.ID),
		RoleName: req.Name,
	})

	if err != nil {
		return nil, role_errors.ErrUpdateRole.WithInternal(err)
	}

	return res, nil
}

func (r *roleCommandRepository) TrashedRole(ctx context.Context, id int) (*db.Role, error) {
	res, err := r.db.TrashRole(ctx, int32(id))
	if err != nil {
		return nil, role_errors.ErrTrashedRole.WithInternal(err)
	}
	return res, nil
}

func (r *roleCommandRepository) RestoreRole(ctx context.Context, id int) (*db.Role, error) {
	res, err := r.db.RestoreRole(ctx, int32(id))
	if err != nil {
		return nil, role_errors.ErrRestoreRole.WithInternal(err)
	}
	return res, nil
}

func (r *roleCommandRepository) DeleteRolePermanent(ctx context.Context, role_id int) (bool, error) {
	err := r.db.DeletePermanentRole(ctx, int32(role_id))
	if err != nil {
		return false, role_errors.ErrDeleteRolePermanent.WithInternal(err)
	}
	return true, nil
}

func (r *roleCommandRepository) RestoreAllRole(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllRoles(ctx)

	if err != nil {
		return false, role_errors.ErrRestoreAllRoles.WithInternal(err)
	}

	return true, nil
}

func (r *roleCommandRepository) DeleteAllRolePermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentRoles(ctx)

	if err != nil {
		return false, role_errors.ErrDeleteAllRoles.WithInternal(err)
	}

	return true, nil
}

func (r *roleCommandRepository) CreateUserRole(ctx context.Context, userID, roleID int) (*db.Role, error) {
	_, err := r.db.AssignRoleToUser(ctx, db.AssignRoleToUserParams{
		UserID: int32(userID),
		RoleID: int32(roleID),
	})

	if err != nil {
		return nil, role_errors.ErrCreateRole.WithInternal(err) // Should have a specific association error
	}

	// This method returns *db.UserRole, but we need *db.Role or similar.
	// Since the interface says *db.Role, I might need to fetch the role or just return the role info.
	// Actually, AssignRoleToUser returns *db.UserRole which doesn't contain RoleName.
	// I'll fetch the role info to satisfy the contract.
	role, err := r.db.GetRole(ctx, int32(roleID))
	if err != nil {
		return nil, role_errors.ErrUpdateRole.WithInternal(err)
	}

	return role, nil
}

func (r *roleCommandRepository) DeleteUserRole(ctx context.Context, userID, roleID int) (bool, error) {
	err := r.db.RemoveRoleFromUser(ctx, db.RemoveRoleFromUserParams{
		UserID: int32(userID),
		RoleID: int32(roleID),
	})

	if err != nil {
		return false, role_errors.ErrUpdateRole.WithInternal(err)
	}

	return true, nil
}
