package repository

import (
	"context"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/models"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

// UserRoleRepository manages role assignment for the auth service through the
// user-role adapter.
type UserRoleRepository interface {
	FindByUserId(ctx context.Context, userID int) ([]*models.Role, error)
	AssignRoleToUser(ctx context.Context, req *requests.CreateUserRoleRequest) (*models.UserRole, error)
	RemoveRoleFromUser(ctx context.Context, req *requests.RemoveUserRoleRequest) error
}

type userRoleRepository struct {
	adapter adapter.UserRoleAdapter
}

// NewUserRoleRepository wraps a UserRoleAdapter so the repository layer can manage
// role assignment without depending on gRPC directly.
func NewUserRoleRepository(userRoleAdapter adapter.UserRoleAdapter) UserRoleRepository {
	return &userRoleRepository{adapter: userRoleAdapter}
}

func (r *userRoleRepository) FindByUserId(ctx context.Context, userID int) ([]*models.Role, error) {
	return r.adapter.FindByUserId(ctx, userID)
}

func (r *userRoleRepository) AssignRoleToUser(ctx context.Context, req *requests.CreateUserRoleRequest) (*models.UserRole, error) {
	return r.adapter.AssignRoleToUser(ctx, req)
}

func (r *userRoleRepository) RemoveRoleFromUser(ctx context.Context, req *requests.RemoveUserRoleRequest) error {
	return r.adapter.RemoveRoleFromUser(ctx, req)
}
