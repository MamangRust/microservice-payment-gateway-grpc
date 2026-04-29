package handler

import (
	"context"

	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/role"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/role/service"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	role_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/role_errors/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type roleCommandHandleGrpc struct {
	pb.UnimplementedRoleCommandServiceServer
	roleCommand service.RoleCommandService
}

func NewRoleCommandHandleGrpc(roleCommand service.RoleCommandService) RoleCommandHandlerGrpc {
	return &roleCommandHandleGrpc{
		roleCommand: roleCommand,
	}
}

func (s *roleCommandHandleGrpc) CreateRole(ctx context.Context, reqPb *pb.CreateRoleRequest) (*pb.ApiResponseRole, error) {
	req := &requests.CreateRoleRequest{
		Name: reqPb.Name,
	}

	if err := req.Validate(); err != nil {
		return nil, role_errors.ErrGrpcValidateCreateRole
	}

	role, err := s.roleCommand.CreateRole(ctx, req)

	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	protoRole := &pb.RoleResponse{
		Id:        int32(role.RoleID),
		Name:      role.RoleName,
		CreatedAt: role.CreatedAt.Time.Format("2006-01-02"),
		UpdatedAt: role.UpdatedAt.Time.Format("2006-01-02"),
	}

	return &pb.ApiResponseRole{
		Status:  "success",
		Message: "Successfully created role",
		Data:    protoRole,
	}, nil
}

func (s *roleCommandHandleGrpc) UpdateRole(ctx context.Context, reqPb *pb.UpdateRoleRequest) (*pb.ApiResponseRole, error) {
	roleID := int(reqPb.GetId())

	if roleID == 0 {
		return nil, role_errors.ErrGrpcRoleInvalidId
	}

	name := reqPb.GetName()

	req := &requests.UpdateRoleRequest{
		ID:   &roleID,
		Name: name,
	}

	if err := req.Validate(); err != nil {
		return nil, role_errors.ErrGrpcValidateUpdateRole
	}

	role, err := s.roleCommand.UpdateRole(ctx, req)

	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	protoRole := &pb.RoleResponse{
		Id:        int32(role.RoleID),
		Name:      role.RoleName,
		CreatedAt: role.CreatedAt.Time.Format("2006-01-02"),
		UpdatedAt: role.UpdatedAt.Time.Format("2006-01-02"),
	}

	return &pb.ApiResponseRole{
		Status:  "success",
		Message: "Successfully updated role",
		Data:    protoRole,
	}, nil
}

func (s *roleCommandHandleGrpc) TrashedRole(ctx context.Context, req *pb.FindByIdRoleRequest) (*pb.ApiResponseRoleDeleteAt, error) {
	roleID := int(req.GetRoleId())

	if roleID == 0 {
		return nil, role_errors.ErrGrpcRoleInvalidId
	}

	role, err := s.roleCommand.TrashedRole(ctx, roleID)

	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	protoRole := &pb.RoleResponseDeleteAt{
		Id:        int32(role.RoleID),
		Name:      role.RoleName,
		CreatedAt: role.CreatedAt.Time.Format("2006-01-02"),
		UpdatedAt: role.UpdatedAt.Time.Format("2006-01-02"),
		DeletedAt: wrapperspb.String(role.DeletedAt.Time.Format("2006-01-02")),
	}

	return &pb.ApiResponseRoleDeleteAt{
		Status:  "success",
		Message: "Successfully trashed role",
		Data:    protoRole,
	}, nil
}

func (s *roleCommandHandleGrpc) RestoreRole(ctx context.Context, req *pb.FindByIdRoleRequest) (*pb.ApiResponseRoleDeleteAt, error) {
	roleID := int(req.GetRoleId())

	if roleID == 0 {
		return nil, role_errors.ErrGrpcRoleInvalidId
	}

	role, err := s.roleCommand.RestoreRole(ctx, roleID)

	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	protoRole := &pb.RoleResponseDeleteAt{
		Id:        int32(role.RoleID),
		Name:      role.RoleName,
		CreatedAt: role.CreatedAt.Time.Format("2006-01-02"),
		UpdatedAt: role.UpdatedAt.Time.Format("2006-01-02"),
		DeletedAt: wrapperspb.String(role.DeletedAt.Time.Format("2006-01-02")),
	}

	return &pb.ApiResponseRoleDeleteAt{
		Status:  "success",
		Message: "Successfully restored role",
		Data:    protoRole,
	}, nil
}

func (s *roleCommandHandleGrpc) DeleteRolePermanent(ctx context.Context, req *pb.FindByIdRoleRequest) (*pb.ApiResponseRoleDelete, error) {
	roleID := int(req.GetRoleId())

	if roleID == 0 {
		return nil, role_errors.ErrGrpcRoleInvalidId
	}

	_, err := s.roleCommand.DeleteRolePermanent(ctx, roleID)

	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	return &pb.ApiResponseRoleDelete{
		Status:  "success",
		Message: "Successfully deleted role permanently",
	}, nil
}

func (s *roleCommandHandleGrpc) RestoreAllRole(ctx context.Context, _ *emptypb.Empty) (*pb.ApiResponseRoleAll, error) {
	_, err := s.roleCommand.RestoreAllRole(ctx)

	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	return &pb.ApiResponseRoleAll{
		Status:  "success",
		Message: "Successfully restore all roles",
	}, nil
}

func (s *roleCommandHandleGrpc) DeleteAllRolePermanent(ctx context.Context, _ *emptypb.Empty) (*pb.ApiResponseRoleAll, error) {
	_, err := s.roleCommand.DeleteAllRolePermanent(ctx)

	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	return &pb.ApiResponseRoleAll{
		Status:  "success",
		Message: "delete all roles permanent",
	}, nil
}
func (s *roleCommandHandleGrpc) CreateUserRole(ctx context.Context, request *pb.CreateUserRoleRequest) (*pb.ApiResponseRole, error) {
	userID := int(request.GetUserId())
	roleID := int(request.GetRoleId())

	if userID == 0 || roleID == 0 {
		return nil, role_errors.ErrGrpcRoleInvalidId
	}

	role, err := s.roleCommand.CreateUserRole(ctx, userID, roleID)

	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	protoRole := &pb.RoleResponse{
		Id:        int32(role.RoleID),
		Name:      role.RoleName,
		CreatedAt: role.CreatedAt.Time.Format("2006-01-02"),
		UpdatedAt: role.UpdatedAt.Time.Format("2006-01-02"),
	}

	return &pb.ApiResponseRole{
		Status:  "success",
		Message: "Successfully associated role with user",
		Data:    protoRole,
	}, nil
}

func (s *roleCommandHandleGrpc) DeleteUserRole(ctx context.Context, request *pb.DeleteUserRoleRequest) (*pb.ApiResponseRole, error) {
	userID := int(request.GetUserId())
	roleID := int(request.GetRoleId())

	if userID == 0 || roleID == 0 {
		return nil, role_errors.ErrGrpcRoleInvalidId
	}

	_, err := s.roleCommand.DeleteUserRole(ctx, userID, roleID)

	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	return &pb.ApiResponseRole{
		Status:  "success",
		Message: "Successfully removed role from user",
	}, nil
}
