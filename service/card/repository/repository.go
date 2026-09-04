package repository

import (
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/user"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
)

// Repositories contains all repositories used by the card service.
type Repositories struct {
	CardCommand         CardCommandRepository
	CardQuery           CardQueryRepository
	User                UserRepository
	CardAuthTransaction CardAuthTransactionRepository
	CardPayment         CardPaymentRepository
	CardReward          CardRewardRepository
	BillingCycle        BillingCycleRepository
}

func NewRepositories(database *db.Queries, userClient pb.UserQueryServiceClient) *Repositories {
	return &Repositories{
		CardQuery:           NewCardQueryRepository(database),
		CardCommand:         NewCardCommandRepository(database),
		User:                NewUserRepository(userClient),
		CardAuthTransaction: NewCardAuthTransactionRepository(database),
		CardPayment:         NewCardPaymentRepository(database),
		CardReward:          NewCardRewardRepository(database),
		BillingCycle:        NewBillingCycleRepository(database),
	}
}
