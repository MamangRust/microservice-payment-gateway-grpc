package service

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	mencache "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/redis"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharederrorhandler "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errorhandler"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// cardQueryServiceDeps defines dependencies for cardQueryService.
type cardQueryServiceDeps struct {
	Cache               mencache.CardQueryCache
	CardQueryRepository repository.CardQueryRepository
	UserRepository      repository.UserRepository
	Logger              logger.LoggerInterface
	Observability       observability.TraceLoggerObservability
}

// cardQueryService implements CardQueryService.
type cardQueryService struct {
	cache               mencache.CardQueryCache
	cardQueryRepository repository.CardQueryRepository
	userRepository      repository.UserRepository
	logger              logger.LoggerInterface
	observability       observability.TraceLoggerObservability
}

// NewCardQueryService initializes a new instance of cardQueryService with the provided parameters.
//
// It sets up Prometheus metrics for counting and measuring the duration of card query requests.
//
// Parameters:
// - params: A pointer to cardQueryServiceDeps containing the necessary dependencies.
//
// Returns:
// - A pointer to a newly created cardQueryService.
func NewCardQueryService(
	params *cardQueryServiceDeps,
) CardQueryService {
	return &cardQueryService{
		cardQueryRepository: params.CardQueryRepository,
		userRepository:      params.UserRepository,
		logger:              params.Logger,
		observability:       params.Observability,
		cache:               params.Cache,
	}
}

func (s *cardQueryService) FindAll(ctx context.Context, req *requests.FindAllCards) ([]*db.GetCardsRow, *int, error) {
	const method = "FindAll"

	page := req.Page
	pageSize := req.PageSize

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("page", page),
		attribute.Int("pageSize", pageSize),
		attribute.String("search", req.Search))

	defer func() {
		end(status, "grpc")
	}()

	if data, total, found := s.cache.GetFindAllCache(ctx, req); found {
		logSuccess("Successfully fetched card records from cache", zap.Int("totalRecords", *total), zap.Int("page", page), zap.Int("pageSize", pageSize))
		return data, total, nil
	}

	cards, err := s.cardQueryRepository.FindAllCards(ctx, req)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandlerErrorPagination[[]*db.GetCardsRow](
			s.logger,
			sharedErrors.ErrFailed("find all cards"),
			method,
			span,
			zap.Int("page", req.Page),
			zap.Int("pageSize", req.PageSize),
			zap.String("search", req.Search),
		)
	}

	var totalCount int

	if len(cards) > 0 {
		totalCount = int(cards[0].TotalCount)
	} else {
		totalCount = 0
	}

	s.cache.SetFindAllCache(ctx, req, cards, &totalCount)

	logSuccess("Successfully fetched card records",
		zap.Int("totalRecords", totalCount),

		zap.Int("page", page),
		zap.Int("pageSize", pageSize))

	return cards, &totalCount, nil
}

func (s *cardQueryService) FindByActive(ctx context.Context, req *requests.FindAllCards) ([]*db.GetActiveCardsWithCountRow, *int, error) {
	const method = "FindByActive"

	page := req.Page
	pageSize := req.PageSize
	search := req.Search

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("page", page),
		attribute.Int("pageSize", pageSize),
		attribute.String("search", search))

	defer func() {
		end(status, "grpc")
	}()

	if data, total, found := s.cache.GetByActiveCache(ctx, req); found {
		logSuccess("Successfully fetched active card records from cache", zap.Int("totalRecords", *total), zap.Int("page", page), zap.Int("pageSize", pageSize))
		return data, total, nil
	}

	res, err := s.cardQueryRepository.FindByActive(ctx, req)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandlerErrorPagination[[]*db.GetActiveCardsWithCountRow](
			s.logger,
			sharedErrors.ErrFailed("find active cards"),
			method,
			span,

			zap.Int("page", page),
			zap.Int("pageSize", pageSize),
			zap.String("search", search),
		)
	}

	var totalCount int

	if len(res) > 0 {
		totalCount = int(res[0].TotalCount)
	} else {
		totalCount = 0
	}

	s.cache.SetByActiveCache(ctx, req, res, &totalCount)

	logSuccess("Successfully fetched active card records",
		zap.Int("totalRecords", totalCount),
		zap.Int("page", page),
		zap.Int("pageSize", pageSize))

	return res, &totalCount, nil
}

func (s *cardQueryService) FindByTrashed(ctx context.Context, req *requests.FindAllCards) ([]*db.GetTrashedCardsWithCountRow, *int, error) {
	const method = "FindByTrashed"

	page := req.Page
	pageSize := req.PageSize
	search := req.Search

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("page", page),
		attribute.Int("pageSize", pageSize),
		attribute.String("search", search))

	defer func() {
		end(status, "grpc")
	}()

	if data, total, found := s.cache.GetByTrashedCache(ctx, req); found {
		logSuccess("Successfully fetched trashed card records from cache", zap.Int("totalRecords", *total), zap.Int("page", page), zap.Int("pageSize", pageSize))
		return data, total, nil
	}

	res, err := s.cardQueryRepository.FindByTrashed(ctx, req)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandlerErrorPagination[[]*db.GetTrashedCardsWithCountRow](
			s.logger,
			sharedErrors.ErrFailed("find trashed cards"),
			method,
			span,

			zap.Int("page", page),
			zap.Int("pageSize", pageSize),
			zap.String("search", search),
		)
	}

	var totalCount int

	if len(res) > 0 {
		totalCount = int(res[0].TotalCount)
	} else {
		totalCount = 0
	}

	s.cache.SetByTrashedCache(ctx, req, res, &totalCount)

	logSuccess("Successfully fetched trashed card records",
		zap.Int("totalRecords", totalCount),
		zap.Int("page", page),
		zap.Int("pageSize", pageSize))

	return res, &totalCount, nil
}

func (s *cardQueryService) FindById(ctx context.Context, card_id int) (*db.GetCardByIDRow, error) {
	const method = "FindById"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("card_id", card_id))

	defer func() {
		end(status, "grpc")
	}()

	if data, found := s.cache.GetByIdCache(ctx, card_id); found {
		logSuccess("Successfully fetched card from cache", zap.Int("card.id", card_id))
		return data, nil
	}

	res, err := s.cardQueryRepository.FindById(ctx, card_id)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandleError[*db.GetCardByIDRow](
			s.logger,
			err,
			method,
			span,

			zap.Int("card_id", card_id),
		)
	}

	s.cache.SetByIdCache(ctx, card_id, res)

	logSuccess("Successfully fetched card", zap.Int("card_id", card_id))

	return res, nil
}

func (s *cardQueryService) FindByCardNumber(ctx context.Context, card_number string) (*db.GetCardByCardNumberRow, error) {
	const method = "FindByCardNumber"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.String("card_number", card_number))

	defer func() {
		end(status, "grpc")
	}()

	if data, found := s.cache.GetByCardNumberCache(ctx, card_number); found {
		logSuccess("Successfully fetched card record by card number from cache", zap.String("card_number", card_number))
		return data, nil
	}

	res, err := s.cardQueryRepository.FindCardByCardNumber(ctx, card_number)
	if err != nil {
		status = "error"
		// Preserve the repository's domain/dependency error. Mapping every
		// failure to not-found hid user/database outages as a misleading card
		// lookup error and made financial commands return generic 500s.
		return sharederrorhandler.HandleError[*db.GetCardByCardNumberRow](
			s.logger,
			err,
			method,
			span,
			zap.String("card_number", card_number),
		)
	}

	s.cache.SetByCardNumberCache(ctx, card_number, res)

	logSuccess("Successfully fetched card record by card number", zap.String("card_number", card_number))

	return res, nil
}

func (s *cardQueryService) FindByUserID(ctx context.Context, user_id int) (*db.GetCardByUserIDRow, error) {
	const method = "FindByUserID"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("user_id", user_id))

	defer func() {
		end(status, "grpc")
	}()

	if data, found := s.cache.GetByUserIDCache(ctx, user_id); found {
		logSuccess("Successfully fetched card records by user ID from cache", zap.Int("user.id", user_id))
		return data, nil
	}

	res, err := s.cardQueryRepository.FindCardByUserId(ctx, user_id)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandleError[*db.GetCardByUserIDRow](
			s.logger,
			err,
			method,
			span,

			zap.Int("user_id", user_id),
		)
	}

	s.cache.SetByUserIDCache(ctx, user_id, res)

	logSuccess("Successfully fetched card records by user ID", zap.Int("user_id", user_id))

	return res, nil
}

func (s *cardQueryService) FindUserCardByCardNumber(ctx context.Context, card_number string) (*db.GetUserEmailByCardNumberRow, error) {
	const method = "FindUserCardByCardNumber"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.String("card_number", card_number))

	defer func() {
		end(status, "grpc")
	}()

	if data, found := s.cache.GetUserCardByCardNumberCache(ctx, card_number); found {
		if data.Email != "" {
			logSuccess("Successfully fetched card records by user ID from cache", zap.String("card_number", card_number))
			return data, nil
		}
		// A pre-fix cache entry may contain card data without the user email.
		// Re-enrich it instead of returning an unusable notification recipient.
		if err := s.enrichUserEmail(ctx, data); err != nil {
			status = "error"
			return nil, err
		}
		s.cache.SetUserCardByCardNumberCache(ctx, card_number, data)
		logSuccess("Successfully enriched cached card record", zap.String("card_number", card_number))
		return data, nil
	}

	res, err := s.cardQueryRepository.FindUserCardByCardNumber(ctx, card_number)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandleError[*db.GetUserEmailByCardNumberRow](
			s.logger,
			sharedErrors.ErrFailed("find card by user ID"),
			method,
			span,
			zap.String("card_number", card_number),
		)
	}

	// User profiles live in the user service, not card_db. Enrich the card
	// lookup through the existing gRPC repository instead of a cross-database SQL JOIN.
	if err := s.enrichUserEmail(ctx, res); err != nil {
		status = "error"
		return nil, err
	}

	s.cache.SetUserCardByCardNumberCache(ctx, card_number, res)

	logSuccess("Successfully fetched card records by user ID", zap.String("card_number", card_number))

	return res, nil
}

// normalizePagination normalizes pagination page and pageSize arguments.
//
// If page or pageSize is less than or equal to 0, it is set to the default value of 1 or 10, respectively.
//
// Parameters:
//   - page: The input page number.
//   - pageSize: The input page size.
//
// Returns:
//   - The normalized page number.
//   - The normalized page size.
func (s *cardQueryService) enrichUserEmail(ctx context.Context, card *db.GetUserEmailByCardNumberRow) error {
	if s.userRepository == nil {
		return sharedErrors.ErrInternal.WithMessage("user repository is not configured")
	}

	user, err := s.userRepository.FindById(ctx, int(card.UserID))
	if err != nil {
		return err
	}
	if user == nil || user.Email == "" {
		return sharedErrors.ErrNotFoundResponse("user").WithMessage("user email not found")
	}

	card.Email = user.Email
	return nil
}

func (s *cardQueryService) normalizePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return page, pageSize
}
