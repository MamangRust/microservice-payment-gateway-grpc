package service

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	mencache "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/redis"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/events"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharederrorhandler "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errorhandler"
	card_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/card_errors/service"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"time"
)

// cardCommandServiceDeps defines dependencies for cardCommandService.
type cardCommandServiceDeps struct {
	Cache                 mencache.CardCommandCache
	Kafka                 *kafka.Kafka
	UserAdapter           adapter.UserAdapter
	CardCommandRepository repository.CardCommandRepository
	Logger                logger.LoggerInterface
	Observability         observability.TraceLoggerObservability
}

// cardCommandService implements CardCommandService.
type cardCommandService struct {
	cache                 mencache.CardCommandCache
	kafka                 *kafka.Kafka
	userAdapter           adapter.UserAdapter
	cardCommandRepository repository.CardCommandRepository
	logger                logger.LoggerInterface
	observability         observability.TraceLoggerObservability
}

func NewCardCommandService(params *cardCommandServiceDeps) CardCommandService {

	return &cardCommandService{
		cache:                 params.Cache,
		kafka:                 params.Kafka,
		userAdapter:           params.UserAdapter,
		cardCommandRepository: params.CardCommandRepository,
		logger:                params.Logger,
		observability:         params.Observability,
	}
}

func (s *cardCommandService) CreateCard(ctx context.Context, request *requests.CreateCardRequest) (*db.CreateCardRow, error) {
	const method = "CreateCard"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status)
	}()

	_, err := s.userAdapter.FindById(ctx, request.UserID)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandleError[*db.CreateCardRow](s.logger, err, method, span, zap.Int("user_id", request.UserID))
	}

	res, err := s.cardCommandRepository.CreateCard(ctx, request)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandleError[*db.CreateCardRow](s.logger, err, method, span, zap.Int("user_id", request.UserID))
	}

	if s.kafka != nil {
		go func() {
			saldoPayload := map[string]any{
				"card_number":   res.CardNumber,
				"total_balance": 0,
			}

			payloadBytes, err := json.Marshal(saldoPayload)
			if err != nil {
				s.logger.Error("failed to marshal saldo payload for new card", zap.Error(err), zap.Int("card_id", int(res.CardID)))
				return
			}

			err = s.kafka.SendMessage("saldo-service-topic-create-saldo", strconv.Itoa(int(res.CardID)), payloadBytes)
			if err != nil {
				s.logger.Error("failed to send create saldo message to kafka", zap.Error(err), zap.Int("card_id", int(res.CardID)))
			}

			// Stats Event
			statsEvent := events.CardEvent{
				CardID:       uint64(res.CardID),
				UserID:       uint64(request.UserID),
				CardNumber:   res.CardNumber,
				CardType:     res.CardType,
				CardProvider: res.CardProvider,
				Status:       "active", // Default status for new card
				CreatedAt:    time.Now(),
			}
			statsPayloadByte, _ := json.Marshal(statsEvent)
			_ = s.kafka.SendMessage("stats-topic-card-events", strconv.Itoa(int(res.CardID)), statsPayloadByte)
		}()
	}

	logSuccess("Successfully created card", zap.Int("card.id", int(res.CardID)), zap.String("card.card_number", res.CardNumber))

	return res, nil
}

func (s *cardCommandService) UpdateCard(ctx context.Context, request *requests.UpdateCardRequest) (*db.UpdateCardRow, error) {
	const method = "UpdateCard"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status)
	}()

	_, err := s.userAdapter.FindById(ctx, request.UserID)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandleError[*db.UpdateCardRow](s.logger, err, method, span, zap.Int("user_id", request.UserID))
	}

	res, err := s.cardCommandRepository.UpdateCard(ctx, request)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandleError[*db.UpdateCardRow](s.logger, err, method, span, zap.Int("card_id", request.CardID))
	}

	s.cache.DeleteCardCommandCache(ctx, request.CardID)

	logSuccess("Successfully updated card", zap.Int("card.id", int(res.CardID)))

	return res, nil
}

func (s *cardCommandService) TrashedCard(ctx context.Context, card_id int) (*db.Card, error) {
	const method = "TrashedCard"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("card_id", card_id))

	defer func() {
		end(status)
	}()

	res, err := s.cardCommandRepository.TrashedCard(ctx, card_id)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandleError[*db.Card](
			s.logger,
			card_errors.ErrFailedTrashCard,
			method,
			span,

			zap.Int("card_id", card_id),
		)
	}

	s.cache.DeleteCardCommandCache(ctx, card_id)

	logSuccess("Successfully trashed card", zap.Int("card_id", int(res.CardID)))

	return res, nil
}

func (s *cardCommandService) RestoreCard(ctx context.Context, card_id int) (*db.Card, error) {
	const method = "RestoreCard"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("card_id", card_id))

	defer func() {
		end(status)
	}()

	s.logger.Debug("Restoring card", zap.Int("card_id", card_id))

	res, err := s.cardCommandRepository.RestoreCard(ctx, card_id)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandleError[*db.Card](
			s.logger,
			card_errors.ErrFailedRestoreCard,
			method,
			span,

			zap.Int("card_id", card_id),
		)
	}

	s.cache.DeleteCardCommandCache(ctx, card_id)

	logSuccess("Successfully restored card", zap.Int("card_id", int(res.CardID)))

	return res, nil
}

func (s *cardCommandService) DeleteCardPermanent(ctx context.Context, card_id int) (bool, error) {
	const method = "DeleteCardPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("card_id", card_id))

	defer func() {
		end(status)
	}()

	_, err := s.cardCommandRepository.DeleteCardPermanent(ctx, card_id)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandleError[bool](
			s.logger,
			card_errors.ErrFailedDeleteCard,
			method,
			span,

			zap.Int("card_id", card_id),
		)
	}

	s.cache.DeleteCardCommandCache(ctx, card_id)

	logSuccess("Successfully deleted card permanently", zap.Int("card_id", card_id))

	return true, nil
}

func (s *cardCommandService) RestoreAllCard(ctx context.Context) (bool, error) {
	const method = "RestoreAllCard"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status)
	}()

	_, err := s.cardCommandRepository.RestoreAllCard(ctx)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandleError[bool](
			s.logger,
			card_errors.ErrFailedRestoreAllCards,
			method,
			span,
		)
	}

	logSuccess("Successfully restored all cards")

	return true, nil
}

func (s *cardCommandService) DeleteAllCardPermanent(ctx context.Context) (bool, error) {
	const method = "DeleteAllCardPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status)
	}()

	_, err := s.cardCommandRepository.DeleteAllCardPermanent(ctx)
	if err != nil {
		status = "error"
		return sharederrorhandler.HandleError[bool](
			s.logger,
			card_errors.ErrFailedDeleteAllCards,
			method,
			span,
		)
	}

	logSuccess("Successfully deleted all cards permanently")

	return true, nil
}
