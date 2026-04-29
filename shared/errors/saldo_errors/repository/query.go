package saldorepositoryerrors

import (
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

var (
	// ErrFindAllSaldosFailed is returned when fetching all saldo records fails.
	ErrFindAllSaldosFailed = errors.ErrInternal.WithMessage("Failed to find all saldo records")

	// ErrFindActiveSaldosFailed is returned when fetching active saldo records fails.
	ErrFindActiveSaldosFailed = errors.ErrInternal.WithMessage("Failed to find active saldo records")

	// ErrFindTrashedSaldosFailed is returned when fetching trashed saldo records fails.
	ErrFindTrashedSaldosFailed = errors.ErrInternal.WithMessage("Failed to find trashed saldo records")

	// ErrFindSaldoByIdFailed is returned when fetching a saldo by its ID fails.
	ErrFindSaldoByIdFailed = errors.ErrInternal.WithMessage("Failed to find saldo by ID")

	// ErrFindSaldoByCardNumberFailed is returned when fetching saldo by card number fails.
	ErrFindSaldoByCardNumberFailed = errors.ErrInternal.WithMessage("Failed to find saldo by card number")
)
