package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/MamangRust/microservice-payment-gateway-grpc/pb/ai_security"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/email"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	carddb "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/database/schema"
	mencache "github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/redis"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/async"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/events"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/validation"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errorhandler"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/idempotency"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/security"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// withdrawCommandServiceDeps defines dependencies for withdrawCommandService.
type withdrawCommandServiceDeps struct {
	Cache mencache.WithdrawCommandCache
	Kafka *kafka.Kafka

	CardAdapter       adapter.CardAdapter
	SaldoAdapter      adapter.SaldoAdapter
	CommandRepository repository.WithdrawCommandRepository
	QueryRepository   repository.WithdrawQueryRepository
	IdempotencyStore  idempotency.Store
	OutboxStore       repository.OutboxRepository

	Logger               logger.LoggerInterface
	Observability        observability.TraceLoggerObservability
	AISecurityClient     ai_security.AISecurityServiceClient
	DailyWithdrawalLimit int64
}

// withdrawCommandService handles command-side withdraw operations.
type withdrawCommandService struct {
	cache mencache.WithdrawCommandCache
	kafka *kafka.Kafka

	cardAdapter  adapter.CardAdapter
	saldoAdapter adapter.SaldoAdapter

	withdrawCommandRepository repository.WithdrawCommandRepository
	withdrawQueryRepository   repository.WithdrawQueryRepository
	idempotencyStore          idempotency.Store
	outboxStore               repository.OutboxRepository

	logger               logger.LoggerInterface
	observability        observability.TraceLoggerObservability
	aiSecurityClient     ai_security.AISecurityServiceClient
	dailyWithdrawalLimit int64
}

func NewWithdrawCommandService(
	deps *withdrawCommandServiceDeps,
) WithdrawCommandService {
	return &withdrawCommandService{
		kafka:                     deps.Kafka,
		cache:                     deps.Cache,
		cardAdapter:               deps.CardAdapter,
		saldoAdapter:              deps.SaldoAdapter,
		withdrawCommandRepository: deps.CommandRepository,
		withdrawQueryRepository:   deps.QueryRepository,
		idempotencyStore:          deps.IdempotencyStore,
		outboxStore:               deps.OutboxStore,
		logger:                    deps.Logger,
		observability:             deps.Observability,
		aiSecurityClient:          deps.AISecurityClient,
		dailyWithdrawalLimit:      deps.DailyWithdrawalLimit,
	}
}

func (s *withdrawCommandService) Create(ctx context.Context, request *requests.CreateWithdrawRequest) (*db.UpdateWithdrawStatusRow, error) {
	const method = "CreateWithdraw"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status, "grpc") }()

	// Idempotency guard: claim the client retry key before any balance mutation.
	// Retries with the same key and payload replay the stored response; the same
	// key with a different payload is rejected so money is never moved twice.
	var idemHash string
	if request.IdempotencyKey != "" && s.idempotencyStore != nil {
		idemHash = idempotency.HashRequest(request)
		if _, err := s.idempotencyStore.Claim(ctx, "withdraw", request.IdempotencyKey, idemHash); err == nil {
			defer func() {
				if status == "error" {
					if relErr := s.idempotencyStore.Release(ctx, "withdraw", request.IdempotencyKey, idemHash); relErr != nil {
						s.logger.Error("idempotency: failed to release key", zap.Error(relErr), zap.String("idempotency_key", request.IdempotencyKey))
					}
				}
			}()
		} else {
			existing, gErr := s.idempotencyStore.Get(ctx, "withdraw", request.IdempotencyKey)
			if gErr != nil {
				status = "error"
				return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, gErr, method, span, zap.String("idempotency_key", request.IdempotencyKey))
			}
			if existing.RequestHash != idemHash {
				status = "error"
				return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, idempotency.ConflictError(), method, span, zap.String("idempotency_key", request.IdempotencyKey))
			}
			if existing.Status == idempotency.StatusSuccess {
				var row db.UpdateWithdrawStatusRow
				if uErr := json.Unmarshal(existing.ResponseJSON, &row); uErr != nil {
					status = "error"
					return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, uErr, method, span, zap.String("idempotency_key", request.IdempotencyKey))
				}
				logSuccess("Replayed idempotent withdraw response", zap.String("idempotency_key", request.IdempotencyKey))
				return &row, nil
			}
			status = "error"
			return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, idempotency.ProcessingError(), method, span, zap.String("idempotency_key", request.IdempotencyKey))
		}
	}

	// P1.7: reject non-positive / overflowing amounts before any balance
	// mutation or remote call.
	if err := validation.ValidateAmount(request.WithdrawAmount); err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}

	// Keep the critical money path independent from user/email enrichment. The
	// card service identity lookup still returns owner metadata for authorization
	// at the gateway/service boundary and for the stats event.
	cardIdentity, err := s.cardAdapter.FindCardByCardNumber(ctx, request.CardNumber)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}
	if request.AuthenticatedUserID > 0 && int(cardIdentity.UserID) != request.AuthenticatedUserID {
		status = "error"
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, sharedErrors.NewForbiddenError("card does not belong to authenticated user"), method, span)
	}
	card := &carddb.GetUserEmailByCardNumberRow{
		CardID: cardIdentity.CardID, UserID: cardIdentity.UserID,
		CardNumber: cardIdentity.CardNumber, CardType: cardIdentity.CardType,
		ExpireDate: cardIdentity.ExpireDate, Cvv: cardIdentity.Cvv,
		CardProvider: cardIdentity.CardProvider, CreatedAt: cardIdentity.CreatedAt,
		UpdatedAt: cardIdentity.UpdatedAt,
	}

	saldo, err := s.saldoAdapter.FindByCardNumber(ctx, request.CardNumber)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}

	if int(saldo.TotalBalance) < request.WithdrawAmount {
		status = "error"
		// P1.5 error contract: insufficient balance is a domain conflict (409),
		// not an internal error.
		err := sharedErrors.ErrConflict.WithMessage("insufficient balance")
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span, zap.Float64("balance", float64(saldo.TotalBalance)), zap.Float64("amount", float64(request.WithdrawAmount)))
	}

	// Daily withdrawal limit guard (WITHDRAW_DAILY_LIMIT): today's successful
	// withdrawals plus the new amount must not exceed the configured limit.
	// Checked before any balance mutation so an over-limit request is a no-op.
	if s.dailyWithdrawalLimit > 0 {
		todaySum, err := s.withdrawQueryRepository.GetTodayWithdrawSumByCardNumber(ctx, request.CardNumber)
		if err != nil {
			status = "error"
			return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
		}
		if todaySum+int64(request.WithdrawAmount) > s.dailyWithdrawalLimit {
			status = "error"
			return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, sharedErrors.NewBadRequestError("Daily withdrawal limit exceeded"), method, span,
				zap.Int64("today_withdraw_sum", todaySum), zap.Int("withdraw_amount", request.WithdrawAmount), zap.Int64("limit", s.dailyWithdrawalLimit))
		}
	}

	// AI Security Check
	if s.aiSecurityClient != nil {
		secRes, err := s.aiSecurityClient.VerifySecurity(ctx, &ai_security.SecurityRequest{
			Domain:   ai_security.SecurityDomain_WITHDRAW,
			EntityId: request.CardNumber,
			Amount:   float64(request.WithdrawAmount),
		})
		if err == nil && !secRes.IsSafe {
			status = "error"
			s.logger.Warn("Withdrawal blocked by AI Security", zap.String("reason", secRes.Reason))
			return nil, errors.New("security block: " + secRes.Reason)
		}
	}

	// Gap #6 decision: metadata fields withdraw_amount/withdraw_time on the saldo
	// record are legacy from the monolith and are NOT consumed by any business
	// logic (only surfaced in API responses). UpdateSaldoWithdraw is intentionally
	// NOT called here because it performs its own atomic DebitSaldo internally -
	// chaining it after this debit would double-debit the balance. The ledger entry
	// from DebitSaldo plus the withdraws table row are the source of truth.
	debitedSaldo, err := s.saldoAdapter.DebitSaldo(ctx, &requests.DebitSaldoRequest{
		CardNumber:  request.CardNumber,
		Amount:      request.WithdrawAmount,
		OperationID: "withdraw:" + request.CardNumber + ":" + request.WithdrawTime.UTC().Format(time.RFC3339Nano) + ":" + strconv.Itoa(request.WithdrawAmount),
		SourceType:  "withdraw",
		SourceID:    request.CardNumber,
	})
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}
	newTotalBalance := int(debitedSaldo.TotalBalance)
	withdrawRecord, err := s.withdrawCommandRepository.CreateWithdraw(ctx, request)
	if err != nil {
		status = "error"
		if _, rollbackErr := s.saldoAdapter.CreditSaldo(ctx, &requests.CreditSaldoRequest{
			CardNumber:  request.CardNumber,
			Amount:      request.WithdrawAmount,
			OperationID: "withdraw:" + request.CardNumber + ":" + request.WithdrawTime.UTC().Format(time.RFC3339Nano) + ":" + strconv.Itoa(request.WithdrawAmount) + ":rollback-create",
			SourceType:  "withdraw_compensation",
			SourceID:    request.CardNumber,
		}); rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, rollbackErr, method, span, zap.String("rollback_for", "saldo"))
		}
		if withdrawRecord != nil {
			s.markWithdrawAsFailed(ctx, int(withdrawRecord.WithdrawID), method, span)
		}
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span)
	}

	updatedWithdraw, err := s.withdrawCommandRepository.UpdateWithdrawStatus(ctx, &requests.UpdateWithdrawStatus{
		WithdrawID: int(withdrawRecord.WithdrawID),
		Status:     "success",
	})
	if err != nil {
		status = "error"
		_, _ = s.saldoAdapter.CreditSaldo(ctx, &requests.CreditSaldoRequest{
			CardNumber:  request.CardNumber,
			Amount:      request.WithdrawAmount,
			OperationID: "withdraw:" + request.CardNumber + ":" + request.WithdrawTime.UTC().Format(time.RFC3339Nano) + ":" + strconv.Itoa(request.WithdrawAmount) + ":rollback-status",
			SourceType:  "withdraw_compensation",
			SourceID:    request.CardNumber,
		})
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span, zap.Int("withdraw_id", int(withdrawRecord.WithdrawID)))
	}

	// Phase 3: Outbox instead of fire-and-forget goroutine
	s.enqueueWithdrawEvents(ctx, withdrawRecord, updatedWithdraw, card, request, newTotalBalance)

	logSuccess("Successfully created withdraw", zap.Int("withdraw.id", int(updatedWithdraw.WithdrawID)))

	if request.IdempotencyKey != "" && s.idempotencyStore != nil {
		if respBytes, mErr := json.Marshal(updatedWithdraw); mErr == nil {
			resourceID := updatedWithdraw.WithdrawID
			if cErr := s.idempotencyStore.Complete(ctx, "withdraw", request.IdempotencyKey, idemHash, idempotency.Outcome{
				Status:       idempotency.StatusSuccess,
				ResponseJSON: respBytes,
				ResourceID:   &resourceID,
			}); cErr != nil {
				s.logger.Error("idempotency: failed to complete key", zap.Error(cErr), zap.String("idempotency_key", request.IdempotencyKey))
			}
		}
	}

	return updatedWithdraw, nil
}

func (s *withdrawCommandService) Update(ctx context.Context, request *requests.UpdateWithdrawRequest) (*db.UpdateWithdrawRow, error) {
	const method = "UpdateWithdraw"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status, "grpc") }()

	// P1.7: reject invalid amounts before adjusting balances.
	if err := validation.ValidateAmount(request.WithdrawAmount); err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}

	oldWithdraw, err := s.withdrawQueryRepository.FindById(ctx, *request.WithdrawID)
	if err != nil {
		status = "error"
		s.markWithdrawAsFailed(ctx, *request.WithdrawID, method, span)
		return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, err, method, span, zap.Int("withdraw_id", *request.WithdrawID))
	}

	saldo, err := s.saldoAdapter.FindByCardNumber(ctx, request.CardNumber)
	if err != nil {
		status = "error"
		s.markWithdrawAsFailed(ctx, *request.WithdrawID, method, span)
		return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}

	amountDifference := request.WithdrawAmount - int(oldWithdraw.WithdrawAmount)
	if amountDifference > int(saldo.TotalBalance) {
		status = "error"
		// P1.5 error contract: insufficient balance is a domain conflict (409).
		err := sharedErrors.ErrConflict.WithMessage("insufficient balance for update")
		s.markWithdrawAsFailed(ctx, *request.WithdrawID, method, span)
		return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, err, method, span, zap.Float64("balance", float64(saldo.TotalBalance)), zap.Float64("amount_difference", float64(amountDifference)))
	}

	// Daily withdrawal limit guard applies to the positive delta: increasing the
	// amount must not push today's total past WITHDRAW_DAILY_LIMIT.
	if s.dailyWithdrawalLimit > 0 && amountDifference > 0 {
		todaySum, err := s.withdrawQueryRepository.GetTodayWithdrawSumByCardNumber(ctx, request.CardNumber)
		if err != nil {
			status = "error"
			s.markWithdrawAsFailed(ctx, *request.WithdrawID, method, span)
			return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
		}
		if todaySum+int64(amountDifference) > s.dailyWithdrawalLimit {
			status = "error"
			s.markWithdrawAsFailed(ctx, *request.WithdrawID, method, span)
			return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, sharedErrors.NewBadRequestError("Daily withdrawal limit exceeded"), method, span,
				zap.Int64("today_withdraw_sum", todaySum), zap.Int("amount_difference", amountDifference), zap.Int64("limit", s.dailyWithdrawalLimit))
		}
	}
	if amountDifference > 0 {
		_, debitErr := s.saldoAdapter.DebitSaldo(ctx, &requests.DebitSaldoRequest{
			CardNumber:  request.CardNumber,
			Amount:      amountDifference,
			OperationID: fmt.Sprintf("withdraw:update:%d:%d:debit", *request.WithdrawID, request.WithdrawAmount),
			SourceType:  "withdraw_update",
			SourceID:    strconv.Itoa(*request.WithdrawID),
		})
		err = debitErr
	} else if amountDifference < 0 {
		_, creditErr := s.saldoAdapter.CreditSaldo(ctx, &requests.CreditSaldoRequest{
			CardNumber:  request.CardNumber,
			Amount:      -amountDifference,
			OperationID: fmt.Sprintf("withdraw:update:%d:%d:credit", *request.WithdrawID, request.WithdrawAmount),
			SourceType:  "withdraw_update",
			SourceID:    strconv.Itoa(*request.WithdrawID),
		})
		err = creditErr
	}
	if err != nil {
		status = "error"
		s.markWithdrawAsFailed(ctx, *request.WithdrawID, method, span)
		return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, err, method, span, zap.String("card_number", security.MaskCardNumber(request.CardNumber)))
	}

	updatedWithdraw, err := s.withdrawCommandRepository.UpdateWithdraw(ctx, request)
	if err != nil {
		status = "error"
		if amountDifference > 0 {
			_, rollbackErr := s.saldoAdapter.CreditSaldo(ctx, &requests.CreditSaldoRequest{CardNumber: saldo.CardNumber, Amount: amountDifference, OperationID: fmt.Sprintf("withdraw:update:%d:%d:rollback-credit", *request.WithdrawID, request.WithdrawAmount), SourceType: "withdraw_update_compensation", SourceID: strconv.Itoa(*request.WithdrawID)})
			if rollbackErr != nil {
				return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, rollbackErr, method, span, zap.String("rollback_for", "saldo"))
			}
		} else if amountDifference < 0 {
			_, rollbackErr := s.saldoAdapter.DebitSaldo(ctx, &requests.DebitSaldoRequest{CardNumber: saldo.CardNumber, Amount: -amountDifference, OperationID: fmt.Sprintf("withdraw:update:%d:%d:rollback-debit", *request.WithdrawID, request.WithdrawAmount), SourceType: "withdraw_update_compensation", SourceID: strconv.Itoa(*request.WithdrawID)})
			if rollbackErr != nil {
				return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, rollbackErr, method, span, zap.String("rollback_for", "saldo"))
			}
		}
		s.markWithdrawAsFailed(ctx, *request.WithdrawID, method, span)
		return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, err, method, span, zap.Int("withdraw_id", *request.WithdrawID))
	}

	if _, err := s.withdrawCommandRepository.UpdateWithdrawStatus(ctx, &requests.UpdateWithdrawStatus{
		WithdrawID: int(updatedWithdraw.WithdrawID),
		Status:     "success",
	}); err != nil {
		status = "error"
		if amountDifference > 0 {
			_, _ = s.saldoAdapter.CreditSaldo(ctx, &requests.CreditSaldoRequest{CardNumber: saldo.CardNumber, Amount: amountDifference, OperationID: fmt.Sprintf("withdraw:update:%d:%d:rollback-credit-status", *request.WithdrawID, request.WithdrawAmount), SourceType: "withdraw_update_compensation", SourceID: strconv.Itoa(*request.WithdrawID)})
		} else if amountDifference < 0 {
			_, _ = s.saldoAdapter.DebitSaldo(ctx, &requests.DebitSaldoRequest{CardNumber: saldo.CardNumber, Amount: -amountDifference, OperationID: fmt.Sprintf("withdraw:update:%d:%d:rollback-debit-status", *request.WithdrawID, request.WithdrawAmount), SourceType: "withdraw_update_compensation", SourceID: strconv.Itoa(*request.WithdrawID)})
		}
		return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, err, method, span, zap.Int("withdraw_id", int(updatedWithdraw.WithdrawID)))
	}

	logSuccess("Successfully updated withdraw", zap.Int("withdraw.id", int(updatedWithdraw.WithdrawID)))

	return updatedWithdraw, nil
}

func (s *withdrawCommandService) TrashedWithdraw(ctx context.Context, withdraw_id int) (*db.Withdraw, error) {
	const method = "TrashedWithdraw"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("withdraw_id", withdraw_id))

	defer func() {
		end(status, "grpc")
	}()

	s.logger.Debug("Trashing withdraw", zap.Int("withdraw_id", withdraw_id))

	res, err := s.withdrawCommandRepository.TrashedWithdraw(ctx, withdraw_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.Withdraw](
			s.logger,
			err,
			method,
			span,

			zap.Int("withdraw_id", withdraw_id),
		)
	}

	logSuccess("Successfully trashed withdraw", zap.Int("withdraw_id", withdraw_id))

	return res, nil
}

func (s *withdrawCommandService) RestoreWithdraw(ctx context.Context, withdraw_id int) (*db.Withdraw, error) {
	const method = "RestoreWithdraw"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("withdraw_id", withdraw_id))

	defer func() {
		end(status, "grpc")
	}()

	s.logger.Debug("Restoring withdraw", zap.Int("withdraw_id", withdraw_id))

	res, err := s.withdrawCommandRepository.RestoreWithdraw(ctx, withdraw_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.Withdraw](
			s.logger,
			err,
			method,
			span,

			zap.Int("withdraw_id", withdraw_id),
		)
	}

	logSuccess("Successfully restored withdraw", zap.Int("withdraw_id", withdraw_id))

	return res, nil
}

func (s *withdrawCommandService) DeleteWithdrawPermanent(ctx context.Context, withdraw_id int) (bool, error) {
	const method = "DeleteWithdrawPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("withdraw_id", withdraw_id))

	defer func() {
		end(status, "grpc")
	}()

	s.logger.Debug("Deleting withdraw permanently", zap.Int("withdraw_id", withdraw_id))

	_, err := s.withdrawCommandRepository.DeleteWithdrawPermanent(ctx, withdraw_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			err,
			method,
			span,

			zap.Int("withdraw_id", withdraw_id),
		)
	}

	logSuccess("Successfully deleted withdraw permanently", zap.Int("withdraw_id", withdraw_id))

	return true, nil
}

func (s *withdrawCommandService) RestoreAllWithdraw(ctx context.Context) (bool, error) {
	const method = "RestoreAllWithdraw"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status, "grpc")
	}()

	s.logger.Debug("Restoring all withdraws")

	_, err := s.withdrawCommandRepository.RestoreAllWithdraw(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			sharedErrors.ErrFailed("restore all withdraws"),
			method,
			span,
		)
	}

	logSuccess("Successfully restored all withdraws")
	return true, nil
}

func (s *withdrawCommandService) DeleteAllWithdrawPermanent(ctx context.Context) (bool, error) {
	const method = "DeleteAllWithdrawPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status, "grpc")
	}()

	s.logger.Debug("Permanently deleting all withdraws")

	_, err := s.withdrawCommandRepository.DeleteAllWithdrawPermanent(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			sharedErrors.ErrFailed("delete all withdraws permanently"),
			method,
			span,
		)
	}

	logSuccess("Successfully deleted all withdraws permanently")
	return true, nil
}

func (s *withdrawCommandService) enqueueWithdrawEvents(ctx context.Context, withdraw *db.CreateWithdrawRow, updatedWithdraw *db.UpdateWithdrawStatusRow, card *carddb.GetUserEmailByCardNumberRow, request *requests.CreateWithdrawRequest, newTotalBalance int) {
	if s.outboxStore == nil {
		return
	}
	wid := strconv.Itoa(int(withdraw.WithdrawID))

	// Email enrichment is intentionally best-effort and asynchronous. Never
	// enqueue an email event with an empty recipient after money has moved.
	if card.Email != "" {
		htmlBody, err := email.GenerateEmailHTML(map[string]string{
			"Title": "Withdraw Successful", "Message": fmt.Sprintf("Your withdrawal of %d has been processed successfully.", request.WithdrawAmount),
			"Button": "View History", "Link": "https://sanedge.example.com/withdraw/history",
		})
		if err == nil {
			emailPayload, _ := json.Marshal(map[string]any{"email": card.Email, "subject": "Withdraw Successful - SanEdge", "body": htmlBody})
			if iErr := s.outboxStore.Insert(ctx, db.OutboxRecord{AggregateType: "withdraw", AggregateID: wid, EventType: "withdraw.created", Payload: emailPayload}); iErr != nil {
				s.logger.Error("outbox: failed to enqueue withdraw email event", zap.Error(iErr))
			}
		}
	}

	statsEvent := events.WithdrawEvent{
		WithdrawID: uint64(updatedWithdraw.WithdrawID),
		WithdrawNo: fmt.Sprintf("%x-%x-%x-%x-%x", updatedWithdraw.WithdrawNo.Bytes[0:4], updatedWithdraw.WithdrawNo.Bytes[4:6], updatedWithdraw.WithdrawNo.Bytes[6:8], updatedWithdraw.WithdrawNo.Bytes[8:10], updatedWithdraw.WithdrawNo.Bytes[10:16]),
		CardNumber: request.CardNumber, CardType: card.CardType,
		Amount: int64(request.WithdrawAmount), Status: "success", CreatedAt: time.Now(),
	}
	statsBytes, _ := json.Marshal(statsEvent)
	if iErr := s.outboxStore.Insert(ctx, db.OutboxRecord{AggregateType: "withdraw", AggregateID: wid, EventType: "withdraw.stats", Payload: statsBytes}); iErr != nil {
		s.logger.Error("outbox: failed to enqueue withdraw stats event", zap.Error(iErr))
	}

	saldoBytes, _ := json.Marshal(events.SaldoEvent{CardNumber: request.CardNumber, TotalBalance: int64(newTotalBalance), CreatedAt: time.Now()})
	if iErr := s.outboxStore.Insert(ctx, db.OutboxRecord{AggregateType: "withdraw", AggregateID: request.CardNumber, EventType: "withdraw.saldo", Payload: saldoBytes}); iErr != nil {
		s.logger.Error("outbox: failed to enqueue withdraw saldo event", zap.Error(iErr))
	}
}

func (s *withdrawCommandService) markWithdrawAsFailed(ctx context.Context, withdrawID int, method string, span trace.Span) {
	req := &requests.UpdateWithdrawStatus{
		WithdrawID: withdrawID,
		Status:     "failed",
	}
	// P0.4: detached context with fixed timeout - never the request context,
	// which may be cancelled by the client before the goroutine runs.
	async.RunWithTimeout(5*time.Second, func(ctx context.Context) {
		if _, err := s.withdrawCommandRepository.UpdateWithdrawStatus(ctx, req); err != nil {
			s.logger.Error("compensation: failed to mark withdraw as failed", zap.Error(err), zap.Int("withdraw_id", withdrawID), zap.String("method", method))
		}
	})
}
