package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/email"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	mencache "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/redis"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errorhandler"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/outbox"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/security"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type saldoCommandParams struct {
	Cache                  mencache.SaldoCommandCache
	saldoCommandRepository repository.SaldoCommandRepository
	CardAdapter            adapter.CardAdapter
	Logger                 logger.LoggerInterface
	Observability          observability.TraceLoggerObservability
	Kafka                  *kafka.Kafka
	Outbox                 outbox.Store[db.OutboxRecord]
}

type saldoCommandService struct {
	ctx                    context.Context
	cache                  mencache.SaldoCommandCache
	cardAdapter            adapter.CardAdapter
	logger                 logger.LoggerInterface
	saldoCommandRepository repository.SaldoCommandRepository
	observability          observability.TraceLoggerObservability
	kafka                  *kafka.Kafka
	outbox                 outbox.Store[db.OutboxRecord]
}

func NewSaldoCommandService(params *saldoCommandParams) SaldoCommandService {
	return &saldoCommandService{
		cache:                  params.Cache,
		saldoCommandRepository: params.saldoCommandRepository,
		cardAdapter:            params.CardAdapter,
		logger:                 params.Logger,
		observability:          params.Observability,
		kafka:                  params.Kafka,
		outbox:                 params.Outbox,
	}
}

func (s *saldoCommandService) CreateSaldo(ctx context.Context, request *requests.CreateSaldoRequest) (*db.CreateSaldoRow, error) {
	const method = "CreateSaldo"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status, "grpc") }()

	_, err := s.cardAdapter.FindCardByCardNumber(ctx, request.CardNumber)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.CreateSaldoRow](s.logger, err, method, span, zap.String("card_number", request.CardNumber))
	}

	res, err := s.saldoCommandRepository.CreateSaldo(ctx, request)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.CreateSaldoRow](s.logger, err, method, span, zap.String("card_number", request.CardNumber))
	}

	// Gap #5: publish the "Saldo Account Created" email so the email worker's
	// subscription on email-service-topic-saldo-create is not an orphan.
	// Best-effort: enrichment failures must never fail the already-committed
	// create. The topic is created in kafka-create-topic-email-job.yaml.
	s.publishSaldoCreatedEmail(ctx, request.CardNumber)

	logSuccess("Successfully created saldo record", zap.String("card_number", request.CardNumber), zap.Float64("amount", float64(request.TotalBalance)))

	return res, nil
}

func (s *saldoCommandService) publishSaldoCreatedEmail(ctx context.Context, cardNumber string) {
	if s.kafka == nil {
		return
	}

	card, err := s.cardAdapter.FindUserCardByCardNumber(ctx, cardNumber)
	if err != nil {
		s.logger.Warn("saldo email: failed to resolve user email for card", zap.Error(err), zap.String("card_number", cardNumber))
		return
	}
	if card == nil || card.Email == "" {
		s.logger.Warn("saldo email: no user email available for card", zap.String("card_number", cardNumber))
		return
	}

	htmlBody, err := email.GenerateEmailHTML(map[string]string{
		"Title":   "Saldo Account Created",
		"Message": fmt.Sprintf("Your payment account for card %s has been created successfully.", cardNumber),
		"Button":  "View Balance",
		"Link":    "https://sanedge.example.com/balance",
	})
	if err != nil {
		s.logger.Warn("saldo email: failed to generate HTML body", zap.Error(err), zap.String("card_number", cardNumber))
		return
	}

	emailPayload, err := json.Marshal(map[string]any{"email": card.Email, "subject": "Saldo Account Created - SanEdge", "body": htmlBody})
	if err != nil {
		s.logger.Warn("saldo email: failed to marshal payload", zap.Error(err), zap.String("card_number", cardNumber))
		return
	}

	if err := s.kafka.SendMessage("email-service-topic-saldo-create", cardNumber, emailPayload); err != nil {
		s.logger.Warn("saldo email: failed to publish create event", zap.Error(err), zap.String("card_number", cardNumber))
	}
}

func (s *saldoCommandService) CreateSaldoIfNotExists(ctx context.Context, request *requests.CreateSaldoRequest) error {
	return s.saldoCommandRepository.CreateSaldoIfNotExists(ctx, request)
}

// saldoChangedEvent is the durable balance-change event. Its JSON fields are a
// superset of events.SaldoEvent so the stats-writer consumer keeps working, and
// it carries saldo_id so the cache invalidator can drop the by-id key too.
type saldoChangedEvent struct {
	SaldoID      int64     `json:"saldo_id"`
	CardNumber   string    `json:"card_number"`
	TotalBalance int64     `json:"total_balance"`
	CreatedAt    time.Time `json:"created_at"`
}

// enqueueSaldoChanged is the single place that invalidates the saldo query
// cache after a balance change and publishes a durable saldo.changed event.
//
// P1.4 cache consistency: the fast path deletes both cache keys (by-id and
// by-card-number) synchronously after commit; the outbox row makes the change
// replayable so the async cache invalidator can also react after a crash, and
// the 5-minute TTL remains the safety net. Cache is never a source of truth -
// reads that miss go to the DB.
func (s *saldoCommandService) enqueueSaldoChanged(ctx context.Context, saldoID int64, cardNumber string, totalBalance int64) {
	s.cache.DeleteSaldoCache(ctx, int(saldoID))
	s.cache.DeleteSaldoCacheByCardNumber(ctx, cardNumber)

	if s.outbox == nil {
		return
	}

	payload, err := json.Marshal(saldoChangedEvent{
		SaldoID:      saldoID,
		CardNumber:   cardNumber,
		TotalBalance: totalBalance,
		CreatedAt:    time.Now(),
	})
	if err != nil {
		s.logger.Error("failed to marshal saldo.changed event", zap.Error(err))
		return
	}

	if err := s.outbox.Insert(ctx, db.OutboxRecord{
		AggregateType: "saldo",
		AggregateID:   cardNumber,
		EventType:     "saldo.changed",
		Payload:       payload,
	}); err != nil {
		s.logger.Error("outbox: failed to enqueue saldo.changed event",
			zap.Error(err),
			zap.String("card_number", security.MaskCardNumber(cardNumber)),
		)
	}
}

func (s *saldoCommandService) UpdateSaldo(ctx context.Context, request *requests.UpdateSaldoRequest) (*db.UpdateSaldoRow, error) {
	const method = "UpdateSaldo"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status, "grpc") }()

	_, err := s.cardAdapter.FindCardByCardNumber(ctx, request.CardNumber)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateSaldoRow](s.logger, err, method, span, zap.String("card_number", request.CardNumber))
	}

	res, err := s.saldoCommandRepository.UpdateSaldo(ctx, request)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateSaldoRow](s.logger, err, method, span, zap.String("card_number", request.CardNumber))
	}

	s.enqueueSaldoChanged(ctx, int64(res.SaldoID), res.CardNumber, res.TotalBalance)

	logSuccess("Successfully updated saldo record", zap.String("card_number", security.MaskCardNumber(request.CardNumber)), zap.Float64("amount", float64(request.TotalBalance)))

	return res, nil
}

func (s *saldoCommandService) DebitSaldo(ctx context.Context, request *requests.DebitSaldoRequest) (*db.DebitSaldoRow, error) {
	const method = "DebitSaldo"
	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status, "grpc") }()

	if err := request.Validate(); err != nil {
		status = "error"
		return errorhandler.HandleError[*db.DebitSaldoRow](s.logger, sharedErrors.NewBadRequestError("invalid debit saldo request").WithInternal(err), method, span)
	}

	res, err := s.saldoCommandRepository.DebitSaldo(ctx, request)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.DebitSaldoRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}
	s.enqueueSaldoChanged(ctx, int64(res.SaldoID), res.CardNumber, res.TotalBalance)
	logSuccess("Successfully debited saldo", zap.String("card_number", security.MaskCardNumber(request.CardNumber)), zap.Int("amount", request.Amount))
	return res, nil
}

func (s *saldoCommandService) CreditSaldo(ctx context.Context, request *requests.CreditSaldoRequest) (*db.CreditSaldoRow, error) {
	const method = "CreditSaldo"
	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status, "grpc") }()

	if err := request.Validate(); err != nil {
		status = "error"
		return errorhandler.HandleError[*db.CreditSaldoRow](s.logger, sharedErrors.NewBadRequestError("invalid credit saldo request").WithInternal(err), method, span)
	}

	res, err := s.saldoCommandRepository.CreditSaldo(ctx, request)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.CreditSaldoRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}
	s.enqueueSaldoChanged(ctx, int64(res.SaldoID), res.CardNumber, res.TotalBalance)
	logSuccess("Successfully credited saldo", zap.String("card_number", security.MaskCardNumber(request.CardNumber)), zap.Int("amount", request.Amount))
	return res, nil
}

func (s *saldoCommandService) ApplySaldoAdjustment(ctx context.Context, request *requests.ApplySaldoAdjustmentRequest) (*db.SaldoAdjustmentRow, error) {
	const method = "ApplySaldoAdjustment"
	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status, "grpc") }()

	if err := request.Validate(); err != nil {
		status = "error"
		return errorhandler.HandleError[*db.SaldoAdjustmentRow](s.logger, sharedErrors.NewBadRequestError("invalid saldo adjustment request").WithInternal(err), method, span)
	}
	res, err := s.saldoCommandRepository.ApplySaldoAdjustment(ctx, request)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.SaldoAdjustmentRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}
	s.enqueueSaldoChanged(ctx, int64(res.SaldoID), res.CardNumber, res.TotalBalance)
	logSuccess("Successfully applied saldo adjustment", zap.String("card_number", security.MaskCardNumber(request.CardNumber)), zap.Int64("delta", request.Delta))
	return res, nil
}

func (s *saldoCommandService) ResolveReconciliation(ctx context.Context, queueID int64, operationID, note string) error {
	return s.saldoCommandRepository.ResolveReconciliation(ctx, queueID, operationID, note)
}

func (s *saldoCommandService) UpdateSaldoWithdraw(ctx context.Context, request *requests.UpdateSaldoWithdraw) (*db.UpdateSaldoWithdrawRow, error) {
	const method = "UpdateSaldoWithdraw"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status, "grpc") }()

	_, err := s.cardAdapter.FindCardByCardNumber(ctx, request.CardNumber)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateSaldoWithdrawRow](s.logger, err, method, span, zap.String("card_number", request.CardNumber))
	}

	res, err := s.saldoCommandRepository.UpdateSaldoWithdraw(ctx, request)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateSaldoWithdrawRow](s.logger, err, method, span, zap.String("card_number", request.CardNumber))
	}

	s.enqueueSaldoChanged(ctx, int64(res.SaldoID), res.CardNumber, res.TotalBalance)

	logSuccess("Successfully updated saldo withdraw record", zap.String("card_number", security.MaskCardNumber(request.CardNumber)))

	return res, nil
}

func (s *saldoCommandService) TrashSaldo(ctx context.Context, saldo_id int) (*db.Saldo, error) {
	const method = "TrashSaldo"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("saldo_id", saldo_id))

	defer func() {
		end(status, "grpc")
	}()

	res, err := s.saldoCommandRepository.TrashedSaldo(ctx, saldo_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.Saldo](
			s.logger,
			err,
			method,
			span,

			zap.Int("saldo_id", saldo_id),
		)
	}

	logSuccess("Successfully trashed saldo", zap.Int("saldo_id", saldo_id))

	return res, nil
}

func (s *saldoCommandService) RestoreSaldo(ctx context.Context, saldo_id int) (*db.Saldo, error) {
	const method = "RestoreSaldo"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("saldo_id", saldo_id))

	defer func() {
		end(status, "grpc")
	}()

	res, err := s.saldoCommandRepository.RestoreSaldo(ctx, saldo_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.Saldo](
			s.logger,
			err,
			method,
			span,

			zap.Int("saldo_id", saldo_id),
		)
	}

	logSuccess("Successfully restored saldo", zap.Int("saldo_id", saldo_id))

	return res, nil
}

func (s *saldoCommandService) DeleteSaldoPermanent(ctx context.Context, saldo_id int) (bool, error) {
	const method = "DeleteSaldoPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("saldo_id", saldo_id))

	defer func() {
		end(status, "grpc")
	}()

	_, err := s.saldoCommandRepository.DeleteSaldoPermanent(ctx, saldo_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			err,
			method,
			span,

			zap.Int("saldo_id", saldo_id),
		)
	}

	logSuccess("Successfully deleted saldo permanently", zap.Int("saldo_id", saldo_id))

	return true, nil
}

func (s *saldoCommandService) RestoreAllSaldo(ctx context.Context) (bool, error) {
	const method = "RestoreAllSaldo"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status, "grpc")
	}()

	_, err := s.saldoCommandRepository.RestoreAllSaldo(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			sharedErrors.ErrFailed("restore all saldo"),
			method,
			span,
		)
	}

	logSuccess("Successfully restored all saldo")
	return true, nil
}

func (s *saldoCommandService) DeleteAllSaldoPermanent(ctx context.Context) (bool, error) {
	const method = "DeleteAllSaldoPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status, "grpc")
	}()

	_, err := s.saldoCommandRepository.DeleteAllSaldoPermanent(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			sharedErrors.ErrFailed("delete all saldo permanently"),
			method,
			span,
		)
	}

	logSuccess("Successfully deleted all saldo permanently")
	return true, nil
}
