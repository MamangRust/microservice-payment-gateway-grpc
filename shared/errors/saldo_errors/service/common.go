package saldoserviceerror

import (
	"net/http"

	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

// ErrFailedInsuffientBalance returns a 400 error when a transaction is attempted with insufficient balance.
var ErrFailedInsuffientBalance = errors.NewErrorResponse("Insufficient balance", http.StatusBadRequest)
