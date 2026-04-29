package cardrepositoryerrors

import (
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

var (
	// ErrGetMonthlyTopupAmountFailed is returned when fetching monthly top-up amounts fails.
	ErrGetMonthlyTopupAmountFailed = errors.ErrInternal.WithMessage("Failed to get monthly topup amount")

	// ErrGetYearlyTopupAmountFailed is returned when fetching yearly top-up amounts fails.
	ErrGetYearlyTopupAmountFailed = errors.ErrInternal.WithMessage("Failed to get yearly topup amount")

	// ErrGetMonthlyWithdrawAmountFailed is returned when fetching monthly withdrawal amounts fails.
	ErrGetMonthlyWithdrawAmountFailed = errors.ErrInternal.WithMessage("Failed to get monthly withdraw amount")

	// ErrGetYearlyWithdrawAmountFailed is returned when fetching yearly withdrawal amounts fails.
	ErrGetYearlyWithdrawAmountFailed = errors.ErrInternal.WithMessage("Failed to get yearly withdraw amount")

	// ErrGetMonthlyTransactionAmountFailed is returned when fetching monthly transaction amounts fails.
	ErrGetMonthlyTransactionAmountFailed = errors.ErrInternal.WithMessage("Failed to get monthly transaction amount")

	// ErrGetYearlyTransactionAmountFailed is returned when fetching yearly transaction amounts fails.
	ErrGetYearlyTransactionAmountFailed = errors.ErrInternal.WithMessage("Failed to get yearly transaction amount")

	// ErrGetMonthlyTransferAmountSenderFailed is returned when fetching monthly transfer amounts by sender fails.
	ErrGetMonthlyTransferAmountSenderFailed = errors.ErrInternal.WithMessage("Failed to get monthly transfer amount by sender")

	// ErrGetYearlyTransferAmountSenderFailed is returned when fetching yearly transfer amounts by sender fails.
	ErrGetYearlyTransferAmountSenderFailed = errors.ErrInternal.WithMessage("Failed to get yearly transfer amount by sender")

	// ErrGetMonthlyTransferAmountReceiverFailed is returned when fetching monthly transfer amounts by receiver fails.
	ErrGetMonthlyTransferAmountReceiverFailed = errors.ErrInternal.WithMessage("Failed to get monthly transfer amount by receiver")

	// ErrGetYearlyTransferAmountReceiverFailed is returned when fetching yearly transfer amounts by receiver fails.
	ErrGetYearlyTransferAmountReceiverFailed = errors.ErrInternal.WithMessage("Failed to get yearly transfer amount by receiver")
)
