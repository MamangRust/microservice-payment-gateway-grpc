package service

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

//go:generate mockgen -source=interfaces.go -destination=mocks/service.go
type CardQueryService interface {
	FindAll(ctx context.Context, req *requests.FindAllCards) ([]*db.GetCardsRow, *int, error)
	FindByActive(ctx context.Context, req *requests.FindAllCards) ([]*db.GetActiveCardsWithCountRow, *int, error)
	FindByTrashed(ctx context.Context, req *requests.FindAllCards) ([]*db.GetTrashedCardsWithCountRow, *int, error)
	FindById(ctx context.Context, cardID int) (*db.GetCardByIDRow, error)
	FindByUserID(ctx context.Context, userID int) (*db.GetCardByUserIDRow, error)
	FindByCardNumber(ctx context.Context, cardNumber string) (*db.GetCardByCardNumberRow, error)
	FindUserCardByCardNumber(ctx context.Context, cardNumber string) (*db.GetUserEmailByCardNumberRow, error)
}

type CardCommandService interface {
	CreateCard(ctx context.Context, request *requests.CreateCardRequest) (*db.CreateCardRow, error)
	UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*db.UpdateCardRow, error)
	TrashedCard(ctx context.Context, cardID int) (*db.Card, error)
	RestoreCard(ctx context.Context, cardID int) (*db.Card, error)
	DeleteCardPermanent(ctx context.Context, cardID int) (bool, error)
	RestoreAllCard(ctx context.Context) (bool, error)
	DeleteAllCardPermanent(ctx context.Context) (bool, error)
	ToggleCardStatus(ctx context.Context, request *requests.ToggleCardStatusRequest) (*db.UpdateCardStatusRow, error)
	UpdateCreditLimit(ctx context.Context, request *requests.UpdateCreditLimitRequest) (*db.UpdateCreditLimitRow, error)
	RedeemPoints(ctx context.Context, request *requests.RedeemPointsRequest) (*db.RedeemRewardPointsRow, error)
	ProcessBillingCycles(ctx context.Context) error
}

// BillingEngineService owns statement-period calculation and processing. It is
// deliberately separate from CardCommandService so scheduled billing cannot be
// mistaken for a card CRUD command.
type BillingEngineService interface {
	TriggerBillingCycle(ctx context.Context, billingCycleDay int) (int, error)
	GetStatement(ctx context.Context, cardNumber string) (*db.BillingCycle, error)
	GetStatementsByCard(ctx context.Context, cardNumber string, page, pageSize int) ([]*db.BillingCycle, error)
	GetBillingCyclesByCardNumber(ctx context.Context, cardNumber string) ([]*db.BillingCycle, error)
}
