package cardserviceerrors

import (
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

var (
	// ErrFailedFindMonthlyTopupAmount indicates a failure in retrieving the monthly top-up amount.
	ErrFailedFindMonthlyTopupAmount = errors.ErrInternal.WithMessage("Failed to get monthly topup amount")

	// ErrFailedFindYearlyTopupAmount indicates a failure in retrieving the yearly top-up amount.
	ErrFailedFindYearlyTopupAmount = errors.ErrInternal.WithMessage("Failed to get yearly topup amount")

	// ErrFailedFindMonthlyWithdrawAmount indicates a failure in retrieving the monthly withdraw amount.
	ErrFailedFindMonthlyWithdrawAmount = errors.ErrInternal.WithMessage("Failed to get monthly withdraw amount")

	// ErrFailedFindYearlyWithdrawAmount indicates a failure in retrieving the yearly withdraw amount.
	ErrFailedFindYearlyWithdrawAmount = errors.ErrInternal.WithMessage("Failed to get yearly withdraw amount")

	// ErrFailedFindMonthlyTransactionAmount indicates a failure in retrieving the monthly transaction amount.
	ErrFailedFindMonthlyTransactionAmount = errors.ErrInternal.WithMessage("Failed to get monthly transaction amount")

	// ErrFailedFindYearlyTransactionAmount indicates a failure in retrieving the yearly transaction amount.
	ErrFailedFindYearlyTransactionAmount = errors.ErrInternal.WithMessage("Failed to get yearly transaction amount")

	// ErrFailedFindMonthlyTransferAmountSender indicates a failure in retrieving the monthly transfer amount by sender.
	ErrFailedFindMonthlyTransferAmountSender = errors.ErrInternal.WithMessage("Failed to get monthly transfer amount by sender")

	// ErrFailedFindYearlyTransferAmountSender indicates a failure in retrieving the yearly transfer amount by sender.
	ErrFailedFindYearlyTransferAmountSender = errors.ErrInternal.WithMessage("Failed to get yearly transfer amount by sender")

	// ErrFailedFindMonthlyTransferAmountReceiver indicates a failure in retrieving the monthly transfer amount by receiver.
	ErrFailedFindMonthlyTransferAmountReceiver = errors.ErrInternal.WithMessage("Failed to get monthly transfer amount by receiver")

	// ErrFailedFindYearlyTransferAmountReceiver indicates a failure in retrieving the yearly transfer amount by receiver.
	ErrFailedFindYearlyTransferAmountReceiver = errors.ErrInternal.WithMessage("Failed to get yearly transfer amount by receiver")
)

//
