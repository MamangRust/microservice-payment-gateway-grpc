package cardgrpcerrors

import (
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	"google.golang.org/grpc/codes"
)

var (
	ErrGrpcValidateCreateCardRequest = errors.NewGrpcError("Invalid input for create card", int(codes.InvalidArgument))
	ErrGrpcValidateUpdateCardRequest = errors.NewGrpcError("Invalid input for update card", int(codes.InvalidArgument))
)
