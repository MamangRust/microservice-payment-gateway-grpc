package userserviceerrors

import (
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

var (
	// ErrUserIDInValid is returned when an invalid user ID is provided.
	ErrUserIDInValid = errors.ErrBadRequest.WithMessage("Invalid user ID")

	// ErrUserNotFoundRes is returned when a user is not found.
	ErrUserNotFoundRes = errors.ErrNotFound.WithMessage("User not found")

	// ErrUserEmailAlready is returned when a user email already exists.
	ErrUserEmailAlready = errors.ErrBadRequest.WithMessage("User email already exists")

	// ErrFailedFindAll is returned when fetching users fails.
	ErrFailedFindAll = errors.ErrInternal.WithMessage("Failed to fetch users")

	// ErrFailedFindActive is returned when fetching active users fails.
	ErrFailedFindActive = errors.ErrInternal.WithMessage("Failed to fetch active users")

	// ErrFailedFindTrashed is returned when fetching trashed users fails.
	ErrFailedFindTrashed = errors.ErrInternal.WithMessage("Failed to fetch trashed users")
)
