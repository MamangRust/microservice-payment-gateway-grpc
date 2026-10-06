package userrolegrpcerrors

import (
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

var (
	ErrFindUserRole       = errors.ErrInternal.WithMessage("Failed to find user role")
	ErrAssignRoleToUser   = errors.ErrInternal.WithMessage("Failed to assign role to user")
	ErrRemoveRoleFromUser = errors.ErrInternal.WithMessage("Failed to remove role from user")
)
