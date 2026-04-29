package cardrepositoryerrors

import (
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

var (
	// ErrCreateCardFailed is returned when creating a new card fails.
	ErrCreateCardFailed = errors.ErrInternal.WithMessage("Failed to create card")

	// ErrUpdateCardFailed is returned when updating a card fails.
	ErrUpdateCardFailed = errors.ErrInternal.WithMessage("Failed to update card")

	// ErrTrashCardFailed is returned when trashing a card fails.
	ErrTrashCardFailed = errors.ErrInternal.WithMessage("Failed to move card to trash")

	// ErrRestoreCardFailed is returned when restoring a trashed card fails.
	ErrRestoreCardFailed = errors.ErrInternal.WithMessage("Failed to restore card from trash")

	// ErrDeleteCardPermanentFailed is returned when permanently deleting a card fails.
	ErrDeleteCardPermanentFailed = errors.ErrInternal.WithMessage("Failed to permanently delete card")

	// ErrRestoreAllCardsFailed is returned when restoring all trashed cards fails.
	ErrRestoreAllCardsFailed = errors.ErrInternal.WithMessage("Failed to restore all cards")

	// ErrDeleteAllCardsPermanentFailed is returned when permanently deleting all cards fails.
	ErrDeleteAllCardsPermanentFailed = errors.ErrInternal.WithMessage("Failed to permanently delete all cards")
)
