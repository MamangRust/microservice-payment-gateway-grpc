package repositorydashboard

import db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"

type CardDashboardRepository interface {
	CardDashboardBalanceRepository
	CardDashboardTopupRepository
	CardDashboardTransactionRepository
	CardDashboardTransferRepository
	CardDashboardWithdrawRepository
}

type repository struct {
	CardDashboardBalanceRepository
	CardDashboardTopupRepository
	CardDashboardTransactionRepository
	CardDashboardTransferRepository
	CardDashboardWithdrawRepository
}

func NewCardDashboardRepository(db *db.Queries) CardDashboardRepository {
	return &repository{
		CardDashboardBalanceRepository:     NewCardDashboardBalanceRepository(db),
		CardDashboardTopupRepository:       NewCardDashboardTopupRepository(db),
		CardDashboardTransactionRepository: NewCardDashboardTransactionRepository(db),
		CardDashboardTransferRepository:    NewCardDashboardTransferRepository(db),
		CardDashboardWithdrawRepository:    NewCardDashboardWithdrawRepository(db),
	}
}
