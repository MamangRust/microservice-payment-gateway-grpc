package repository

import (
	"context"
	"time"

	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/role"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	role_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/role_errors/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// roleRepository is a struct that implements the RoleRepository interface
type roleRepository struct {
	roleQueryClient pb.RoleQueryServiceClient
}

// NewRoleRepository creates a new RoleRepository instance
func NewRoleRepository(queryClient pb.RoleQueryServiceClient) *roleRepository {
	return &roleRepository{
		roleQueryClient: queryClient,
	}
}

// FindById retrieves a role by its unique ID.
func (r *roleRepository) FindById(ctx context.Context, id int) (*db.Role, error) {
	resp, err := r.roleQueryClient.FindByIdRole(ctx, &pb.FindByIdRoleRequest{
		RoleId: int32(id),
	})

	if err != nil {
		return nil, role_errors.ErrRoleNotFound.WithInternal(err)
	}

	role := resp.GetData()
	parseTime := func(ts string) pgtype.Timestamp {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return pgtype.Timestamp{Valid: false}
		}
		return pgtype.Timestamp{Time: t, Valid: true}
	}

	return &db.Role{
		RoleID:    int32(role.Id),
		RoleName:  role.Name,
		CreatedAt: parseTime(role.CreatedAt),
		UpdatedAt: parseTime(role.UpdatedAt),
	}, nil
}

// FindByName retrieves a role by its name from the database via GRPC.
func (r *roleRepository) FindByName(ctx context.Context, name string) (*db.Role, error) {
	resp, err := r.roleQueryClient.FindAllRole(ctx, &pb.FindAllRoleRequest{
		Search:   name,
		Page:     1,
		PageSize: 1,
	})

	if err != nil {
		return nil, role_errors.ErrRoleNotFound.WithInternal(err)
	}

	roles := resp.GetData()
	if len(roles) == 0 {
		return nil, role_errors.ErrRoleNotFound
	}

	role := roles[0]
	parseTime := func(ts string) pgtype.Timestamp {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return pgtype.Timestamp{Valid: false}
		}
		return pgtype.Timestamp{Time: t, Valid: true}
	}

	return &db.Role{
		RoleID:    int32(role.Id),
		RoleName:  role.Name,
		CreatedAt: parseTime(role.CreatedAt),
		UpdatedAt: parseTime(role.UpdatedAt),
	}, nil
}
