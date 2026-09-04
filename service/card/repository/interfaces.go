package repository

import (
	"context"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	userdb "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

//go:generate mockgen -source=interfaces.go -destination=mocks/card.go
type CardCommandRepository interface {
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
}

type CardQueryRepository interface {
	FindAllCards(ctx context.Context, req *requests.FindAllCards) ([]*db.GetCardsRow, error)
	FindByActive(ctx context.Context, req *requests.FindAllCards) ([]*db.GetActiveCardsWithCountRow, error)
	FindByTrashed(ctx context.Context, req *requests.FindAllCards) ([]*db.GetTrashedCardsWithCountRow, error)
	FindById(ctx context.Context, cardID int) (*db.GetCardByIDRow, error)
	FindCardByUserId(ctx context.Context, userID int) (*db.GetCardByUserIDRow, error)
	FindCardByCardNumber(ctx context.Context, cardNumber string) (*db.GetCardByCardNumberRow, error)
	FindUserCardByCardNumber(ctx context.Context, cardNumber string) (*db.GetUserEmailByCardNumberRow, error)
}

type UserRepository interface {
	FindById(ctx context.Context, userID int) (*userdb.GetUserByIDRow, error)
}

type CardAuthTransactionRepository interface {
	InsertPending(ctx context.Context, req *requests.AuthorizeCardRequest) (*db.CardAuthTransaction, error)
	Approve(ctx context.Context, txnID string) (*db.CardAuthTransaction, error)
	Decline(ctx context.Context, txnID string) (*db.CardAuthTransaction, error)
	Reverse(ctx context.Context, txnID string) (*db.CardAuthTransaction, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*db.CardAuthTransaction, error)
	FindByTxnID(ctx context.Context, txnID string) (*db.CardAuthTransaction, error)
	FindByCardNumber(ctx context.Context, cardNumber string, page, pageSize int) ([]*db.GetAuthTransactionsByCardNumberRow, error)
	CountRecentByCardNumber(ctx context.Context, cardNumber string, since time.Time) (int, error)
	UpdateRiskScore(ctx context.Context, txnID string, score int) error
}

type CardPaymentRepository interface {
	PostPayment(ctx context.Context, req *requests.PostPaymentRequest) (*db.CardPayment, error)
	GetPaymentHistory(ctx context.Context, cardNumber string, page, pageSize int) ([]*db.GetCardPaymentsByCardNumberRow, error)
	CountPayments(ctx context.Context, cardNumber string) (int, error)
}

type CardRewardRepository interface {
	EarnRewards(ctx context.Context, req *requests.EarnRewardsRequest) (*db.CardReward, error)
	GetBalance(ctx context.Context, cardNumber string) (int64, error)
	GetHistory(ctx context.Context, cardNumber string) ([]*db.CardReward, error)
	RedeemRewards(ctx context.Context, cardNumber string, points int64) (int64, error)
}

type BillingCycleRepository interface {
	GetBillingCyclesByCardNumber(ctx context.Context, cardNumber string) ([]*db.BillingCycle, error)
	// CreateBillingCycles creates one statement per active card for a period.
	// The database unique constraint makes retries and concurrent schedulers idempotent.
	CreateBillingCycles(ctx context.Context, cycleStart, cycleEnd, dueDate time.Time) (int, error)
}
