package repository

import (
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/user"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
)

// Repositories contains all the repositories used in the application
type Repositories struct {
	CardCommand         CardCommandRepository
	CardQuery           CardQueryRepository
	User                UserRepository
}

func NewRepositories(db *db.Queries, userClient pb.UserQueryServiceClient) *Repositories {

	return &Repositories{
		CardQuery:     NewCardQueryRepository(db),
		CardCommand:   NewCardCommandRepository(db),
		User:          NewUserRepository(userClient),
	}
}
