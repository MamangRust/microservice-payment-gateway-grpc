package repository

import (
	"context"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/models"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

// UserRepository resolves and mutates users for the auth service through the
// user adapter.
type UserRepository interface {
	CreateUser(ctx context.Context, req *requests.RegisterRequest) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUserIsVerified(ctx context.Context, userID int, isVerified bool) (*models.User, error)
}

type userRepository struct {
	adapter adapter.AuthUserAdapter
}

// NewUserRepository wraps an AuthUserAdapter so the repository layer can manage
// users without depending on gRPC directly.
func NewUserRepository(userAdapter adapter.AuthUserAdapter) UserRepository {
	return &userRepository{adapter: userAdapter}
}

func (r *userRepository) CreateUser(ctx context.Context, req *requests.RegisterRequest) (*models.User, error) {
	return r.adapter.CreateUser(ctx, req)
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.adapter.FindByEmail(ctx, email)
}

func (r *userRepository) UpdateUserIsVerified(ctx context.Context, userID int, isVerified bool) (*models.User, error) {
	if err := r.adapter.UpdateUserIsVerified(ctx, userID, isVerified); err != nil {
		return nil, err
	}
	return r.adapter.FindById(ctx, userID)
}
