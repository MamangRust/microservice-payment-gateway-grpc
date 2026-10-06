package adapter

import (
	"context"

	pbrole "github.com/MamangRust/microservice-payment-gateway-grpc/pb/role"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/models"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/resilience"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

// RoleAdapter is the contract consumers depend on for role reads.
type RoleAdapter interface {
	FindById(ctx context.Context, roleID int) (*models.Role, error)
	FindByName(ctx context.Context, name string) (*models.Role, error)
}

type roleGRPCAdapter struct {
	QueryClient   pbrole.RoleQueryServiceClient
	CommandClient pbrole.RoleCommandServiceClient
	guard         *resilience.DependencyGuard
}

func (a *roleGRPCAdapter) SetGuard(g *resilience.DependencyGuard) {
	a.guard = g
}

// NewRoleAdapter builds a role adapter over the generated role query/command
// clients. The command client is retained so the adapter can grow write-side
// role operations without a signature change for existing call sites.
func NewRoleAdapter(queryClient pbrole.RoleQueryServiceClient, commandClient pbrole.RoleCommandServiceClient, opts ...GuardOption) RoleAdapter {
	a := &roleGRPCAdapter{
		QueryClient:   queryClient,
		CommandClient: commandClient,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *roleGRPCAdapter) FindById(ctx context.Context, roleID int) (*models.Role, error) {
	var resp *pbrole.ApiResponseRole
	err := a.guard.Call(ctx, func(ctx context.Context) error {
		var callErr error
		resp, callErr = a.QueryClient.FindByIdRole(ctx, &pbrole.FindByIdRoleRequest{RoleId: int32(roleID)})
		return callErr
	})
	if err != nil || resp == nil || resp.Data == nil {
		return nil, sharedErrors.ErrRoleNotFound.WithInternal(err)
	}
	return &models.Role{RoleID: resp.Data.Id, RoleName: resp.Data.Name}, nil
}

func (a *roleGRPCAdapter) FindByName(ctx context.Context, name string) (*models.Role, error) {
	var resp *pbrole.ApiResponseRole
	err := a.guard.Call(ctx, func(ctx context.Context) error {
		var callErr error
		resp, callErr = a.QueryClient.FindByNameRole(ctx, &pbrole.FindByNameRoleRequest{Name: name})
		return callErr
	})
	if err != nil || resp == nil || resp.Data == nil {
		return nil, sharedErrors.ErrRoleNotFound.WithInternal(err)
	}
	return &models.Role{RoleID: resp.Data.Id, RoleName: resp.Data.Name}, nil
}
