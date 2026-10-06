package repository

import (
	pbrole "github.com/MamangRust/microservice-payment-gateway-grpc/pb/role"
	pbuser "github.com/MamangRust/microservice-payment-gateway-grpc/pb/user"
	pbuserrole "github.com/MamangRust/microservice-payment-gateway-grpc/pb/user_role"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/auth/database/schema"
)

type Repositories struct {
	User         UserRepository
	RefreshToken RefreshTokenRepository
	UserRole     UserRoleRepository
	Role         RoleRepository
	ResetToken   ResetTokenRepository
}

type GuardOptions struct {
	User     []adapter.GuardOption
	UserRole []adapter.GuardOption
	Role     []adapter.GuardOption
}

func NewRepositories(
	db *db.Queries,
	userQueryClient pbuser.UserQueryServiceClient,
	userCommandClient pbuser.UserCommandServiceClient,
	roleQueryClient pbrole.RoleQueryServiceClient,
	roleCommandClient pbrole.RoleCommandServiceClient,
	userRoleClient pbuserrole.UserRoleServiceClient,
	guards ...GuardOptions,
) *Repositories {
	var g GuardOptions
	if len(guards) > 0 {
		g = guards[0]
	}

	return &Repositories{
		User:         NewUserRepository(adapter.NewAuthUserAdapter(userQueryClient, userCommandClient, g.User...)),
		UserRole:     adapter.NewUserRoleAdapter(userRoleClient, g.UserRole...),
		RefreshToken: NewRefreshTokenRepository(db),
		Role:         adapter.NewRoleAdapter(roleQueryClient, roleCommandClient, g.Role...),
		ResetToken:   NewResetTokenRepository(db),
	}
}
