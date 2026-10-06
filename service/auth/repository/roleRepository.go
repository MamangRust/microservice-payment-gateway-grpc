package repository

import (
	"context"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/models"
)

// RoleRepository resolves roles for the auth service through the role adapter.
type RoleRepository interface {
	FindById(ctx context.Context, roleID int) (*models.Role, error)
	FindByName(ctx context.Context, name string) (*models.Role, error)
}

type roleRepository struct {
	adapter adapter.RoleAdapter
}

// NewRoleRepository wraps a RoleAdapter so the repository layer can resolve
// roles without depending on gRPC directly.
func NewRoleRepository(roleAdapter adapter.RoleAdapter) RoleRepository {
	return &roleRepository{adapter: roleAdapter}
}

func (r *roleRepository) FindById(ctx context.Context, roleID int) (*models.Role, error) {
	return r.adapter.FindById(ctx, roleID)
}

func (r *roleRepository) FindByName(ctx context.Context, name string) (*models.Role, error) {
	return r.adapter.FindByName(ctx, name)
}
