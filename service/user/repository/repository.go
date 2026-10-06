package repository

import (
	pbrole "github.com/MamangRust/microservice-payment-gateway-grpc/pb/role"
	pbuserrole "github.com/MamangRust/microservice-payment-gateway-grpc/pb/user_role"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/database/schema"
)

type GuardOptions struct {
	UserRole []adapter.GuardOption
	Role     []adapter.GuardOption
}

type Repositories struct {
	UserCommand UserCommandRepository
	UserQuery   UserQueryRepository
	Role        RoleRepository
	UserRole    UserRoleRepository
}

type Deps struct {
	Db                *db.Queries
	RoleQueryClient   pbrole.RoleQueryServiceClient
	RoleCommandClient pbrole.RoleCommandServiceClient
	UserRoleClient    pbuserrole.UserRoleServiceClient
	Guard             GuardOptions
}

func NewRepositories(deps *Deps) Repositories {
	return Repositories{
		UserCommand: NewUserCommandRepository(deps.Db),
		UserQuery:   NewUserQueryRepository(deps.Db),
		Role:        NewRoleRepository(adapter.NewRoleAdapter(deps.RoleQueryClient, deps.RoleCommandClient, deps.Guard.Role...)),
		UserRole:    NewUserRoleRepository(adapter.NewUserRoleAdapter(deps.UserRoleClient, deps.Guard.UserRole...)),
	}
}
