package cardserviceerrors

import (
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

var (
	// ErrFailedCreateCard is returned when creating a new Card record fails.
	ErrFailedCreateCard = errors.ErrInternal.WithMessage("Failed to create card")

	// ErrFailedUpdateCard is returned when updating an existing Card record fails.
	ErrFailedUpdateCard = errors.ErrInternal.WithMessage("Failed to update card")

	// ErrFailedTrashCard is returned when moving a Card to trash fails.
	ErrFailedTrashCard = errors.ErrInternal.WithMessage("Failed to move card to trash")

	// ErrFailedRestoreCard is returned when restoring a trashed Card fails.
	ErrFailedRestoreCard = errors.ErrInternal.WithMessage("Failed to restore card")

	// ErrFailedDeleteCard is returned when permanently deleting a Card fails.
	ErrFailedDeleteCard = errors.ErrInternal.WithMessage("Failed to delete card permanently")

	// ErrFailedRestoreAllCards is returned when restoring all trashed Cards fails.
	ErrFailedRestoreAllCards = errors.ErrInternal.WithMessage("Failed to restore all cards")

	// ErrFailedDeleteAllCards is returned when permanently deleting all Cards fails.
	ErrFailedDeleteAllCards = errors.ErrInternal.WithMessage("Failed to delete all cards permanently")
)
