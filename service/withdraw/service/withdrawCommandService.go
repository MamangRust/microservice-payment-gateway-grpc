package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/email"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errorhandler"
	withdraw_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/withdraw_errors/service"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/events"
	mencache "github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/redis"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pb/ai_security"
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

	Logger        logger.LoggerInterface
	Observability observability.TraceLoggerObservability
	AISecurityClient ai_security.AISecurityServiceClient
}

// withdrawCommandService handles command-side withdraw operations.
type withdrawCommandService struct {
	cache mencache.WithdrawCommandCache
	kafka *kafka.Kafka

	cardAdapter  adapter.CardAdapter
	saldoAdapter adapter.SaldoAdapter

	withdrawCommandRepository repository.WithdrawCommandRepository
	withdrawQueryRepository   repository.WithdrawQueryRepository

	logger        logger.LoggerInterface
	observability observability.TraceLoggerObservability
	aiSecurityClient ai_security.AISecurityServiceClient
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
		logger:                    deps.Logger,
		observability:             deps.Observability,
		aiSecurityClient:          deps.AISecurityClient,
	}
}

func (s *withdrawCommandService) Create(ctx context.Context, request *requests.CreateWithdrawRequest) (*db.UpdateWithdrawStatusRow, error) {
	const method = "CreateWithdraw"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status) }()

	card, err := s.cardAdapter.FindUserCardByCardNumber(ctx, request.CardNumber)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span, zap.String("card_number", request.CardNumber))
	}

	saldo, err := s.saldoAdapter.FindByCardNumber(ctx, request.CardNumber)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span, zap.String("card_number", request.CardNumber))
	}

	if int(saldo.TotalBalance) < request.WithdrawAmount {
		status = "error"
		err := errors.New("insufficient balance")
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span, zap.Float64("balance", float64(saldo.TotalBalance)), zap.Float64("amount", float64(request.WithdrawAmount)))
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

	newTotalBalance := int(saldo.TotalBalance) - request.WithdrawAmount
	updateData := &requests.UpdateSaldoWithdraw{
		CardNumber:     request.CardNumber,
		TotalBalance:   newTotalBalance,
		WithdrawAmount: &request.WithdrawAmount,
		WithdrawTime:   &request.WithdrawTime,
	}
	_, err = s.saldoAdapter.UpdateSaldoWithdraw(ctx, updateData)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span, zap.String("card_number", request.CardNumber))
	}

	withdrawRecord, err := s.withdrawCommandRepository.CreateWithdraw(ctx, request)
	if err != nil {
		status = "error"
		rollbackData := &requests.UpdateSaldoWithdraw{
			CardNumber:     request.CardNumber,
			TotalBalance:   int(saldo.TotalBalance),
			WithdrawAmount: &request.WithdrawAmount,
			WithdrawTime:   &request.WithdrawTime,
		}
		if _, rollbackErr := s.saldoAdapter.UpdateSaldoWithdraw(ctx, rollbackData); rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, rollbackErr, method, span, zap.String("rollback_for", "saldo"))
		}
		s.markWithdrawAsFailed(ctx, int(withdrawRecord.WithdrawID), method, span)
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span, zap.Int("withdraw_id", int(withdrawRecord.WithdrawID)))
	}

	updatedWithdraw, err := s.withdrawCommandRepository.UpdateWithdrawStatus(ctx, &requests.UpdateWithdrawStatus{
		WithdrawID: int(withdrawRecord.WithdrawID),
		Status:     "success",
	})
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateWithdrawStatusRow](s.logger, err, method, span, zap.Int("withdraw_id", int(withdrawRecord.WithdrawID)))
	}

	go func() {
		// Email Notification
		htmlBody := email.GenerateEmailHTML(map[string]string{
			"Title":   "Withdraw Successful",
			"Message": fmt.Sprintf("Your withdrawal of %d has been processed successfully.", request.WithdrawAmount),
			"Button":  "View History",
			"Link":    "https://sanedge.example.com/withdraw/history",
		})

		emailPayload := map[string]any{
			"email":   card.Email,
			"subject": "Withdraw Successful - SanEdge",
			"body":    htmlBody,
		}

		payloadBytes, _ := json.Marshal(emailPayload)
		if s.kafka != nil {
			_ = s.kafka.SendMessage("email-service-topic-withdraw-create", strconv.Itoa(int(updatedWithdraw.WithdrawID)), payloadBytes)
		}

		// Stats Event
		statsEvent := events.WithdrawEvent{
			WithdrawID: uint64(updatedWithdraw.WithdrawID),
			WithdrawNo: fmt.Sprintf("%x-%x-%x-%x-%x", updatedWithdraw.WithdrawNo.Bytes[0:4], updatedWithdraw.WithdrawNo.Bytes[4:6], updatedWithdraw.WithdrawNo.Bytes[6:8], updatedWithdraw.WithdrawNo.Bytes[8:10], updatedWithdraw.WithdrawNo.Bytes[10:16]),
			CardNumber: request.CardNumber,
			CardType:   card.CardType,
			Amount:     int64(request.WithdrawAmount),
			Status:     "success",
			CreatedAt:  time.Now(),
		}
		statsBytes, _ := json.Marshal(statsEvent)
		if s.kafka != nil {
			_ = s.kafka.SendMessage("stats-topic-withdraw-events", strconv.Itoa(int(updatedWithdraw.WithdrawID)), statsBytes)
		}

		// Saldo Event
		saldoEvent := events.SaldoEvent{
			CardNumber:   request.CardNumber,
			TotalBalance: int64(newTotalBalance),
			CreatedAt:    time.Now(),
		}
		saldoBytes, _ := json.Marshal(saldoEvent)
		if s.kafka != nil {
			_ = s.kafka.SendMessage("stats-topic-saldo-events", request.CardNumber, saldoBytes)
		}
	}()

	logSuccess("Successfully created withdraw", zap.Int("withdraw.id", int(updatedWithdraw.WithdrawID)))

	return updatedWithdraw, nil
}

func (s *withdrawCommandService) Update(ctx context.Context, request *requests.UpdateWithdrawRequest) (*db.UpdateWithdrawRow, error) {
	const method = "UpdateWithdraw"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status) }()

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
		return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, err, method, span, zap.String("card_number", request.CardNumber))
	}

	if int(saldo.TotalBalance) < request.WithdrawAmount {
		status = "error"
		err := errors.New("insufficient balance for update")
		s.markWithdrawAsFailed(ctx, *request.WithdrawID, method, span)
		return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, err, method, span, zap.Float64("balance", float64(saldo.TotalBalance)), zap.Float64("amount", float64(request.WithdrawAmount)))
	}

	newTotalBalance := int(saldo.TotalBalance) + int(oldWithdraw.WithdrawAmount) - request.WithdrawAmount
	updateSaldoData := &requests.UpdateSaldoBalance{
		CardNumber:   saldo.CardNumber,
		TotalBalance: newTotalBalance,
	}
	_, err = s.saldoAdapter.UpdateSaldoBalance(ctx, updateSaldoData)
	if err != nil {
		status = "error"
		s.markWithdrawAsFailed(ctx, *request.WithdrawID, method, span)
		return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, err, method, span, zap.String("card_number", request.CardNumber))
	}

	updatedWithdraw, err := s.withdrawCommandRepository.UpdateWithdraw(ctx, request)
	if err != nil {
		status = "error"
		rollbackData := &requests.UpdateSaldoBalance{
			CardNumber:   saldo.CardNumber,
			TotalBalance: int(saldo.TotalBalance),
		}
		if _, rollbackErr := s.saldoAdapter.UpdateSaldoBalance(ctx, rollbackData); rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, rollbackErr, method, span, zap.String("rollback_for", "saldo"))
		}
		s.markWithdrawAsFailed(ctx, *request.WithdrawID, method, span)
		return errorhandler.HandleError[*db.UpdateWithdrawRow](s.logger, err, method, span, zap.Int("withdraw_id", *request.WithdrawID))
	}

	if _, err := s.withdrawCommandRepository.UpdateWithdrawStatus(ctx, &requests.UpdateWithdrawStatus{
		WithdrawID: int(updatedWithdraw.WithdrawID),
		Status:     "success",
	}); err != nil {
		status = "error"
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
		end(status)
	}()

	s.logger.Debug("Trashing withdraw", zap.Int("withdraw_id", withdraw_id))

	res, err := s.withdrawCommandRepository.TrashedWithdraw(ctx, withdraw_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.Withdraw](
			s.logger,
			withdraw_errors.ErrFailedTrashedWithdraw,
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
		end(status)
	}()

	s.logger.Debug("Restoring withdraw", zap.Int("withdraw_id", withdraw_id))

	res, err := s.withdrawCommandRepository.RestoreWithdraw(ctx, withdraw_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.Withdraw](
			s.logger,
			withdraw_errors.ErrFailedRestoreWithdraw,
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
		end(status)
	}()

	s.logger.Debug("Deleting withdraw permanently", zap.Int("withdraw_id", withdraw_id))

	_, err := s.withdrawCommandRepository.DeleteWithdrawPermanent(ctx, withdraw_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			withdraw_errors.ErrFailedDeleteWithdrawPermanent,
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
		end(status)
	}()

	s.logger.Debug("Restoring all withdraws")

	_, err := s.withdrawCommandRepository.RestoreAllWithdraw(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			withdraw_errors.ErrFailedRestoreAllWithdraw,
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
		end(status)
	}()

	s.logger.Debug("Permanently deleting all withdraws")

	_, err := s.withdrawCommandRepository.DeleteAllWithdrawPermanent(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			withdraw_errors.ErrFailedDeleteAllWithdrawPermanent,
			method,
			span,
		)
	}

	logSuccess("Successfully deleted all withdraws permanently")
	return true, nil
}

func (s *withdrawCommandService) markWithdrawAsFailed(ctx context.Context, withdrawID int, method string, span trace.Span) {
	req := &requests.UpdateWithdrawStatus{
		WithdrawID: withdrawID,
		Status:     "failed",
	}
	go func() {
		if _, err := s.withdrawCommandRepository.UpdateWithdrawStatus(ctx, req); err != nil {
			s.logger.Error("compensation: failed to mark withdraw as failed", zap.Error(err), zap.Int("withdraw_id", withdrawID), zap.String("method", method))
		}
	}()
}
