package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/email"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	carddb "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/async"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/validation"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errorhandler"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/idempotency"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/security"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	mencache "github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/redis"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/topup/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/events"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// topupCommandDeps groups dependencies for top-up command service.
type topupCommandDeps struct {
	Kafka                  *kafka.Kafka
	Cache                  mencache.TopupCommandCache
	CardAdapter            adapter.CardAdapter
	TopupQueryRepository   repository.TopupQueryRepository
	TopupCommandRepository repository.TopupCommandRepository
	SaldoAdapter           adapter.SaldoAdapter
	IdempotencyStore       idempotency.Store
	OutboxStore            repository.OutboxRepository
	Logger                 logger.LoggerInterface
	Observability          observability.TraceLoggerObservability
}

// topupCommandService handles top-up command operations.
type topupCommandService struct {
	kafka                  *kafka.Kafka
	cache                  mencache.TopupCommandCache
	topupQueryRepository   repository.TopupQueryRepository
	cardAdapter            adapter.CardAdapter
	topupCommandRepository repository.TopupCommandRepository
	saldoAdapter           adapter.SaldoAdapter
	idempotencyStore       idempotency.Store
	outboxStore            repository.OutboxRepository
	logger                 logger.LoggerInterface
	observability          observability.TraceLoggerObservability
}

func NewTopupCommandService(
	params *topupCommandDeps,
) TopupCommandService {
	return &topupCommandService{
		kafka:                  params.Kafka,
		cache:                  params.Cache,
		topupQueryRepository:   params.TopupQueryRepository,
		topupCommandRepository: params.TopupCommandRepository,
		saldoAdapter:           params.SaldoAdapter,
		idempotencyStore:       params.IdempotencyStore,
		outboxStore:            params.OutboxStore,
		cardAdapter:            params.CardAdapter,
		logger:                 params.Logger,
		observability:          params.Observability,
	}
}

func (s *topupCommandService) CreateTopup(ctx context.Context, request *requests.CreateTopupRequest) (*db.UpdateTopupStatusRow, error) {
	const method = "CreateTopup"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status, "grpc") }()

	// Idempotency guard: claim the client retry key before any balance mutation.
	// Retries with the same key and payload replay the stored response; the same
	// key with a different payload is rejected so money is never moved twice.
	var idemHash string
	if request.IdempotencyKey != "" && s.idempotencyStore != nil {
		idemHash = idempotency.HashRequest(request)
		if _, err := s.idempotencyStore.Claim(ctx, "topup", request.IdempotencyKey, idemHash); err == nil {
			defer func() {
				if status == "error" {
					if relErr := s.idempotencyStore.Release(ctx, "topup", request.IdempotencyKey, idemHash); relErr != nil {
						s.logger.Error("idempotency: failed to release key", zap.Error(relErr), zap.String("idempotency_key", request.IdempotencyKey))
					}
				}
			}()
		} else {
			existing, gErr := s.idempotencyStore.Get(ctx, "topup", request.IdempotencyKey)
			if gErr != nil {
				status = "error"
				return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, gErr, method, span, zap.String("idempotency_key", request.IdempotencyKey))
			}
			if existing.RequestHash != idemHash {
				status = "error"
				return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, idempotency.ConflictError(), method, span, zap.String("idempotency_key", request.IdempotencyKey))
			}
			if existing.Status == idempotency.StatusSuccess {
				var row db.UpdateTopupStatusRow
				if uErr := json.Unmarshal(existing.ResponseJSON, &row); uErr != nil {
					status = "error"
					return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, uErr, method, span, zap.String("idempotency_key", request.IdempotencyKey))
				}
				logSuccess("Replayed idempotent topup response", zap.String("idempotency_key", request.IdempotencyKey))
				return &row, nil
			}
			status = "error"
			return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, idempotency.ProcessingError(), method, span, zap.String("idempotency_key", request.IdempotencyKey))
		}
	}

	// P1.7: reject non-positive / overflowing amounts before any balance
	// mutation or remote call.
	if err := validation.ValidateAmount(request.TopupAmount); err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}

	// Money movement only needs card identity and ownership metadata. Do not make
	// a valid top-up depend on synchronous user/email enrichment; notifications
	// are best-effort and can be retried by their consumer.
	cardIdentity, err := s.cardAdapter.FindCardByCardNumber(ctx, request.CardNumber)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}
	if request.AuthenticatedUserID > 0 && int(cardIdentity.UserID) != request.AuthenticatedUserID {
		status = "error"
		return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, sharedErrors.NewForbiddenError("card does not belong to authenticated user"), method, span)
	}
	card := &carddb.GetUserEmailByCardNumberRow{
		CardID: cardIdentity.CardID, UserID: cardIdentity.UserID,
		CardNumber: cardIdentity.CardNumber, CardType: cardIdentity.CardType,
		ExpireDate: cardIdentity.ExpireDate, Cvv: cardIdentity.Cvv,
		CardProvider: cardIdentity.CardProvider, CreatedAt: cardIdentity.CreatedAt,
		UpdatedAt: cardIdentity.UpdatedAt,
	}

	topup, err := s.topupCommandRepository.CreateTopup(ctx, request)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}

	creditedSaldo, err := s.saldoAdapter.CreditSaldo(ctx, &requests.CreditSaldoRequest{
		CardNumber:  request.CardNumber,
		Amount:      request.TopupAmount,
		OperationID: "topup:" + strconv.Itoa(int(topup.TopupID)) + ":credit",
		SourceType:  "topup",
		SourceID:    strconv.Itoa(int(topup.TopupID)),
	})
	if err != nil {
		status = "error"
		s.markTopupAsFailed(ctx, int(topup.TopupID), method, span)
		return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}
	newBalance := int(creditedSaldo.TotalBalance)

	expireDate := card.ExpireDate.Time

	_, err = s.cardAdapter.UpdateCard(ctx, &requests.UpdateCardRequest{
		CardID:       int(card.CardID),
		UserID:       int(card.UserID),
		CardType:     card.CardType,
		ExpireDate:   expireDate,
		CVV:          card.Cvv,
		CardProvider: card.CardProvider,
	})
	if err != nil {
		status = "error"
		_, rollbackErr := s.saldoAdapter.DebitSaldo(ctx, &requests.DebitSaldoRequest{
			CardNumber:  request.CardNumber,
			Amount:      request.TopupAmount,
			OperationID: "topup:" + strconv.Itoa(int(topup.TopupID)) + ":rollback",
			SourceType:  "topup_compensation",
			SourceID:    strconv.Itoa(int(topup.TopupID)),
		})
		if rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, rollbackErr, method, span, zap.String("rollback_for", "saldo"))
		}
		s.markTopupAsFailed(ctx, int(topup.TopupID), method, span)
		return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}

	updatedTopup, err := s.topupCommandRepository.UpdateTopupStatus(ctx, &requests.UpdateTopupStatus{
		TopupID: int(topup.TopupID),
		Status:  "success",
	})
	if err != nil {
		status = "error"
		_, rollbackErr := s.saldoAdapter.DebitSaldo(ctx, &requests.DebitSaldoRequest{
			CardNumber:  request.CardNumber,
			Amount:      request.TopupAmount,
			OperationID: "topup:" + strconv.Itoa(int(topup.TopupID)) + ":rollback",
			SourceType:  "topup_compensation",
			SourceID:    strconv.Itoa(int(topup.TopupID)),
		})
		if rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, rollbackErr, method, span, zap.String("rollback_for", "saldo"))
		}
		s.markTopupAsFailed(ctx, int(topup.TopupID), method, span)
		return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, err, method, span, zap.Int("topup_id", int(topup.TopupID)))
	}

	// Phase 3: Outbox instead of fire-and-forget goroutine
	s.enqueueTopupEvents(ctx, topup, updatedTopup, card, request, newBalance)

	logSuccess("Topup created successfully", zap.String("cardNumber", security.MaskCardNumber(request.CardNumber)), zap.Int("topupID", int(topup.TopupID)), zap.Float64("topupAmount", float64(request.TopupAmount)))

	if request.IdempotencyKey != "" && s.idempotencyStore != nil {
		if respBytes, mErr := json.Marshal(updatedTopup); mErr == nil {
			resourceID := updatedTopup.TopupID
			if cErr := s.idempotencyStore.Complete(ctx, "topup", request.IdempotencyKey, idemHash, idempotency.Outcome{
				Status:       idempotency.StatusSuccess,
				ResponseJSON: respBytes,
				ResourceID:   &resourceID,
			}); cErr != nil {
				s.logger.Error("idempotency: failed to complete key", zap.Error(cErr), zap.String("idempotency_key", request.IdempotencyKey))
			}
		}
	}

	return updatedTopup, nil
}

func (s *topupCommandService) UpdateTopup(ctx context.Context, request *requests.UpdateTopupRequest) (*db.UpdateTopupStatusRow, error) {
	const method = "UpdateTopup"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status, "grpc") }()

	_, err := s.cardAdapter.FindCardByCardNumber(ctx, request.CardNumber)
	if err != nil {
		status = "error"
		s.markTopupAsFailed(ctx, *request.TopupID, method, span)
		return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}

	existingTopup, err := s.topupQueryRepository.FindById(ctx, *request.TopupID)
	if err != nil {
		status = "error"
		s.markTopupAsFailed(ctx, *request.TopupID, method, span)
		return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, err, method, span, zap.Int("topup_id", *request.TopupID))
	}

	_, err = s.topupCommandRepository.UpdateTopup(ctx, request)
	if err != nil {
		status = "error"
		s.markTopupAsFailed(ctx, *request.TopupID, method, span)
		return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, err, method, span, zap.Int("topup_id", *request.TopupID))
	}

	topupDifference := request.TopupAmount - int(existingTopup.TopupAmount)
	if topupDifference > 0 {
		_, creditErr := s.saldoAdapter.CreditSaldo(ctx, &requests.CreditSaldoRequest{
			CardNumber:  request.CardNumber,
			Amount:      topupDifference,
			OperationID: "topup:update:" + strconv.Itoa(*request.TopupID) + ":credit",
			SourceType:  "topup_update",
			SourceID:    strconv.Itoa(*request.TopupID),
		})
		err = creditErr
	} else if topupDifference < 0 {
		_, debitErr := s.saldoAdapter.DebitSaldo(ctx, &requests.DebitSaldoRequest{
			CardNumber:  request.CardNumber,
			Amount:      -topupDifference,
			OperationID: "topup:update:" + strconv.Itoa(*request.TopupID) + ":debit",
			SourceType:  "topup_update",
			SourceID:    strconv.Itoa(*request.TopupID),
		})
		err = debitErr
	}
	if err != nil {
		status = "error"
		// 6. Jalankan logika rollback
		_, rollbackErr := s.topupCommandRepository.UpdateTopupAmount(ctx, &requests.UpdateTopupAmount{
			TopupID:     *request.TopupID,
			TopupAmount: int(existingTopup.TopupAmount),
		})
		if rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, rollbackErr, method, span, zap.Int("topup_id", *request.TopupID))
		}
		s.markTopupAsFailed(ctx, *request.TopupID, method, span)
		return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}

	updatedTopup, err := s.topupCommandRepository.UpdateTopupStatus(ctx, &requests.UpdateTopupStatus{
		TopupID: *request.TopupID,
		Status:  "success",
	})
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateTopupStatusRow](s.logger, err, method, span, zap.Int("topup_id", *request.TopupID))
	}

	logSuccess("UpdateTopup process completed", zap.Int("topup_id", *request.TopupID))

	return updatedTopup, nil
}

func (s *topupCommandService) TrashedTopup(ctx context.Context, topup_id int) (*db.Topup, error) {
	const method = "TrashedTopup"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("topup_id", topup_id))

	defer func() {
		end(status, "grpc")
	}()

	s.logger.Debug("Starting TrashedTopup process", zap.Int("topup_id", topup_id))

	res, err := s.topupCommandRepository.TrashedTopup(ctx, topup_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.Topup](
			s.logger,
			err,
			method,
			span,

			zap.Int("topup_id", topup_id),
		)
	}

	logSuccess("TrashedTopup process completed", zap.Int("topup_id", topup_id))

	return res, nil
}

func (s *topupCommandService) RestoreTopup(ctx context.Context, topup_id int) (*db.Topup, error) {
	const method = "RestoreTopup"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("topup_id", topup_id))

	defer func() {
		end(status, "grpc")
	}()

	s.logger.Debug("Starting RestoreTopup process", zap.Int("topup_id", topup_id))

	res, err := s.topupCommandRepository.RestoreTopup(ctx, topup_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.Topup](
			s.logger,
			err,
			method,
			span,

			zap.Int("topup_id", topup_id),
		)
	}

	logSuccess("RestoreTopup process completed", zap.Int("topup_id", topup_id))

	return res, nil
}

func (s *topupCommandService) DeleteTopupPermanent(ctx context.Context, topup_id int) (bool, error) {
	const method = "DeleteTopupPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("topup_id", topup_id))

	defer func() {
		end(status, "grpc")
	}()

	s.logger.Debug("Starting DeleteTopupPermanent process", zap.Int("topup_id", topup_id))

	_, err := s.topupCommandRepository.DeleteTopupPermanent(ctx, topup_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			err,
			method,
			span,

			zap.Int("topup_id", topup_id),
		)
	}

	logSuccess("DeleteTopupPermanent process completed", zap.Int("topup_id", topup_id))

	return true, nil
}

func (s *topupCommandService) RestoreAllTopup(ctx context.Context) (bool, error) {
	const method = "RestoreAllTopup"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status, "grpc")
	}()

	s.logger.Debug("Restoring all topups")

	_, err := s.topupCommandRepository.RestoreAllTopup(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			sharedErrors.ErrFailed("restore all topups"),
			method,
			span,
		)
	}

	logSuccess("Successfully restored all topups")
	return true, nil
}

func (s *topupCommandService) DeleteAllTopupPermanent(ctx context.Context) (bool, error) {
	const method = "DeleteAllTopupPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status, "grpc")
	}()

	s.logger.Debug("Permanently deleting all topups")

	_, err := s.topupCommandRepository.DeleteAllTopupPermanent(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			sharedErrors.ErrFailed("delete all topups permanently"),
			method,
			span,
		)
	}

	logSuccess("Successfully deleted all topups permanently")
	return true, nil
}

func (s *topupCommandService) enqueueTopupEvents(ctx context.Context, topup *db.CreateTopupRow, updatedTopup *db.UpdateTopupStatusRow, card *carddb.GetUserEmailByCardNumberRow, request *requests.CreateTopupRequest, newBalance int) {
	if s.outboxStore == nil {
		return
	}
	topupID := strconv.Itoa(int(topup.TopupID))

	// Email enrichment is intentionally outside the money path. Older callers
	// may still provide an email, but an empty recipient must not create a
	// malformed notification or fail the already-committed top-up.
	if card.Email != "" {
		htmlBody, err := email.GenerateEmailHTML(map[string]string{
			"Title":   "Topup Successful",
			"Message": fmt.Sprintf("Your topup of %d has been processed successfully.", request.TopupAmount),
			"Button":  "View History",
			"Link":    "https://sanedge.example.com/topup/history",
		})
		if err == nil {
			emailPayload, _ := json.Marshal(map[string]any{"email": card.Email, "subject": "Topup Successful - SanEdge", "body": htmlBody})
			if iErr := s.outboxStore.Insert(ctx, db.OutboxRecord{
				AggregateType: "topup", AggregateID: topupID,
				EventType: "topup.created", Payload: emailPayload,
			}); iErr != nil {
				s.logger.Error("outbox: failed to enqueue topup email event", zap.Error(iErr))
			}
		}
	}

	// Stats event
	statsEvent := events.TopupEvent{
		TopupID:       uint64(updatedTopup.TopupID),
		TopupNo:       fmt.Sprintf("%x-%x-%x-%x-%x", updatedTopup.TopupNo.Bytes[0:4], updatedTopup.TopupNo.Bytes[4:6], updatedTopup.TopupNo.Bytes[6:8], updatedTopup.TopupNo.Bytes[8:10], updatedTopup.TopupNo.Bytes[10:16]),
		CardNumber:    request.CardNumber,
		CardType:      card.CardType,
		CardProvider:  card.CardProvider,
		Amount:        int64(request.TopupAmount),
		PaymentMethod: request.TopupMethod,
		Status:        "success",
		CreatedAt:     time.Now(),
	}
	statsBytes, _ := json.Marshal(statsEvent)
	if iErr := s.outboxStore.Insert(ctx, db.OutboxRecord{
		AggregateType: "topup", AggregateID: topupID,
		EventType: "topup.stats", Payload: statsBytes,
	}); iErr != nil {
		s.logger.Error("outbox: failed to enqueue topup stats event", zap.Error(iErr))
	}

	// Saldo event
	saldoEvent := events.SaldoEvent{CardNumber: request.CardNumber, TotalBalance: int64(newBalance), CreatedAt: time.Now()}
	saldoBytes, _ := json.Marshal(saldoEvent)
	if iErr := s.outboxStore.Insert(ctx, db.OutboxRecord{
		AggregateType: "topup", AggregateID: request.CardNumber,
		EventType: "topup.saldo", Payload: saldoBytes,
	}); iErr != nil {
		s.logger.Error("outbox: failed to enqueue topup saldo event", zap.Error(iErr))
	}
}

func (s *topupCommandService) markTopupAsFailed(ctx context.Context, topupID int, method string, span trace.Span) {
	req := requests.UpdateTopupStatus{
		TopupID: topupID,
		Status:  "failed",
	}
	// P0.4: detached context with fixed timeout - never the request context,
	// which may be cancelled by the client before the goroutine runs.
	async.RunWithTimeout(5*time.Second, func(ctx context.Context) {
		if _, err := s.topupCommandRepository.UpdateTopupStatus(ctx, &req); err != nil {
			s.logger.Error("compensation: failed to mark topup as failed", zap.Error(err), zap.Int("topup_id", topupID), zap.String("method", method))
		}
	})
}
