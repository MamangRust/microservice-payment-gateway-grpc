package repository

import "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"

// CardRepository exposes the card gRPC adapter to the saldo repository layer.
type CardRepository = adapter.CardAdapter
