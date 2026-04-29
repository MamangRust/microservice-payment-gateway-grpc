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
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/events"
	mencache "github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/redis"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pb/ai_security"
	"go.uber.org/zap"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	transfer_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/transfer_errors/service"
)

// transferCommandDeps defines dependencies for transferCommandService.
type transferCommandDeps struct {
	Kafka *kafka.Kafka
	Cache mencache.TransferCommandCache

	CardAdapter  adapter.CardAdapter
	SaldoAdapter adapter.SaldoAdapter

	TransferQueryRepository   repository.TransferQueryRepository
	TransferCommandRepository repository.TransferCommandRepository

	Logger        logger.LoggerInterface
	Observability observability.TraceLoggerObservability
	AISecurityClient ai_security.AISecurityServiceClient
}

// transferCommandService handles command-side transfer operations.
type transferCommandService struct {
	kafka *kafka.Kafka
	cache mencache.TransferCommandCache

	cardAdapter  adapter.CardAdapter
	saldoAdapter adapter.SaldoAdapter

	transferQueryRepository   repository.TransferQueryRepository
	transferCommandRepository repository.TransferCommandRepository

	logger        logger.LoggerInterface
	observability observability.TraceLoggerObservability
	aiSecurityClient ai_security.AISecurityServiceClient
}

func NewTransferCommandService(
	params *transferCommandDeps,
) TransferCommandService {
	return &transferCommandService{
		kafka:                     params.Kafka,
		cache:                     params.Cache,
		cardAdapter:               params.CardAdapter,
		saldoAdapter:              params.SaldoAdapter,
		transferQueryRepository:   params.TransferQueryRepository,
		transferCommandRepository: params.TransferCommandRepository,
		logger:                    params.Logger,
		observability:             params.Observability,
		aiSecurityClient:          params.AISecurityClient,
	}
}

func (s *transferCommandService) CreateTransaction(ctx context.Context, request *requests.CreateTransferRequest) (*db.UpdateTransferStatusRow, error) {
	const method = "CreateTransaction"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status) }()

	senderCard, err := s.cardAdapter.FindUserCardByCardNumber(ctx, request.TransferFrom)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateTransferStatusRow](s.logger, err, method, span, zap.String("from_card", request.TransferFrom))
	}

	_, err = s.cardAdapter.FindCardByCardNumber(ctx, request.TransferTo)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateTransferStatusRow](s.logger, err, method, span, zap.String("to_card", request.TransferTo))
	}

	senderSaldo, err := s.saldoAdapter.FindByCardNumber(ctx, request.TransferFrom)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateTransferStatusRow](s.logger, err, method, span, zap.String("from_card", request.TransferFrom))
	}

	receiverSaldo, err := s.saldoAdapter.FindByCardNumber(ctx, request.TransferTo)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateTransferStatusRow](s.logger, err, method, span, zap.String("to_card", request.TransferTo))
	}

	if int(senderSaldo.TotalBalance) < request.TransferAmount {
		status = "error"
		err := errors.New("insufficient balance for transfer")
		return errorhandler.HandleError[*db.UpdateTransferStatusRow](s.logger, err, method, span, zap.String("from_card", request.TransferFrom), zap.Float64("balance", float64(senderSaldo.TotalBalance)), zap.Float64("amount", float64(request.TransferAmount)))
	}

	// AI Security Check
	if s.aiSecurityClient != nil {
		secRes, err := s.aiSecurityClient.VerifySecurity(ctx, &ai_security.SecurityRequest{
			Domain:   ai_security.SecurityDomain_TRANSFER,
			EntityId: request.TransferFrom,
			Amount:   float64(request.TransferAmount),
			Metadata: map[string]string{
				"recipient_card": request.TransferTo,
			},
		})
		if err == nil && !secRes.IsSafe {
			status = "error"
			s.logger.Warn("Transfer blocked by AI Security", zap.String("reason", secRes.Reason))
			return nil, errors.New("security block: " + secRes.Reason)
		}
	}

	senderNewBalance := int(senderSaldo.TotalBalance) - request.TransferAmount

	_, err = s.saldoAdapter.UpdateSaldoBalance(ctx, &requests.UpdateSaldoBalance{
		CardNumber:   senderSaldo.CardNumber,
		TotalBalance: senderNewBalance,
	})
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateTransferStatusRow](s.logger, err, method, span, zap.String("from_card", request.TransferFrom))
	}

	receiverNewBalance := int(receiverSaldo.TotalBalance) + request.TransferAmount
	_, err = s.saldoAdapter.UpdateSaldoBalance(ctx, &requests.UpdateSaldoBalance{
		CardNumber:   receiverSaldo.CardNumber,
		TotalBalance: receiverNewBalance,
	})
	if err != nil {
		status = "error"
		if _, rollbackErr := s.saldoAdapter.UpdateSaldoBalance(ctx, &requests.UpdateSaldoBalance{
			CardNumber:   senderSaldo.CardNumber,
			TotalBalance: int(senderSaldo.TotalBalance),
		}); rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateTransferStatusRow](s.logger, rollbackErr, method, span, zap.String("rollback_for", "sender"))
		}
		return errorhandler.HandleError[*db.UpdateTransferStatusRow](s.logger, err, method, span, zap.String("failed_to_credit", "receiver"))
	}

	transfer, err := s.transferCommandRepository.CreateTransfer(ctx, request)
	if err != nil {
		status = "error"
		if _, rollbackErr := s.saldoAdapter.UpdateSaldoBalance(ctx, &requests.UpdateSaldoBalance{
			CardNumber:   senderSaldo.CardNumber,
			TotalBalance: int(senderSaldo.TotalBalance),
		}); rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateTransferStatusRow](s.logger, rollbackErr, method, span)
		}
		if _, rollbackErr := s.saldoAdapter.UpdateSaldoBalance(ctx, &requests.UpdateSaldoBalance{
			CardNumber:   receiverSaldo.CardNumber,
			TotalBalance: int(receiverSaldo.TotalBalance),
		}); rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateTransferStatusRow](s.logger, rollbackErr, method, span)
		}
		s.markTransferAsFailed(ctx, int(transfer.TransferID), method, span)
		return errorhandler.HandleError[*db.UpdateTransferStatusRow](s.logger, err, method, span)
	}

	updatedTransfer, err := s.transferCommandRepository.UpdateTransferStatus(ctx, &requests.UpdateTransferStatus{
		TransferID: int(transfer.TransferID),
		Status:     "success",
	})
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateTransferStatusRow](s.logger, err, method, span, zap.Int("transfer_id", int(transfer.TransferID)))
	}

	go func() {
		// Email Notification
		htmlBody := email.GenerateEmailHTML(map[string]string{
			"Title":   "Transfer Successful",
			"Message": fmt.Sprintf("Your Transfer of %d has been processed successfully.", request.TransferAmount),
			"Button":  "View History",
			"Link":    "https://sanedge.example.com/withdraw/history",
		})

		emailPayload := map[string]any{
			"email":   senderCard.Email,
			"subject": "Transfer Successful - SanEdge",
			"body":    htmlBody,
		}

		payloadBytes, _ := json.Marshal(emailPayload)
		if s.kafka != nil {
			_ = s.kafka.SendMessage("email-service-topic-transfer-create", strconv.Itoa(int(updatedTransfer.TransferID)), payloadBytes)
		}

		// Stats Event
		statsEvent := events.TransferEvent{
			TransferID:      uint64(updatedTransfer.TransferID),
			TransferNo:      fmt.Sprintf("%x-%x-%x-%x-%x", updatedTransfer.TransferNo.Bytes[0:4], updatedTransfer.TransferNo.Bytes[4:6], updatedTransfer.TransferNo.Bytes[6:8], updatedTransfer.TransferNo.Bytes[8:10], updatedTransfer.TransferNo.Bytes[10:16]),
			SourceCard:      request.TransferFrom,
			DestinationCard: request.TransferTo,
			Amount:          int64(request.TransferAmount),
			Status:          "success",
			CreatedAt:       time.Now(),
		}
		statsBytes, _ := json.Marshal(statsEvent)
		if s.kafka != nil {
			_ = s.kafka.SendMessage("stats-topic-transfer-events", strconv.Itoa(int(updatedTransfer.TransferID)), statsBytes)
		}

		// Saldo Event (Sender)
		senderSaldoEvent := events.SaldoEvent{
			CardNumber:   request.TransferFrom,
			TotalBalance: int64(senderNewBalance),
			CreatedAt:    time.Now(),
		}
		senderSaldoBytes, _ := json.Marshal(senderSaldoEvent)
		if s.kafka != nil {
			_ = s.kafka.SendMessage("stats-topic-saldo-events", request.TransferFrom, senderSaldoBytes)
		}

		// Saldo Event (Receiver)
		receiverSaldoEvent := events.SaldoEvent{
			CardNumber:   request.TransferTo,
			TotalBalance: int64(receiverNewBalance),
			CreatedAt:    time.Now(),
		}
		receiverSaldoBytes, _ := json.Marshal(receiverSaldoEvent)
		if s.kafka != nil {
			_ = s.kafka.SendMessage("stats-topic-saldo-events", request.TransferTo, receiverSaldoBytes)
		}
	}()

	logSuccess("Transfer created successfully", zap.Int("transfer_id", int(updatedTransfer.TransferID)), zap.String("from", request.TransferFrom), zap.String("to", request.TransferTo))

	return updatedTransfer, nil
}

// UpdateTransaction updates an existing transfer transaction.
//
// Parameters:
//   - ctx: The context for timeout and cancellation.
//   - request: The request containing updated transfer details.
//
// Returns:
//   - *response.TransferResponse: The updated transfer data.
//   - *response.ErrorResponse: Error details if operation fails.
func (s *transferCommandService) UpdateTransaction(ctx context.Context, request *requests.UpdateTransferRequest) (*db.UpdateTransferRow, error) {
	const method = "UpdateTransaction"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)
	defer func() { end(status) }()

	// 1. Dapatkan data transfer yang ada
	transfer, err := s.transferQueryRepository.FindById(ctx, *request.TransferID)
	if err != nil {
		status = "error"
		s.markTransferAsFailed(ctx, *request.TransferID, method, span)
		return errorhandler.HandleError[*db.UpdateTransferRow](s.logger, err, method, span, zap.Int("transfer_id", *request.TransferID))
	}

	amountDifference := request.TransferAmount - int(transfer.TransferAmount)

	senderSaldo, err := s.saldoAdapter.FindByCardNumber(ctx, transfer.TransferFrom)
	if err != nil {
		status = "error"
		s.markTransferAsFailed(ctx, *request.TransferID, method, span)
		return errorhandler.HandleError[*db.UpdateTransferRow](s.logger, err, method, span, zap.String("from_card", transfer.TransferFrom))
	}

	newSenderBalance := int(senderSaldo.TotalBalance) - amountDifference
	if newSenderBalance < 0 {
		status = "error"
		err := errors.New("insufficient balance for transfer update")
		s.markTransferAsFailed(ctx, *request.TransferID, method, span)
		return errorhandler.HandleError[*db.UpdateTransferRow](s.logger, err, method, span, zap.Float64("balance", float64(senderSaldo.TotalBalance)), zap.Float64("amount_diff", float64(amountDifference)))
	}

	_, err = s.saldoAdapter.UpdateSaldoBalance(ctx, &requests.UpdateSaldoBalance{
		CardNumber:   senderSaldo.CardNumber,
		TotalBalance: newSenderBalance,
	})
	if err != nil {
		status = "error"
		s.markTransferAsFailed(ctx, *request.TransferID, method, span)
		return errorhandler.HandleError[*db.UpdateTransferRow](s.logger, err, method, span, zap.String("from_card", transfer.TransferFrom))
	}

	receiverSaldo, err := s.saldoAdapter.FindByCardNumber(ctx, transfer.TransferTo)
	if err != nil {
		status = "error"
		if _, rollbackErr := s.saldoAdapter.UpdateSaldoBalance(ctx, &requests.UpdateSaldoBalance{
			CardNumber:   senderSaldo.CardNumber,
			TotalBalance: int(senderSaldo.TotalBalance),
		}); rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateTransferRow](s.logger, rollbackErr, method, span, zap.String("rollback_for", "sender"))
		}
		s.markTransferAsFailed(ctx, *request.TransferID, method, span)
		return errorhandler.HandleError[*db.UpdateTransferRow](s.logger, err, method, span, zap.String("to_card", transfer.TransferTo))
	}

	newReceiverBalance := int(receiverSaldo.TotalBalance) + amountDifference
	_, err = s.saldoAdapter.UpdateSaldoBalance(ctx, &requests.UpdateSaldoBalance{
		CardNumber:   receiverSaldo.CardNumber,
		TotalBalance: newReceiverBalance,
	})
	if err != nil {
		status = "error"
		if _, rollbackErr := s.saldoAdapter.UpdateSaldoBalance(ctx, &requests.UpdateSaldoBalance{
			CardNumber:   senderSaldo.CardNumber,
			TotalBalance: int(senderSaldo.TotalBalance),
		}); rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateTransferRow](s.logger, rollbackErr, method, span)
		}
		if _, rollbackErr := s.saldoAdapter.UpdateSaldoBalance(ctx, &requests.UpdateSaldoBalance{
			CardNumber:   receiverSaldo.CardNumber,
			TotalBalance: int(receiverSaldo.TotalBalance),
		}); rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateTransferRow](s.logger, rollbackErr, method, span)
		}
		s.markTransferAsFailed(ctx, *request.TransferID, method, span)
		return errorhandler.HandleError[*db.UpdateTransferRow](s.logger, err, method, span, zap.String("failed_to_credit", "receiver"))
	}

	updatedTransfer, err := s.transferCommandRepository.UpdateTransfer(ctx, request)
	if err != nil {
		status = "error"
		if _, rollbackErr := s.saldoAdapter.UpdateSaldoBalance(ctx, &requests.UpdateSaldoBalance{
			CardNumber:   senderSaldo.CardNumber,
			TotalBalance: int(senderSaldo.TotalBalance),
		}); rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateTransferRow](s.logger, rollbackErr, method, span)
		}
		if _, rollbackErr := s.saldoAdapter.UpdateSaldoBalance(ctx, &requests.UpdateSaldoBalance{
			CardNumber:   receiverSaldo.CardNumber,
			TotalBalance: int(receiverSaldo.TotalBalance),
		}); rollbackErr != nil {
			return errorhandler.HandleError[*db.UpdateTransferRow](s.logger, rollbackErr, method, span)
		}
		s.markTransferAsFailed(ctx, *request.TransferID, method, span)
		return errorhandler.HandleError[*db.UpdateTransferRow](s.logger, err, method, span)
	}

	if _, err := s.transferCommandRepository.UpdateTransferStatus(ctx, &requests.UpdateTransferStatus{
		TransferID: *request.TransferID,
		Status:     "success",
	}); err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateTransferRow](s.logger, err, method, span, zap.Int("transfer_id", *request.TransferID))
	}

	logSuccess("Successfully updated transfer", zap.Int("transfer.id", *request.TransferID))

	return updatedTransfer, nil
}

func (s *transferCommandService) TrashedTransfer(ctx context.Context, transfer_id int) (*db.Transfer, error) {
	const method = "TrashedTransfer"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("transfer_id", transfer_id))

	defer func() {
		end(status)
	}()

	s.logger.Debug("Starting trashed transfer process", zap.Int("transfer_id", transfer_id))

	res, err := s.transferCommandRepository.TrashedTransfer(ctx, transfer_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.Transfer](
			s.logger,
			transfer_errors.ErrFailedTrashedTransfer,
			method,
			span,

			zap.Int("transfer_id", transfer_id),
		)
	}

	logSuccess("Successfully trashed transfer", zap.Int("transfer_id", transfer_id))

	return res, nil
}

func (s *transferCommandService) RestoreTransfer(ctx context.Context, transfer_id int) (*db.Transfer, error) {
	const method = "RestoreTransfer"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("transfer_id", transfer_id))

	defer func() {
		end(status)
	}()

	s.logger.Debug("Starting restore transfer process", zap.Int("transfer_id", transfer_id))

	res, err := s.transferCommandRepository.RestoreTransfer(ctx, transfer_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.Transfer](
			s.logger,
			transfer_errors.ErrFailedRestoreTransfer,
			method,
			span,

			zap.Int("transfer_id", transfer_id),
		)
	}

	logSuccess("Successfully restored transfer", zap.Int("transfer_id", transfer_id))

	return res, nil
}

func (s *transferCommandService) DeleteTransferPermanent(ctx context.Context, transfer_id int) (bool, error) {
	const method = "DeleteTransferPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("transfer_id", transfer_id))

	defer func() {
		end(status)
	}()

	s.logger.Debug("Starting delete transfer permanent process", zap.Int("transfer_id", transfer_id))

	_, err := s.transferCommandRepository.DeleteTransferPermanent(ctx, transfer_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			transfer_errors.ErrFailedDeleteTransferPermanent,
			method,
			span,

			zap.Int("transfer_id", transfer_id),
		)
	}

	logSuccess("Successfully deleted transfer permanently", zap.Int("transfer_id", transfer_id))

	return true, nil
}

func (s *transferCommandService) RestoreAllTransfer(ctx context.Context) (bool, error) {
	const method = "RestoreAllTransfer"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status)
	}()

	s.logger.Debug("Restoring all transfers")

	_, err := s.transferCommandRepository.RestoreAllTransfer(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			transfer_errors.ErrFailedRestoreAllTransfers,
			method,
			span,
		)
	}

	logSuccess("Successfully restored all transfers")
	return true, nil
}

func (s *transferCommandService) DeleteAllTransferPermanent(ctx context.Context) (bool, error) {
	const method = "DeleteAllTransferPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status)
	}()

	s.logger.Debug("Permanently deleting all transfers")

	_, err := s.transferCommandRepository.DeleteAllTransferPermanent(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			transfer_errors.ErrFailedDeleteAllTransfersPermanent,
			method,
			span,
		)
	}

	logSuccess("Successfully deleted all transfers permanently")
	return true, nil
}

func (s *transferCommandService) markTransferAsFailed(ctx context.Context, transferID int, method string, span trace.Span) {
	req := &requests.UpdateTransferStatus{
		TransferID: transferID,
		Status:     "failed",
	}
	go func() {
		if _, err := s.transferCommandRepository.UpdateTransferStatus(ctx, req); err != nil {
			s.logger.Error("compensation: failed to mark transfer as failed", zap.Error(err), zap.Int("transfer_id", transferID), zap.String("method", method))
		}
	}()
}
