package repository

import (
	"context"

	database "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/role/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
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
		if database.IsUniqueViolation(err) {
			return nil, sharedErrors.NewConflictError("role name already exists").WithInternal(err)
		}
		return nil, sharedErrors.ErrFailed("create role").WithInternal(err)
	}

	return res, nil
}

func (r *roleCommandRepository) UpdateRole(ctx context.Context, req *requests.UpdateRoleRequest) (*db.Role, error) {
	res, err := r.db.UpdateRole(ctx, db.UpdateRoleParams{
		RoleID:   int32(*req.ID),
		RoleName: req.Name,
	})

	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, sharedErrors.NewConflictError("role name already exists").WithInternal(err)
		}
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "role", "update role")
	}

	return res, nil
}

func (r *roleCommandRepository) TrashedRole(ctx context.Context, id int) (*db.Role, error) {
	res, err := r.db.TrashRole(ctx, int32(id))
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "role", "trash role")
	}
	return res, nil
}

func (r *roleCommandRepository) RestoreRole(ctx context.Context, id int) (*db.Role, error) {
	res, err := r.db.RestoreRole(ctx, int32(id))
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "role", "restore role")
	}
	return res, nil
}

func (r *roleCommandRepository) DeleteRolePermanent(ctx context.Context, role_id int) (bool, error) {
	err := r.db.DeletePermanentRole(ctx, int32(role_id))
	if err != nil {
		return false, sharedErrors.ErrNoRowsOrFailed(err, "role", "delete role permanently")
	}
	return true, nil
}

func (r *roleCommandRepository) RestoreAllRole(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllRoles(ctx)

	if err != nil {
		return false, sharedErrors.ErrFailed("restore all roles").WithInternal(err)
	}

	return true, nil
}

func (r *roleCommandRepository) DeleteAllRolePermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentRoles(ctx)

	if err != nil {
		return false, sharedErrors.ErrFailed("delete all roles").WithInternal(err)
	}

	return true, nil
}

func (r *roleCommandRepository) CreateUserRole(ctx context.Context, userID, roleID int) (*db.Role, error) {
	_, err := r.db.AssignRoleToUser(ctx, db.AssignRoleToUserParams{
		UserID: int32(userID),
		RoleID: int32(roleID),
	})

	if err != nil {
		return nil, sharedErrors.ErrFailed("create user role").WithInternal(err)
	}

	// This method returns *db.UserRole, but we need *db.Role or similar.
	// Since the interface says *db.Role, I might need to fetch the role or just return the role info.
	// Actually, AssignRoleToUser returns *db.UserRole which doesn't contain RoleName.
	// I'll fetch the role info to satisfy the contract.
	role, err := r.db.GetRole(ctx, int32(roleID))
	if err != nil {
		return nil, sharedErrors.ErrFailed("get role for user role").WithInternal(err)
	}

	return role, nil
}

func (r *roleCommandRepository) DeleteUserRole(ctx context.Context, userID, roleID int) (bool, error) {
	err := r.db.RemoveRoleFromUser(ctx, db.RemoveRoleFromUserParams{
		UserID: int32(userID),
		RoleID: int32(roleID),
	})

	if err != nil {
		return false, sharedErrors.ErrNoRowsOrFailed(err, "user role", "delete user role")
	}

	return true, nil
}
