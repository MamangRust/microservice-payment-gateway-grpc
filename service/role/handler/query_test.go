package handler

import (
	"context"
	"testing"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/role/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/role"
)

type roleQueryServiceStub struct {
	active  []*db.GetActiveRolesRow
	trashed []*db.GetTrashedRolesRow
}

func (s *roleQueryServiceStub) FindAll(context.Context, *requests.FindAllRoles) ([]*db.GetRolesRow, *int, error) {
	return nil, intPointer(0), nil
}

func (s *roleQueryServiceStub) FindByActiveRole(context.Context, *requests.FindAllRoles) ([]*db.GetActiveRolesRow, *int, error) {
	total := len(s.active)
	return s.active, &total, nil
}

func (s *roleQueryServiceStub) FindByTrashedRole(context.Context, *requests.FindAllRoles) ([]*db.GetTrashedRolesRow, *int, error) {
	total := len(s.trashed)
	return s.trashed, &total, nil
}

func (s *roleQueryServiceStub) FindById(context.Context, int) (*db.Role, error) {
	return nil, nil
}

func (s *roleQueryServiceStub) FindByUserId(context.Context, int) ([]*db.Role, error) {
	return nil, nil
}

func (s *roleQueryServiceStub) FindByName(context.Context, string) (*db.Role, error) {
	return nil, nil
}

func intPointer(value int) *int {
	return &value
}

func validTimestamp(value time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: value, Valid: true}
}

func TestRoleQueryHandleGrpcFindByTrashed_NullSafeDeletedAt(t *testing.T) {
	createdAt := validTimestamp(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	updatedAt := validTimestamp(time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC))
	deletedAt := validTimestamp(time.Date(2026, time.January, 3, 0, 0, 0, 0, time.UTC))

	stub := &roleQueryServiceStub{
		trashed: []*db.GetTrashedRolesRow{
			{RoleID: 7, RoleName: "ROLE_TRASHED", CreatedAt: createdAt, UpdatedAt: updatedAt, DeletedAt: deletedAt},
			// A malformed/null row must not panic the handler or create a bogus date.
			{RoleID: 8, RoleName: "ROLE_NULL_DATE", CreatedAt: createdAt, UpdatedAt: updatedAt},
			nil,
		},
	}

	h := NewRoleQueryHandleGrpc(stub)
	res, err := h.FindByTrashed(context.Background(), &pb.FindAllRoleRequest{Page: 1, PageSize: 10})

	require.NoError(t, err)
	require.Len(t, res.Data, 2)
	require.Equal(t, int32(7), res.Data[0].Id)
	require.Equal(t, "2026-01-03", res.Data[0].GetDeletedAt().GetValue())
	require.Equal(t, int32(8), res.Data[1].Id)
	require.Nil(t, res.Data[1].GetDeletedAt())
}

func TestRoleQueryHandleGrpcFindByActive_NullSafeDeletedAt(t *testing.T) {
	createdAt := validTimestamp(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	updatedAt := validTimestamp(time.Date(2026, time.February, 2, 0, 0, 0, 0, time.UTC))

	stub := &roleQueryServiceStub{
		active: []*db.GetActiveRolesRow{
			{RoleID: 9, RoleName: "ROLE_ACTIVE", CreatedAt: createdAt, UpdatedAt: updatedAt},
			nil,
		},
	}

	h := NewRoleQueryHandleGrpc(stub)
	res, err := h.FindByActive(context.Background(), &pb.FindAllRoleRequest{Page: 1, PageSize: 10})

	require.NoError(t, err)
	require.Len(t, res.Data, 1)
	require.Equal(t, int32(9), res.Data[0].Id)
	require.Nil(t, res.Data[0].GetDeletedAt())
}
