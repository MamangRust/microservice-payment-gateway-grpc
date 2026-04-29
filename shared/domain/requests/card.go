package requests

import (
	"fmt"
	"time"

	methodtopup "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/method_topup"
	"github.com/go-playground/validator/v10"
)

// MonthYearCardNumberCard represents a request structure for operations requiring
// card number and year information, typically used for card lookups or expiration checks.
type MonthYearCardNumberCard struct {
	CardNumber string `json:"card_number" validate:"required"` // Full or partial card number (should be masked in logs)
	Year       int    `json:"year" validate:"required"`        // Year component of expiration date (YYYY format)
}

// FindAllCards represents the request parameters for searching and paginating card records.
// Used in card listing operations with search functionality.
type FindAllCards struct {
	Search   string `json:"search" validate:"required"`         // Search term to filter cards (matches against card number or provider)
	Page     int    `json:"page" validate:"min=1"`              // Page number for pagination (1-based index)
	PageSize int    `json:"page_size" validate:"min=1,max=100"` // Number of items per page (1-100)
}

// CreateCardRequest represents the payload for creating a new payment card.
// Contains all necessary information to register a payment card in the system.
type CreateCardRequest struct {
	UserID       int       `json:"user_id"`                           // ID of the user who owns the card
	CardType     string    `json:"card_type" validate:"required"`     // Type of card ("credit" or "debit")
	ExpireDate   time.Time `json:"expire_date" validate:"required"`   // Card expiration date
	CVV          string    `json:"cvv" validate:"required"`           // Card verification value (should be encrypted at rest)
	CardProvider string    `json:"card_provider" validate:"required"` // Issuing bank/provider (e.g., "VISA", "MasterCard")
}

// Validate performs validation of CreateCardRequest fields beyond basic struct validation.
// Ensures card type and provider are valid values.
// Returns:
//   - error: if validation fails, describing the specific validation error
func (r *CreateCardRequest) Validate() error {
	validate := validator.New()

	err := validate.Struct(r)

	if r.CardType != "credit" && r.CardType != "debit" {
		return fmt.Errorf("card type must be credit or debit")
	}

	if !methodtopup.PaymentMethodValidator(r.CardProvider) {
		return fmt.Errorf("card provider not found")
	}

	if err != nil {
		return err
	}

	return nil
}

// UpdateCardRequest represents the payload for updating an existing payment card.
// Contains all modifiable fields for a payment card record.
type UpdateCardRequest struct {
	CardID       int       `json:"card_id" validate:"required,min=1"` // ID of the card being updated
	UserID       int       `json:"user_id" validate:"required,min=1"` // ID of the card owner (for verification)
	CardType     string    `json:"card_type" validate:"required"`     // Updated card type ("credit" or "debit")
	ExpireDate   time.Time `json:"expire_date" validate:"required"`   // Updated expiration date
	CVV          string    `json:"cvv" validate:"required"`           // Updated card verification value
	CardProvider string    `json:"card_provider" validate:"required"` // Updated card provider
}

// Validate performs validation of UpdateCardRequest fields beyond basic struct validation.
// Ensures card type and provider are valid values and IDs are positive.
// Returns:
//   - error: if validation fails, describing the specific validation error
func (r *UpdateCardRequest) Validate() error {
	validate := validator.New()

	err := validate.Struct(r)

	if r.CardType != "credit" && r.CardType != "debit" {
		return fmt.Errorf("card type must be credit or debit")
	}

	if !methodtopup.PaymentMethodValidator(r.CardProvider) {
		return fmt.Errorf("card provider not found")
	}

	if err != nil {
		return err
	}

	return nil
}
