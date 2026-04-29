package saldorepositoryerrors

import (
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

var (
	// ErrGetMonthlyTotalSaldoBalanceFailed is returned when fetching monthly total saldo balance fails.
	ErrGetMonthlyTotalSaldoBalanceFailed = errors.ErrInternal.WithMessage("Failed to get monthly total saldo balance")

	// ErrGetYearTotalSaldoBalanceFailed is returned when fetching yearly total saldo balance fails.
	ErrGetYearTotalSaldoBalanceFailed = errors.ErrInternal.WithMessage("Failed to get yearly total saldo balance")
)
