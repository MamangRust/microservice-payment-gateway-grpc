package repository

import (
	pbuser "github.com/MamangRust/microservice-payment-gateway-grpc/pb/user"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/database/schema"
)

type GuardOptions struct {
	User []adapter.GuardOption
}

type UserRepository = adapter.UserAdapter

type Repositories interface {
	MerchantQueryRepository
	MerchantCommandRepository
	MerchantDocumentQueryRepository
	MerchantDocumentCommandRepository
	MerchantTransactionRepository
	UserRepository
}

type repositories struct {
	MerchantQueryRepository
	MerchantCommandRepository
	MerchantDocumentQueryRepository
	MerchantDocumentCommandRepository
	MerchantTransactionRepository
	UserRepository
}

func NewRepositories(db *db.Queries, userQueryClient pbuser.UserQueryServiceClient, guards ...GuardOptions) Repositories {
	var g GuardOptions

	if len(guards) > 0 {
		g = guards[0]
	}

	return &repositories{
		NewMerchantQueryRepository(db),
		NewMerchantCommandRepository(db),
		NewMerchantDocumentQueryRepository(db),
		NewMerchantDocumentCommandRepository(db),
		NewMerchantTransactionRepository(db),
		adapter.NewUserAdapter(userQueryClient, g.User...),
	}
}
