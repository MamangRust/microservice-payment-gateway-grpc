package repository

import "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"

// RoleRepository resolves roles for the user service through the role adapter.
type RoleRepository = adapter.RoleAdapter

// NewRoleRepository wraps a RoleAdapter so the repository layer can resolve
// roles without depending on gRPC directly.
func NewRoleRepository(roleAdapter adapter.RoleAdapter) RoleRepository {
	return roleAdapter
}

// UserRoleRepository manages role assignment for the user service through the
// user-role adapter.
type UserRoleRepository = adapter.UserRoleAdapter

// NewUserRoleRepository wraps a UserRoleAdapter so the repository layer can
// manage a user's roles without depending on gRPC directly.
func NewUserRoleRepository(userRoleAdapter adapter.UserRoleAdapter) UserRoleRepository {
	return userRoleAdapter
}
