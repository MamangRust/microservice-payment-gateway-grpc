package stats_reader_test

import (
	"context"
	"testing"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/handler"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-reader/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pb/transaction"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pb/topup"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pb/transfer"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pb/withdraw"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pb/saldo"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pb/merchant"
)

// --- mock repos ---

type mockTransactionRepo struct {
	monthlyAmounts func(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyAmount, error)
	yearlyAmounts  func(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyAmount, error)
	monthlyMethod  func(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyMethodStats, error)
	yearlyMethod   func(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyMethodStats, error)
	monthlyStatus  func(ctx context.Context, table, filterField string, filterValue interface{}, year int, targetStatus string) ([]repository.MonthlyStatusStats, error)
	yearlyStatus   func(ctx context.Context, table, filterField string, filterValue interface{}, currentYear int, targetStatus string) ([]repository.YearlyStatusStats, error)
}

func (m *mockTransactionRepo) GetMonthlyAmounts(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyAmount, error) {
	if m.monthlyAmounts != nil {
		return m.monthlyAmounts(ctx, table, filterField, filterValue, year)
	}
	return nil, nil
}
func (m *mockTransactionRepo) GetYearlyAmounts(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyAmount, error) {
	if m.yearlyAmounts != nil {
		return m.yearlyAmounts(ctx, table, filterField, filterValue, startYear, endYear)
	}
	return nil, nil
}
func (m *mockTransactionRepo) GetMonthlyMethodStats(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyMethodStats, error) {
	if m.monthlyMethod != nil {
		return m.monthlyMethod(ctx, table, filterField, filterValue, year)
	}
	return nil, nil
}
func (m *mockTransactionRepo) GetYearlyMethodStats(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyMethodStats, error) {
	if m.yearlyMethod != nil {
		return m.yearlyMethod(ctx, table, filterField, filterValue, startYear, endYear)
	}
	return nil, nil
}
func (m *mockTransactionRepo) GetMonthlyStatusStats(ctx context.Context, table, filterField string, filterValue interface{}, year int, targetStatus string) ([]repository.MonthlyStatusStats, error) {
	if m.monthlyStatus != nil {
		return m.monthlyStatus(ctx, table, filterField, filterValue, year, targetStatus)
	}
	return nil, nil
}
func (m *mockTransactionRepo) GetYearlyStatusStats(ctx context.Context, table, filterField string, filterValue interface{}, currentYear int, targetStatus string) ([]repository.YearlyStatusStats, error) {
	if m.yearlyStatus != nil {
		return m.yearlyStatus(ctx, table, filterField, filterValue, currentYear, targetStatus)
	}
	return nil, nil
}

var _ handler.TransactionRepository = (*mockTransactionRepo)(nil)

type mockTopupRepo struct {
	monthlyAmounts func(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyAmount, error)
	yearlyAmounts  func(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyAmount, error)
	monthlyMethod  func(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyMethodStats, error)
	yearlyMethod   func(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyMethodStats, error)
	monthlyStatus  func(ctx context.Context, table, filterField string, filterValue interface{}, year int, targetStatus string) ([]repository.MonthlyStatusStats, error)
	yearlyStatus   func(ctx context.Context, table, filterField string, filterValue interface{}, currentYear int, targetStatus string) ([]repository.YearlyStatusStats, error)
}

func (m *mockTopupRepo) GetMonthlyAmounts(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyAmount, error) {
	if m.monthlyAmounts != nil {
		return m.monthlyAmounts(ctx, table, filterField, filterValue, year)
	}
	return nil, nil
}
func (m *mockTopupRepo) GetYearlyAmounts(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyAmount, error) {
	if m.yearlyAmounts != nil {
		return m.yearlyAmounts(ctx, table, filterField, filterValue, startYear, endYear)
	}
	return nil, nil
}
func (m *mockTopupRepo) GetMonthlyMethodStats(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyMethodStats, error) {
	if m.monthlyMethod != nil {
		return m.monthlyMethod(ctx, table, filterField, filterValue, year)
	}
	return nil, nil
}
func (m *mockTopupRepo) GetYearlyMethodStats(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyMethodStats, error) {
	if m.yearlyMethod != nil {
		return m.yearlyMethod(ctx, table, filterField, filterValue, startYear, endYear)
	}
	return nil, nil
}
func (m *mockTopupRepo) GetMonthlyStatusStats(ctx context.Context, table, filterField string, filterValue interface{}, year int, targetStatus string) ([]repository.MonthlyStatusStats, error) {
	if m.monthlyStatus != nil {
		return m.monthlyStatus(ctx, table, filterField, filterValue, year, targetStatus)
	}
	return nil, nil
}
func (m *mockTopupRepo) GetYearlyStatusStats(ctx context.Context, table, filterField string, filterValue interface{}, currentYear int, targetStatus string) ([]repository.YearlyStatusStats, error) {
	if m.yearlyStatus != nil {
		return m.yearlyStatus(ctx, table, filterField, filterValue, currentYear, targetStatus)
	}
	return nil, nil
}

var _ handler.TopupRepository = (*mockTopupRepo)(nil)

type mockWithdrawRepo struct {
	monthlyAmounts func(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyAmount, error)
	yearlyAmounts  func(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyAmount, error)
	monthlyStatus  func(ctx context.Context, table, filterField string, filterValue interface{}, year int, targetStatus string) ([]repository.MonthlyStatusStats, error)
	yearlyStatus   func(ctx context.Context, table, filterField string, filterValue interface{}, currentYear int, targetStatus string) ([]repository.YearlyStatusStats, error)
}

func (m *mockWithdrawRepo) GetMonthlyAmounts(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyAmount, error) {
	if m.monthlyAmounts != nil {
		return m.monthlyAmounts(ctx, table, filterField, filterValue, year)
	}
	return nil, nil
}
func (m *mockWithdrawRepo) GetYearlyAmounts(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyAmount, error) {
	if m.yearlyAmounts != nil {
		return m.yearlyAmounts(ctx, table, filterField, filterValue, startYear, endYear)
	}
	return nil, nil
}
func (m *mockWithdrawRepo) GetMonthlyStatusStats(ctx context.Context, table, filterField string, filterValue interface{}, year int, targetStatus string) ([]repository.MonthlyStatusStats, error) {
	if m.monthlyStatus != nil {
		return m.monthlyStatus(ctx, table, filterField, filterValue, year, targetStatus)
	}
	return nil, nil
}
func (m *mockWithdrawRepo) GetYearlyStatusStats(ctx context.Context, table, filterField string, filterValue interface{}, currentYear int, targetStatus string) ([]repository.YearlyStatusStats, error) {
	if m.yearlyStatus != nil {
		return m.yearlyStatus(ctx, table, filterField, filterValue, currentYear, targetStatus)
	}
	return nil, nil
}

var _ handler.WithdrawRepository = (*mockWithdrawRepo)(nil)

type mockTransferRepo struct {
	monthlyAmounts func(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyAmount, error)
	yearlyAmounts  func(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyAmount, error)
	monthlyStatus  func(ctx context.Context, table, filterField string, filterValue interface{}, year int, targetStatus string) ([]repository.MonthlyStatusStats, error)
	yearlyStatus   func(ctx context.Context, table, filterField string, filterValue interface{}, currentYear int, targetStatus string) ([]repository.YearlyStatusStats, error)
}

func (m *mockTransferRepo) GetMonthlyAmounts(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyAmount, error) {
	if m.monthlyAmounts != nil {
		return m.monthlyAmounts(ctx, table, filterField, filterValue, year)
	}
	return nil, nil
}
func (m *mockTransferRepo) GetYearlyAmounts(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyAmount, error) {
	if m.yearlyAmounts != nil {
		return m.yearlyAmounts(ctx, table, filterField, filterValue, startYear, endYear)
	}
	return nil, nil
}
func (m *mockTransferRepo) GetMonthlyStatusStats(ctx context.Context, table, filterField string, filterValue interface{}, year int, targetStatus string) ([]repository.MonthlyStatusStats, error) {
	if m.monthlyStatus != nil {
		return m.monthlyStatus(ctx, table, filterField, filterValue, year, targetStatus)
	}
	return nil, nil
}
func (m *mockTransferRepo) GetYearlyStatusStats(ctx context.Context, table, filterField string, filterValue interface{}, currentYear int, targetStatus string) ([]repository.YearlyStatusStats, error) {
	if m.yearlyStatus != nil {
		return m.yearlyStatus(ctx, table, filterField, filterValue, currentYear, targetStatus)
	}
	return nil, nil
}

var _ handler.TransferRepository = (*mockTransferRepo)(nil)

type mockSaldoRepo struct {
	monthlyAmounts    func(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyAmount, error)
	yearlyAmounts     func(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyAmount, error)
	monthlyTotalSaldo func(ctx context.Context, year int) ([]repository.MonthlyAmount, error)
	yearlyTotalSaldo  func(ctx context.Context, startYear, endYear int) ([]repository.YearlyAmount, error)
}

func (m *mockSaldoRepo) GetMonthlyAmounts(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyAmount, error) {
	if m.monthlyAmounts != nil {
		return m.monthlyAmounts(ctx, table, filterField, filterValue, year)
	}
	return nil, nil
}
func (m *mockSaldoRepo) GetYearlyAmounts(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyAmount, error) {
	if m.yearlyAmounts != nil {
		return m.yearlyAmounts(ctx, table, filterField, filterValue, startYear, endYear)
	}
	return nil, nil
}
func (m *mockSaldoRepo) GetMonthlyTotalSaldo(ctx context.Context, year int) ([]repository.MonthlyAmount, error) {
	if m.monthlyTotalSaldo != nil {
		return m.monthlyTotalSaldo(ctx, year)
	}
	return nil, nil
}
func (m *mockSaldoRepo) GetYearlyTotalSaldo(ctx context.Context, startYear, endYear int) ([]repository.YearlyAmount, error) {
	if m.yearlyTotalSaldo != nil {
		return m.yearlyTotalSaldo(ctx, startYear, endYear)
	}
	return nil, nil
}

var _ handler.SaldoRepository = (*mockSaldoRepo)(nil)

type mockMerchantRepo struct {
	monthlyAmounts                func(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyAmount, error)
	yearlyAmounts                 func(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyAmount, error)
	monthlyMethod                 func(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyMethodStats, error)
	findMerchantTransactions      func(ctx context.Context, merchantID int32) ([]map[string]interface{}, error)
	findMerchantTransactionsByKey func(ctx context.Context, apiKey string) ([]map[string]interface{}, error)
}

func (m *mockMerchantRepo) GetMonthlyAmounts(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyAmount, error) {
	if m.monthlyAmounts != nil {
		return m.monthlyAmounts(ctx, table, filterField, filterValue, year)
	}
	return nil, nil
}
func (m *mockMerchantRepo) GetYearlyAmounts(ctx context.Context, table, filterField string, filterValue interface{}, startYear, endYear int) ([]repository.YearlyAmount, error) {
	if m.yearlyAmounts != nil {
		return m.yearlyAmounts(ctx, table, filterField, filterValue, startYear, endYear)
	}
	return nil, nil
}
func (m *mockMerchantRepo) GetMonthlyMethodStats(ctx context.Context, table, filterField string, filterValue interface{}, year int) ([]repository.MonthlyMethodStats, error) {
	if m.monthlyMethod != nil {
		return m.monthlyMethod(ctx, table, filterField, filterValue, year)
	}
	return nil, nil
}
func (m *mockMerchantRepo) FindMerchantTransactions(ctx context.Context, merchantID int32) ([]map[string]interface{}, error) {
	if m.findMerchantTransactions != nil {
		return m.findMerchantTransactions(ctx, merchantID)
	}
	return nil, nil
}
func (m *mockMerchantRepo) FindMerchantTransactionsByApikey(ctx context.Context, apiKey string) ([]map[string]interface{}, error) {
	if m.findMerchantTransactionsByKey != nil {
		return m.findMerchantTransactionsByKey(ctx, apiKey)
	}
	return nil, nil
}

var _ handler.MerchantRepository = (*mockMerchantRepo)(nil)

func newTestLogger() logger.LoggerInterface {
	z, _ := zap.NewDevelopment()
	return &logger.Logger{Log: z}
}

// --- Transaction Stats Tests ---

func TestTransactionStatsHandler_FindMonthlyAmounts(t *testing.T) {
	repo := &mockTransactionRepo{
		monthlyAmounts: func(_ context.Context, table, _ string, _ interface{}, _ int) ([]repository.MonthlyAmount, error) {
			assert.Equal(t, "transaction_events", table)
			return []repository.MonthlyAmount{
				{Year: "2026", Month: "Mar", TotalAmount: 500000},
			}, nil
		},
	}
	h := handler.NewTransactionStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyAmounts(context.Background(), &transaction.FindYearTransactionStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, int64(500000), resp.Data[0].TotalAmount)
}

func TestTransactionStatsHandler_FindMonthlyAmounts_Error(t *testing.T) {
	repo := &mockTransactionRepo{
		monthlyAmounts: func(_ context.Context, _, _ string, _ interface{}, _ int) ([]repository.MonthlyAmount, error) {
			return nil, assert.AnError
		},
	}
	h := handler.NewTransactionStatsHandler(repo, newTestLogger())

	_, err := h.FindMonthlyAmounts(context.Background(), &transaction.FindYearTransactionStatus{Year: 2026})
	require.Error(t, err)
}

func TestTransactionStatsHandler_FindYearlyAmounts(t *testing.T) {
	repo := &mockTransactionRepo{
		yearlyAmounts: func(_ context.Context, _, _ string, _ interface{}, _, _ int) ([]repository.YearlyAmount, error) {
			return []repository.YearlyAmount{
				{Year: "2026", TotalAmount: 5000000},
			}, nil
		},
	}
	h := handler.NewTransactionStatsHandler(repo, newTestLogger())

	resp, err := h.FindYearlyAmounts(context.Background(), &transaction.FindYearTransactionStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestTransactionStatsHandler_FindMonthlyAmountsByCardNumber(t *testing.T) {
	repo := &mockTransactionRepo{
		monthlyAmounts: func(_ context.Context, _, filterField string, filterValue interface{}, _ int) ([]repository.MonthlyAmount, error) {
			assert.Equal(t, "card_number", filterField)
			assert.Equal(t, "1234567890", filterValue)
			return []repository.MonthlyAmount{
				{Year: "2026", Month: "Apr", TotalAmount: 100000},
			}, nil
		},
	}
	h := handler.NewTransactionStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyAmountsByCardNumber(context.Background(), &transaction.FindByYearCardNumberTransactionRequest{Year: 2026, CardNumber: "1234567890"})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestTransactionStatsHandler_FindMonthlyPaymentMethods(t *testing.T) {
	repo := &mockTransactionRepo{
		monthlyMethod: func(_ context.Context, table, _ string, _ interface{}, _ int) ([]repository.MonthlyMethodStats, error) {
			assert.Equal(t, "transaction_events", table)
			return []repository.MonthlyMethodStats{
				{Month: "May", PaymentMethod: "credit_card", TotalTransactions: 20, TotalAmount: 300000},
			}, nil
		},
	}
	h := handler.NewTransactionStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyPaymentMethods(context.Background(), &transaction.FindYearTransactionStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "credit_card", resp.Data[0].PaymentMethod)
	assert.Equal(t, int32(20), resp.Data[0].TotalTransactions)
}

func TestTransactionStatsHandler_FindMonthlyTransactionStatusSuccess(t *testing.T) {
	repo := &mockTransactionRepo{
		monthlyStatus: func(_ context.Context, table, _ string, _ interface{}, _ int, status string) ([]repository.MonthlyStatusStats, error) {
			assert.Equal(t, "transaction_events", table)
			assert.Equal(t, "success", status)
			return []repository.MonthlyStatusStats{
				{Year: "2026", Month: "Jun", Status: "success", TotalTransactions: 30, TotalAmount: 1500000},
			}, nil
		},
	}
	h := handler.NewTransactionStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyTransactionStatusSuccess(context.Background(), &transaction.FindMonthlyTransactionStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, int32(30), resp.Data[0].TotalSuccess)
}

func TestTransactionStatsHandler_FindMonthlyTransactionStatusFailed(t *testing.T) {
	repo := &mockTransactionRepo{
		monthlyStatus: func(_ context.Context, _, _ string, _ interface{}, _ int, status string) ([]repository.MonthlyStatusStats, error) {
			assert.Equal(t, "failed", status)
			return []repository.MonthlyStatusStats{
				{Year: "2026", Month: "Jul", Status: "failed", TotalTransactions: 5, TotalAmount: 25000},
			}, nil
		},
	}
	h := handler.NewTransactionStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyTransactionStatusFailed(context.Background(), &transaction.FindMonthlyTransactionStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, int32(5), resp.Data[0].TotalFailed)
}

func TestTransactionStatsHandler_FindYearlyTransactionStatusSuccess(t *testing.T) {
	repo := &mockTransactionRepo{
		yearlyStatus: func(_ context.Context, _, _ string, _ interface{}, _ int, status string) ([]repository.YearlyStatusStats, error) {
			assert.Equal(t, "success", status)
			return []repository.YearlyStatusStats{
				{Year: "2026", Status: "success", TotalTransactions: 300, TotalAmount: 15000000},
			}, nil
		},
	}
	h := handler.NewTransactionStatsHandler(repo, newTestLogger())

	resp, err := h.FindYearlyTransactionStatusSuccess(context.Background(), &transaction.FindYearTransactionStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, int32(300), resp.Data[0].TotalSuccess)
}

// --- Topup Stats Tests ---

func TestTopupStatsHandler_FindMonthlyTopupAmounts(t *testing.T) {
	repo := &mockTopupRepo{
		monthlyAmounts: func(_ context.Context, table, _ string, _ interface{}, _ int) ([]repository.MonthlyAmount, error) {
			assert.Equal(t, "topup_events", table)
			return []repository.MonthlyAmount{
				{Year: "2026", Month: "Jan", TotalAmount: 200000},
			}, nil
		},
	}
	h := handler.NewTopupStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyTopupAmounts(context.Background(), &topup.FindYearTopupStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, int64(200000), resp.Data[0].TotalAmount)
}

func TestTopupStatsHandler_FindMonthlyTopupAmounts_Error(t *testing.T) {
	repo := &mockTopupRepo{
		monthlyAmounts: func(_ context.Context, _, _ string, _ interface{}, _ int) ([]repository.MonthlyAmount, error) {
			return nil, assert.AnError
		},
	}
	h := handler.NewTopupStatsHandler(repo, newTestLogger())

	_, err := h.FindMonthlyTopupAmounts(context.Background(), &topup.FindYearTopupStatus{Year: 2026})
	require.Error(t, err)
}

func TestTopupStatsHandler_FindYearlyTopupAmounts(t *testing.T) {
	repo := &mockTopupRepo{
		yearlyAmounts: func(_ context.Context, _, _ string, _ interface{}, _, _ int) ([]repository.YearlyAmount, error) {
			return []repository.YearlyAmount{
				{Year: "2026", TotalAmount: 2000000},
			}, nil
		},
	}
	h := handler.NewTopupStatsHandler(repo, newTestLogger())

	resp, err := h.FindYearlyTopupAmounts(context.Background(), &topup.FindYearTopupStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestTopupStatsHandler_FindMonthlyTopupMethods(t *testing.T) {
	repo := &mockTopupRepo{
		monthlyMethod: func(_ context.Context, _, _ string, _ interface{}, _ int) ([]repository.MonthlyMethodStats, error) {
			return []repository.MonthlyMethodStats{
				{Month: "Feb", PaymentMethod: "bank_transfer", TotalTransactions: 15, TotalAmount: 750000},
			}, nil
		},
	}
	h := handler.NewTopupStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyTopupMethods(context.Background(), &topup.FindYearTopupStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "bank_transfer", resp.Data[0].TopupMethod)
}

func TestTopupStatsHandler_FindMonthlyTopupStatusSuccess(t *testing.T) {
	repo := &mockTopupRepo{
		monthlyStatus: func(_ context.Context, _, _ string, _ interface{}, _ int, status string) ([]repository.MonthlyStatusStats, error) {
			assert.Equal(t, "success", status)
			return []repository.MonthlyStatusStats{
				{Year: "2026", Month: "Mar", Status: "success", TotalTransactions: 25, TotalAmount: 1250000},
			}, nil
		},
	}
	h := handler.NewTopupStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyTopupStatusSuccess(context.Background(), &topup.FindMonthlyTopupStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, int32(25), resp.Data[0].TotalSuccess)
}

func TestTopupStatsHandler_FindMonthlyTopupStatusFailed(t *testing.T) {
	repo := &mockTopupRepo{
		monthlyStatus: func(_ context.Context, _, _ string, _ interface{}, _ int, status string) ([]repository.MonthlyStatusStats, error) {
			assert.Equal(t, "failed", status)
			return []repository.MonthlyStatusStats{
				{Year: "2026", Month: "Apr", Status: "failed", TotalTransactions: 3, TotalAmount: 15000},
			}, nil
		},
	}
	h := handler.NewTopupStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyTopupStatusFailed(context.Background(), &topup.FindMonthlyTopupStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

// --- Withdraw Stats Tests ---

func TestWithdrawStatsHandler_FindMonthlyWithdraws(t *testing.T) {
	repo := &mockWithdrawRepo{
		monthlyAmounts: func(_ context.Context, table, _ string, _ interface{}, _ int) ([]repository.MonthlyAmount, error) {
			assert.Equal(t, "withdraw_events", table)
			return []repository.MonthlyAmount{
				{Year: "2026", Month: "May", TotalAmount: 100000},
			}, nil
		},
	}
	h := handler.NewWithdrawStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyWithdraws(context.Background(), &withdraw.FindYearWithdrawStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestWithdrawStatsHandler_FindYearlyWithdraws(t *testing.T) {
	repo := &mockWithdrawRepo{
		yearlyAmounts: func(_ context.Context, _, _ string, _ interface{}, _, _ int) ([]repository.YearlyAmount, error) {
			return []repository.YearlyAmount{
				{Year: "2026", TotalAmount: 1000000},
			}, nil
		},
	}
	h := handler.NewWithdrawStatsHandler(repo, newTestLogger())

	resp, err := h.FindYearlyWithdraws(context.Background(), &withdraw.FindYearWithdrawStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestWithdrawStatsHandler_FindMonthlyWithdrawStatusSuccess(t *testing.T) {
	repo := &mockWithdrawRepo{
		monthlyStatus: func(_ context.Context, _, _ string, _ interface{}, _ int, status string) ([]repository.MonthlyStatusStats, error) {
			assert.Equal(t, "success", status)
			return []repository.MonthlyStatusStats{
				{Year: "2026", Month: "Jun", Status: "success", TotalTransactions: 10, TotalAmount: 500000},
			}, nil
		},
	}
	h := handler.NewWithdrawStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyWithdrawStatusSuccess(context.Background(), &withdraw.FindMonthlyWithdrawStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestWithdrawStatsHandler_FindMonthlyWithdrawStatusFailed(t *testing.T) {
	repo := &mockWithdrawRepo{
		monthlyStatus: func(_ context.Context, _, _ string, _ interface{}, _ int, status string) ([]repository.MonthlyStatusStats, error) {
			assert.Equal(t, "failed", status)
			return []repository.MonthlyStatusStats{
				{Year: "2026", Month: "Jul", Status: "failed", TotalTransactions: 2, TotalAmount: 10000},
			}, nil
		},
	}
	h := handler.NewWithdrawStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyWithdrawStatusFailed(context.Background(), &withdraw.FindMonthlyWithdrawStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

// --- Transfer Stats Tests ---

func TestTransferStatsHandler_FindMonthlyTransferAmounts(t *testing.T) {
	repo := &mockTransferRepo{
		monthlyAmounts: func(_ context.Context, table, _ string, _ interface{}, _ int) ([]repository.MonthlyAmount, error) {
			assert.Equal(t, "transfer_events", table)
			return []repository.MonthlyAmount{
				{Year: "2026", Month: "Aug", TotalAmount: 300000},
			}, nil
		},
	}
	h := handler.NewTransferStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyTransferAmounts(context.Background(), &transfer.FindYearTransferStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestTransferStatsHandler_FindYearlyTransferAmounts(t *testing.T) {
	repo := &mockTransferRepo{
		yearlyAmounts: func(_ context.Context, _, _ string, _ interface{}, _, _ int) ([]repository.YearlyAmount, error) {
			return []repository.YearlyAmount{
				{Year: "2026", TotalAmount: 3000000},
			}, nil
		},
	}
	h := handler.NewTransferStatsHandler(repo, newTestLogger())

	resp, err := h.FindYearlyTransferAmounts(context.Background(), &transfer.FindYearTransferStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestTransferStatsHandler_FindMonthlyTransferStatusSuccess(t *testing.T) {
	repo := &mockTransferRepo{
		monthlyStatus: func(_ context.Context, _, _ string, _ interface{}, _ int, status string) ([]repository.MonthlyStatusStats, error) {
			assert.Equal(t, "success", status)
			return []repository.MonthlyStatusStats{
				{Year: "2026", Month: "Sep", Status: "success", TotalTransactions: 40, TotalAmount: 2000000},
			}, nil
		},
	}
	h := handler.NewTransferStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyTransferStatusSuccess(context.Background(), &transfer.FindMonthlyTransferStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestTransferStatsHandler_FindMonthlyTransferStatusFailed(t *testing.T) {
	repo := &mockTransferRepo{
		monthlyStatus: func(_ context.Context, _, _ string, _ interface{}, _ int, status string) ([]repository.MonthlyStatusStats, error) {
			assert.Equal(t, "failed", status)
			return []repository.MonthlyStatusStats{
				{Year: "2026", Month: "Oct", Status: "failed", TotalTransactions: 4, TotalAmount: 20000},
			}, nil
		},
	}
	h := handler.NewTransferStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyTransferStatusFailed(context.Background(), &transfer.FindMonthlyTransferStatus{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

// --- Saldo Stats Tests ---

func TestSaldoStatsHandler_FindMonthlySaldoBalances(t *testing.T) {
	repo := &mockSaldoRepo{
		monthlyAmounts: func(_ context.Context, table, _ string, _ interface{}, _ int) ([]repository.MonthlyAmount, error) {
			assert.Equal(t, "saldo_events", table)
			return []repository.MonthlyAmount{
				{Year: "2026", Month: "Nov", TotalAmount: 5000000},
			}, nil
		},
	}
	h := handler.NewSaldoStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlySaldoBalances(context.Background(), &saldo.FindYearlySaldo{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestSaldoStatsHandler_FindYearlySaldoBalances(t *testing.T) {
	repo := &mockSaldoRepo{
		yearlyAmounts: func(_ context.Context, _, _ string, _ interface{}, _, _ int) ([]repository.YearlyAmount, error) {
			return []repository.YearlyAmount{
				{Year: "2026", TotalAmount: 50000000},
			}, nil
		},
	}
	h := handler.NewSaldoStatsHandler(repo, newTestLogger())

	resp, err := h.FindYearlySaldoBalances(context.Background(), &saldo.FindYearlySaldo{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestSaldoStatsHandler_FindMonthlyTotalSaldoBalance(t *testing.T) {
	repo := &mockSaldoRepo{
		monthlyTotalSaldo: func(_ context.Context, _ int) ([]repository.MonthlyAmount, error) {
			return []repository.MonthlyAmount{
				{Year: "2026", Month: "Dec", TotalAmount: 7000000},
			}, nil
		},
	}
	h := handler.NewSaldoStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyTotalSaldoBalance(context.Background(), &saldo.FindMonthlySaldoTotalBalance{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestSaldoStatsHandler_FindYearTotalSaldoBalance(t *testing.T) {
	repo := &mockSaldoRepo{
		yearlyTotalSaldo: func(_ context.Context, _, _ int) ([]repository.YearlyAmount, error) {
			return []repository.YearlyAmount{
				{Year: "2026", TotalAmount: 70000000},
			}, nil
		},
	}
	h := handler.NewSaldoStatsHandler(repo, newTestLogger())

	resp, err := h.FindYearTotalSaldoBalance(context.Background(), &saldo.FindYearlySaldo{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

// --- Merchant Stats Tests ---

func TestMerchantStatsHandler_FindAllTransactionByMerchant(t *testing.T) {
	repo := &mockMerchantRepo{
		findMerchantTransactions: func(_ context.Context, merchantID int32) ([]map[string]interface{}, error) {
			assert.Equal(t, int32(10), merchantID)
			return []map[string]interface{}{
				{"id": uint64(1), "amount": int64(50000), "method": "credit_card", "created_at": "2026-01-15"},
			}, nil
		},
	}
	h := handler.NewMerchantStatsHandler(repo, newTestLogger())

	resp, err := h.FindAllTransactionByMerchant(context.Background(), &merchant.FindAllMerchantTransaction{MerchantId: 10})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestMerchantStatsHandler_FindAllTransactionByApikey(t *testing.T) {
	repo := &mockMerchantRepo{
		findMerchantTransactionsByKey: func(_ context.Context, apiKey string) ([]map[string]interface{}, error) {
			assert.Equal(t, "test-api-key", apiKey)
			return []map[string]interface{}{}, nil
		},
	}
	h := handler.NewMerchantStatsHandler(repo, newTestLogger())

	resp, err := h.FindAllTransactionByApikey(context.Background(), &merchant.FindAllMerchantApikey{ApiKey: "test-api-key"})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
}

func TestMerchantStatsHandler_FindMonthlyAmountMerchant(t *testing.T) {
	repo := &mockMerchantRepo{
		monthlyAmounts: func(_ context.Context, _, _ string, _ interface{}, _ int) ([]repository.MonthlyAmount, error) {
			return []repository.MonthlyAmount{
				{Year: "2026", Month: "Jan", TotalAmount: 1000000},
			}, nil
		},
	}
	h := handler.NewMerchantStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyAmountMerchant(context.Background(), &merchant.FindYearMerchant{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}

func TestMerchantStatsHandler_FindMonthlyPaymentMethodsMerchant(t *testing.T) {
	repo := &mockMerchantRepo{
		monthlyMethod: func(_ context.Context, _, _ string, _ interface{}, _ int) ([]repository.MonthlyMethodStats, error) {
			return []repository.MonthlyMethodStats{
				{Month: "Mar", PaymentMethod: "bank_transfer", TotalTransactions: 100, TotalAmount: 5000000},
			}, nil
		},
	}
	h := handler.NewMerchantStatsHandler(repo, newTestLogger())

	resp, err := h.FindMonthlyPaymentMethodsMerchant(context.Background(), &merchant.FindYearMerchant{Year: 2026})
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	require.Len(t, resp.Data, 1)
}
