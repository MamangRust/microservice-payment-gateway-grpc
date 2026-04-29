package handler

import (
	pbmerchant "github.com/MamangRust/microservice-payment-gateway-grpc/pb/merchant"
	pbdocument "github.com/MamangRust/microservice-payment-gateway-grpc/pb/merchant_document"
)

type MerchantDocumentQueryHandleGrpc interface {
	pbdocument.MerchantDocumentQueryServiceServer
}

type MerchantDocumentCommandHandleGrpc interface {
	pbdocument.MerchantDocumentCommandServiceServer
}

type MerchantQueryHandleGrpc interface {
	pbmerchant.MerchantQueryServiceServer
}

type MerchantCommandHandleGrpc interface {
	pbmerchant.MerchantCommandServiceServer
}

type MerchantTransactionHandleGrpc interface {
	pbmerchant.MerchantTransactionServiceServer
}
