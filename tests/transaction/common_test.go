package transaction_test

import (
	"context"
	carddb "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"

	pbAISecurity "github.com/MamangRust/microservice-payment-gateway-grpc/pb/ai_security"
	card_repo "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

type mockAISecurityServer struct {
	pbAISecurity.UnimplementedAISecurityServiceServer
}

func (m *mockAISecurityServer) DetectFraud(ctx context.Context, in *pbAISecurity.FraudRequest) (*pbAISecurity.FraudResponse, error) {
	return &pbAISecurity.FraudResponse{
		TransactionId: in.TransactionId,
		RiskScore:     0.1,
		IsFraudulent:  false,
		Reason:        "Mocked safe transaction",
	}, nil
}

// Wrapper to satisfy transaction repository requirements
type transactionCardRepo struct {
	query   card_repo.CardQueryRepository
	command card_repo.CardCommandRepository
}

func (r *transactionCardRepo) FindCardByUserId(ctx context.Context, user_id int) (*carddb.GetCardByUserIDRow, error) {
	return r.query.FindCardByUserId(ctx, user_id)
}
func (r *transactionCardRepo) FindUserCardByCardNumber(ctx context.Context, card_number string) (*carddb.GetUserEmailByCardNumberRow, error) {
	return r.query.FindUserCardByCardNumber(ctx, card_number)
}
func (r *transactionCardRepo) FindCardByCardNumber(ctx context.Context, card_number string) (*carddb.GetCardByCardNumberRow, error) {
	return r.query.FindCardByCardNumber(ctx, card_number)
}
func (r *transactionCardRepo) UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*carddb.UpdateCardRow, error) {
	return r.command.UpdateCard(ctx, request)
}
