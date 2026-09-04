package repository

import (
	"context"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"

	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/role"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/auth/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

// userRoleRepository is a struct that implements the UserRoleRepository interface
type userRoleRepository struct {
	roleCommandClient pb.RoleCommandServiceClient
}

// NewUserRoleRepository creates a new UserRoleRepository instance
func NewUserRoleRepository(commandClient pb.RoleCommandServiceClient) *userRoleRepository {
	return &userRoleRepository{
		roleCommandClient: commandClient,
	}
}

// AssignRoleToUser assigns a role to a user via GRPC.
func (r *userRoleRepository) AssignRoleToUser(ctx context.Context, req *requests.CreateUserRoleRequest) (*db.UserRole, error) {
	_, err := r.roleCommandClient.CreateUserRole(ctx, &pb.CreateUserRoleRequest{
		UserId: int32(req.UserId),
		RoleId: int32(req.RoleId),
	})

	if err != nil {
		return nil, sharedErrors.ErrFailed("assign role to user").WithInternal(err)
	}

	// Mapping back to db.UserRole (Note: UserRoleID might be lost or we just return a dummy if not critical)
	return &db.UserRole{
		UserID: int32(req.UserId),
		RoleID: int32(req.RoleId),
	}, nil
}

// RemoveRoleFromUser removes a role assigned to a user via GRPC.
func (r *userRoleRepository) RemoveRoleFromUser(ctx context.Context, req *requests.RemoveUserRoleRequest) error {
	_, err := r.roleCommandClient.DeleteUserRole(ctx, &pb.DeleteUserRoleRequest{
		UserId: int32(req.UserId),
		RoleId: int32(req.RoleId),
	})

	if err != nil {
		return sharedErrors.ErrFailed("remove role from user").WithInternal(err)
	}

	return nil
}
