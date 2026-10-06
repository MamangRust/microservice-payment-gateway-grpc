// Package adapter (user_role) adapts role-assignment operations into the shared
// domain model behind a dedicated UserRoleAdapter.
//
// The user_role operations live on the standalone user_role gRPC service
// (pb/user_role): FindByUserId, AssignRoleToUser and RemoveRoleFromUser. This
// adapter is backed by that service's client so the auth/user repositories can
// resolve and mutate a user's roles through the adapter with a single
// dependency guard.
package adapter

import (
	"context"

	pbrole "github.com/MamangRust/microservice-payment-gateway-grpc/pb/role"
	pbuserrole "github.com/MamangRust/microservice-payment-gateway-grpc/pb/user_role"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/models"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/resilience"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	userrole_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/user_role_errors/grpc"
)

// UserRoleAdapter is the contract consumers depend on for role assignment.
type UserRoleAdapter interface {
	FindByUserId(ctx context.Context, userID int) ([]*models.Role, error)
	AssignRoleToUser(ctx context.Context, req *requests.CreateUserRoleRequest) (*models.UserRole, error)
	RemoveRoleFromUser(ctx context.Context, req *requests.RemoveUserRoleRequest) error
}

type userRoleGRPCAdapter struct {
	Client pbuserrole.UserRoleServiceClient
	guard  *resilience.DependencyGuard
}

func (a *userRoleGRPCAdapter) SetGuard(g *resilience.DependencyGuard) {
	a.guard = g
}

// NewUserRoleAdapter builds a user-role adapter over the user_role gRPC service
// client.
func NewUserRoleAdapter(userRoleClient pbuserrole.UserRoleServiceClient, opts ...GuardOption) UserRoleAdapter {
	a := &userRoleGRPCAdapter{
		Client: userRoleClient,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// FindByUserId resolves the roles assigned to a user.
func (a *userRoleGRPCAdapter) FindByUserId(ctx context.Context, userID int) ([]*models.Role, error) {
	var resp *pbrole.ApiResponsesRole
	err := a.guard.Call(ctx, func(ctx context.Context) error {
		var callErr error
		resp, callErr = a.Client.FindByUserId(ctx, &pbuserrole.FindByIdUserRoleRequest{UserId: int32(userID)})
		return callErr
	})
	if err != nil {
		return nil, userrole_errors.ErrFindUserRole.WithInternal(err)
	}
	if resp == nil {
		return nil, nil
	}

	roles := make([]*models.Role, 0, len(resp.Data))
	for _, item := range resp.Data {
		if item == nil {
			continue
		}
		roles = append(roles, &models.Role{RoleID: item.Id, RoleName: item.Name})
	}
	return roles, nil
}

// AssignRoleToUser implements UserRoleAdapter.
func (a *userRoleGRPCAdapter) AssignRoleToUser(ctx context.Context, req *requests.CreateUserRoleRequest) (*models.UserRole, error) {
	err := a.guard.Call(ctx, func(ctx context.Context) error {
		_, callErr := a.Client.AssignRoleToUser(ctx, &pbuserrole.AssignRoleToUserRequest{
			UserId: int32(req.UserId),
			RoleId: int32(req.RoleId),
		})
		return callErr
	})
	if err != nil {
		return nil, userrole_errors.ErrAssignRoleToUser.WithInternal(err)
	}
	return &models.UserRole{
		UserID: int32(req.UserId),
		RoleID: int32(req.RoleId),
	}, nil
}

// RemoveRoleFromUser implements UserRoleAdapter.
func (a *userRoleGRPCAdapter) RemoveRoleFromUser(ctx context.Context, req *requests.RemoveUserRoleRequest) error {
	err := a.guard.Call(ctx, func(ctx context.Context) error {
		_, callErr := a.Client.RemoveRoleFromUser(ctx, &pbuserrole.RemoveRoleFromUserRequest{
			UserId: int32(req.UserId),
			RoleId: int32(req.RoleId),
		})
		return callErr
	})
	if err != nil {
		return userrole_errors.ErrRemoveRoleFromUser.WithInternal(err)
	}
	return nil
}
