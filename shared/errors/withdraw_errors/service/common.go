package withdrawserviceerrors

import (
	"net/http"

	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

// ErrFailedSendEmail is used when failed to send email
var ErrFailedSendEmail = errors.NewErrorResponse("Failed to send email", http.StatusInternalServerError)
