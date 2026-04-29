package service

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

// RoleQueryService is an interface for querying role records
type RoleQueryService interface {
	FindAll(ctx context.Context, req *requests.FindAllRoles) ([]*db.GetRolesRow, *int, error)
	FindByActiveRole(ctx context.Context, req *requests.FindAllRoles) ([]*db.GetActiveRolesRow, *int, error)
	FindByTrashedRole(ctx context.Context, req *requests.FindAllRoles) ([]*db.GetTrashedRolesRow, *int, error)
	FindById(ctx context.Context, role_id int) (*db.Role, error)
	FindByUserId(ctx context.Context, id int) ([]*db.Role, error)
	FindByName(ctx context.Context, name string) (*db.Role, error)
}

// RoleCommandService is an interface for creating, updating, and deleting role records
type RoleCommandService interface {
	CreateRole(ctx context.Context, request *requests.CreateRoleRequest) (*db.Role, error)
	UpdateRole(ctx context.Context, request *requests.UpdateRoleRequest) (*db.Role, error)
	CreateUserRole(ctx context.Context, userID, roleID int) (*db.Role, error)
	DeleteUserRole(ctx context.Context, userID, roleID int) (bool, error)
	TrashedRole(ctx context.Context, role_id int) (*db.Role, error)
	RestoreRole(ctx context.Context, role_id int) (*db.Role, error)
	DeleteRolePermanent(ctx context.Context, role_id int) (bool, error)

	RestoreAllRole(ctx context.Context) (bool, error)
	DeleteAllRolePermanent(ctx context.Context) (bool, error)
}
