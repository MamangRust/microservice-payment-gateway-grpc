package service

import (
	"context"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/models"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	mencache "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/redis"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errorhandler"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// merchantTransactionDeps holds dependencies for merchant transaction operations.
type merchantTransactionDeps struct {
	TransactionAdapter      adapter.TransactionAdapter
	MerchantQueryRepository repository.MerchantQueryRepository
	Cache                   mencache.MerchantTransactionCache
	Logger                  logger.LoggerInterface
	Observability           observability.TraceLoggerObservability
}

// merchantTransactionService handles merchant transaction operations.
type merchantTransactionService struct {
	transactionAdapter      adapter.TransactionAdapter
	merchantQueryRepository repository.MerchantQueryRepository
	cache                   mencache.MerchantTransactionCache
	logger                  logger.LoggerInterface
	observability           observability.TraceLoggerObservability
}

// NewMerchantTransactionService constructs a MerchantTransactionService.
func NewMerchantTransactionService(
	params *merchantTransactionDeps,
) MerchantTransactionService {
	return &merchantTransactionService{
		transactionAdapter:      params.TransactionAdapter,
		merchantQueryRepository: params.MerchantQueryRepository,
		cache:                   params.Cache,
		logger:                  params.Logger,
		observability:           params.Observability,
	}
}

func (s *merchantTransactionService) FindAllTransactions(ctx context.Context, req *requests.FindAllMerchantTransactions) ([]*models.Transaction, *int, error) {
	const method = "FindAllTransactions"
	page, pageSize := s.normalizePagination(req.Page, req.PageSize)
	search := req.Search

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method, attribute.Int("page", page), attribute.Int("pageSize", pageSize), attribute.String("search", search))
	defer func() { end(status, "grpc") }()

	if data, total, found := s.cache.GetCacheAllMerchantTransactions(ctx, req); found {
		logSuccess("Successfully retrieved all merchant transactions from cache", zap.Int("totalRecords", intValue(total)), zap.Int("page", page), zap.Int("pageSize", pageSize))
		return data, total, nil
	}

	transactions, totalCount, err := s.transactionAdapter.FindAllTransactions(ctx, page, pageSize, search)
	if err != nil {
		status = "error"
		return errorhandler.HandlerErrorPagination[[]*models.Transaction](s.logger, err, method, span, zap.String("search", search))
	}
	s.enrichMerchantNames(ctx, transactions)

	s.cache.SetCacheAllMerchantTransactions(ctx, req, transactions, totalCount)

	logSuccess("Successfully retrieved all merchant transactions", zap.Int("totalRecords", intValue(totalCount)), zap.Int("page", page), zap.Int("pageSize", pageSize))

	return transactions, totalCount, nil
}

func (s *merchantTransactionService) FindAllTransactionsByMerchant(ctx context.Context, req *requests.FindAllMerchantTransactionsById) ([]*models.Transaction, *int, error) {
	const method = "FindAllTransactionsByMerchant"
	page, pageSize := s.normalizePagination(req.Page, req.PageSize)
	search := req.Search

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method, attribute.Int("page", page), attribute.Int("pageSize", pageSize), attribute.String("search", search))
	defer func() { end(status, "grpc") }()

	if data, total, found := s.cache.GetCacheMerchantTransactions(ctx, req); found {
		logSuccess("Successfully retrieved merchant transactions from cache", zap.Int("totalRecords", intValue(total)), zap.Int("page", page), zap.Int("pageSize", pageSize))
		return data, total, nil
	}

	transactions, totalCount, err := s.transactionAdapter.FindAllTransactionsByMerchantId(ctx, req.MerchantID, page, pageSize, search)
	if err != nil {
		status = "error"
		return errorhandler.HandlerErrorPagination[[]*models.Transaction](s.logger, err, method, span, zap.String("search", search))
	}
	s.enrichMerchantNames(ctx, transactions)

	s.cache.SetCacheMerchantTransactions(ctx, req, transactions, totalCount)

	logSuccess("Successfully retrieved merchant transactions", zap.Int("totalRecords", intValue(totalCount)), zap.Int("page", page), zap.Int("pageSize", pageSize))

	return transactions, totalCount, nil
}

func (s *merchantTransactionService) FindAllTransactionsByApikey(ctx context.Context, req *requests.FindAllMerchantTransactionsByApiKey) ([]*models.Transaction, *int, error) {
	const method = "FindAllTransactionsByApikey"
	page, pageSize := s.normalizePagination(req.Page, req.PageSize)
	search := req.Search

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method, attribute.Int("page", page), attribute.Int("pageSize", pageSize), attribute.String("search", search))
	defer func() { end(status, "grpc") }()

	if data, total, found := s.cache.GetCacheMerchantTransactionApikey(ctx, req); found {
		logSuccess("Successfully retrieved merchant transactions by apikey from cache", zap.Int("totalRecords", intValue(total)), zap.Int("page", page), zap.Int("pageSize", pageSize))
		return data, total, nil
	}

	merchant, err := s.merchantQueryRepository.FindByApiKey(ctx, req.ApiKey)
	if err != nil {
		status = "error"
		return errorhandler.HandlerErrorPagination[[]*models.Transaction](s.logger, err, method, span, zap.String("api_key", req.ApiKey))
	}
	if merchant == nil {
		status = "error"
		return errorhandler.HandlerErrorPagination[[]*models.Transaction](s.logger, sharedErrors.ErrNotFoundResponse("merchant"), method, span, zap.String("api_key", req.ApiKey))
	}

	transactions, totalCount, err := s.transactionAdapter.FindAllTransactionsByMerchantId(ctx, int(merchant.MerchantID), page, pageSize, search)
	if err != nil {
		status = "error"
		return errorhandler.HandlerErrorPagination[[]*models.Transaction](s.logger, err, method, span, zap.String("search", search))
	}
	for _, tx := range transactions {
		if tx != nil {
			tx.MerchantName = merchant.Name
		}
	}

	s.cache.SetCacheMerchantTransactionApikey(ctx, req, transactions, totalCount)

	logSuccess("Successfully retrieved merchant transactions by apikey", zap.Int("totalRecords", intValue(totalCount)), zap.Int("page", page), zap.Int("pageSize", pageSize))

	return transactions, totalCount, nil
}

// enrichMerchantNames fills MerchantName from the merchant table this service
// owns. Lookups are deduplicated per distinct merchant ID in the page.
func (s *merchantTransactionService) enrichMerchantNames(ctx context.Context, transactions []*models.Transaction) {
	if s.merchantQueryRepository == nil {
		return
	}
	names := make(map[int]string)
	for _, tx := range transactions {
		if tx == nil || tx.MerchantName != "" {
			continue
		}
		name, ok := names[tx.MerchantID]
		if !ok {
			merchant, err := s.merchantQueryRepository.FindByMerchantId(ctx, tx.MerchantID)
			if err != nil || merchant == nil {
				name = ""
			} else {
				name = merchant.Name
			}
			names[tx.MerchantID] = name
		}
		tx.MerchantName = name
	}
}

func (s *merchantTransactionService) normalizePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return page, pageSize
}

func intValue(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
