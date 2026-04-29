package repository

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

type RoleQueryRepository interface {
	FindAllRoles(ctx context.Context, req *requests.FindAllRoles) ([]*db.GetRolesRow, error)
	FindByActiveRole(ctx context.Context, req *requests.FindAllRoles) ([]*db.GetActiveRolesRow, error)
	FindByTrashedRole(ctx context.Context, req *requests.FindAllRoles) ([]*db.GetTrashedRolesRow, error)
	FindById(ctx context.Context, role_id int) (*db.Role, error)
	FindByName(ctx context.Context, name string) (*db.Role, error)
	FindByUserId(ctx context.Context, user_id int) ([]*db.Role, error)
}

type RoleCommandRepository interface {
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
