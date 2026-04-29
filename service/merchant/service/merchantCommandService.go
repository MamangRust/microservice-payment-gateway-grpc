package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	merchant_errors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors/merchant_errors/service"

	mencache "github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/redis"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/email"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/events"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/errorhandler"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"time"
)

// MerchantCommandServiceDeps contains the dependencies required to initialize a new instance
// of merchantCommandService.
type merchantCommandServiceDeps struct {
	// Kafka is the Kafka producer/consumer instance used to publish or consume merchant-related events.
	Kafka *kafka.Kafka

	// Cache provides caching functionality for merchant command-related data.
	Cache mencache.MerchantCommandCache

	// UserAdapter provides access to user data via gRPC.
	UserAdapter adapter.UserAdapter

	// MerchantQueryRepository is used to fetch merchant data in a read-only manner.
	MerchantQueryRepository repository.MerchantQueryRepository

	// MerchantCommandRepository is responsible for creating, updating, and deleting merchant records.
	MerchantCommandRepository repository.MerchantCommandRepository

	// Logger provides structured logging functionality for observability and debugging.
	Logger logger.LoggerInterface

	Observability observability.TraceLoggerObservability
}

// merchantCommandService provides an interface for interacting with the merchant command service,
// handling operations such as create, update, delete, and business logic for merchants.
type merchantCommandService struct {
	// kafka is the Kafka client used to publish merchant-related events
	// (e.g., merchant created, updated, or deleted) to Kafka topics.
	kafka *kafka.Kafka

	// mencache provides caching functionality for merchant data to reduce repeated database access,
	// typically backed by Redis or in-memory cache.
	cache mencache.MerchantCommandCache

	// userAdapter provides access to user-related data via gRPC.
	userAdapter adapter.UserAdapter

	// merchantQueryRepository is responsible for retrieving merchant data in a read-only manner,
	// often used to validate or enrich command operations.
	merchantQueryRepository repository.MerchantQueryRepository

	// merchantCommandRepository handles the actual persistence of merchant entities,
	// including creation, update, and soft/hard deletion in the database.
	merchantCommandRepository repository.MerchantCommandRepository

	// logger is the logging interface used to record structured logs
	// for observability and debugging during merchant command operations.
	logger logger.LoggerInterface

	observability observability.TraceLoggerObservability
}

// NewMerchantCommandService initializes a new instance of merchantCommandService with the provided parameters.
// It sets up Prometheus metrics for tracking request counts and durations and returns a configured
// merchantCommandService ready for handling merchant-related commands.
//
// Parameters:
// - params: A pointer to merchantCommandServiceDeps containing the necessary dependencies.
//
// Returns:
// - A pointer to an initialized merchantCommandService.
func NewMerchantCommandService(params *merchantCommandServiceDeps) MerchantCommandService {

	return &merchantCommandService{
		kafka:                     params.Kafka,
		cache:                     params.Cache,
		merchantCommandRepository: params.MerchantCommandRepository,
		userAdapter:               params.UserAdapter,
		merchantQueryRepository:   params.MerchantQueryRepository,
		logger:                    params.Logger,
		observability:             params.Observability,
	}
}

func (s *merchantCommandService) CreateMerchant(ctx context.Context, request *requests.CreateMerchantRequest) (*db.CreateMerchantRow, error) {
	const method = "CreateMerchant"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status)
	}()

	user, err := s.userAdapter.FindById(ctx, request.UserID)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.CreateMerchantRow](s.logger, err, method, span, zap.Int("user_id", request.UserID))
	}

	res, err := s.merchantCommandRepository.CreateMerchant(ctx, request)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.CreateMerchantRow](s.logger, err, method, span, zap.Int("user_id", request.UserID))
	}

	go func() {
		htmlBody := email.GenerateEmailHTML(map[string]string{
			"Title":   "Welcome to SanEdge Merchant Portal",
			"Message": "Your merchant account has been created successfully. To continue, please upload the required documents for verification. Once completed, our team will review and activate your account.",
			"Button":  "Upload Documents",
			"Link":    fmt.Sprintf("https://sanedge.example.com/merchant/%d/documents", user.UserID),
		})

		emailPayload := map[string]any{
			"email":   user.Email,
			"subject": "Initial Verification - SanEdge",
			"body":    htmlBody,
		}

		payloadBytes, err := json.Marshal(emailPayload)
		if err != nil {
			s.logger.Error("failed to marshal email payload for new merchant", zap.Error(err), zap.Int("merchant_id", int(res.MerchantID)))
			return
		}

		if s.kafka != nil {
			err = s.kafka.SendMessage("email-service-topic-merchant-created", strconv.Itoa(int(res.MerchantID)), payloadBytes)
			if err != nil {
				s.logger.Error("failed to send merchant creation email via kafka", zap.Error(err), zap.Int("merchant_id", int(res.MerchantID)))
			}

			// Stats Event
			statsEvent := events.MerchantEvent{
				MerchantID: uint64(res.MerchantID),
				UserID:     uint64(request.UserID),
				Name:       request.Name,
				Email:      user.Email,
				Status:     "inactive", // Default status for new merchant
				CreatedAt:  time.Now(),
			}
			statsPayloadByte, _ := json.Marshal(statsEvent)
			_ = s.kafka.SendMessage("stats-topic-merchant-events", strconv.Itoa(int(res.MerchantID)), statsPayloadByte)
		} else {
			s.logger.Warn("kafka is nil, skipping email notification", zap.Int("merchant_id", int(res.MerchantID)))
		}
	}()

	logSuccess("Successfully created merchant", zap.Int("merchant_id", int(res.MerchantID)))

	return res, nil
}

func (s *merchantCommandService) UpdateMerchant(ctx context.Context, request *requests.UpdateMerchantRequest) (*db.UpdateMerchantRow, error) {
	const method = "UpdateMerchant"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status)
	}()

	res, err := s.merchantCommandRepository.UpdateMerchant(ctx, request)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateMerchantRow](s.logger, err, method, span, zap.Int("merchant_id", *request.MerchantID))
	}

	s.cache.DeleteCachedMerchant(ctx, int(res.MerchantID))

	logSuccess("Successfully updated merchant", zap.Int("merchant_id", int(res.MerchantID)))

	return res, nil
}

func (s *merchantCommandService) UpdateMerchantStatus(ctx context.Context, request *requests.UpdateMerchantStatusRequest) (*db.UpdateMerchantStatusRow, error) {
	const method = "UpdateMerchantStatus"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status)
	}()

	merchant, err := s.merchantQueryRepository.FindByMerchantId(ctx, *request.MerchantID)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateMerchantStatusRow](s.logger, err, method, span, zap.Int("merchant_id", *request.MerchantID))
	}

	user, err := s.userAdapter.FindById(ctx, int(merchant.UserID))
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateMerchantStatusRow](s.logger, err, method, span, zap.Int("user_id", int(merchant.UserID)))
	}

	res, err := s.merchantCommandRepository.UpdateMerchantStatus(ctx, request)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.UpdateMerchantStatusRow](s.logger, err, method, span, zap.Int("merchant_id", *request.MerchantID))
	}

	go func() {
		statusReq := request.Status
		subject := ""
		message := ""
		link := fmt.Sprintf("https://sanedge.example.com/merchant/%d/dashboard", *request.MerchantID)

		switch statusReq {
		case "active":
			subject = "Your Merchant Account is Now Active"
			message = "Congratulations! Your merchant account has been verified and is now <b>active</b>. You can now fully access all features in the SanEdge Merchant Portal."
		case "inactive":
			subject = "Merchant Account Set to Inactive"
			message = "Your merchant account status has been set to <b>inactive</b>. Please contact support if you believe this is a mistake."
		case "rejected":
			subject = "Merchant Account Rejected"
			message = "We're sorry to inform you that your merchant account has been <b>rejected</b>. Please contact support or review your submissions."
		default:
			s.logger.Error("invalid merchant status provided for email notification", zap.String("status", statusReq), zap.Int("merchant_id", *request.MerchantID))
			return
		}

		htmlBody := email.GenerateEmailHTML(map[string]string{
			"Title":   subject,
			"Message": message,
			"Button":  "Go to Portal",
			"Link":    link,
		})

		emailPayload := map[string]any{
			"email":   user.Email,
			"subject": subject,
			"body":    htmlBody,
		}

		payloadBytes, err := json.Marshal(emailPayload)
		if err != nil {
			s.logger.Error("failed to marshal email payload for merchant status update", zap.Error(err), zap.Int("merchant_id", *request.MerchantID))
			return
		}

		if s.kafka != nil {
			err = s.kafka.SendMessage("email-service-topic-merchant-update-status", strconv.Itoa(*request.MerchantID), payloadBytes)
			if err != nil {
				s.logger.Error("failed to send merchant status update email via kafka", zap.Error(err), zap.Int("merchant_id", *request.MerchantID))
			}

			// Stats Event
			statsEvent := events.MerchantEvent{
				MerchantID: uint64(res.MerchantID),
				UserID:     uint64(merchant.UserID),
				Name:       merchant.Name,
				Email:      user.Email,
				Status:     res.Status,
				CreatedAt:  time.Now(),
			}
			statsPayloadByte, _ := json.Marshal(statsEvent)
			_ = s.kafka.SendMessage("stats-topic-merchant-events", strconv.Itoa(int(res.MerchantID)), statsPayloadByte)
		} else {
			s.logger.Warn("kafka is nil, skipping email notification", zap.Int("merchant_id", *request.MerchantID))
		}
	}()

	s.cache.DeleteCachedMerchant(ctx, int(res.MerchantID))

	logSuccess("Successfully updated merchant status", zap.Int("merchant_id", int(res.MerchantID)))

	return res, nil
}

func (s *merchantCommandService) TrashedMerchant(ctx context.Context, merchant_id int) (*db.Merchant, error) {
	const method = "TrashedMerchant"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("merchant_id", merchant_id))

	defer func() {
		end(status)
	}()

	s.logger.Debug("Trashing merchant", zap.Int("merchant_id", merchant_id))

	res, err := s.merchantCommandRepository.TrashedMerchant(ctx, merchant_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.Merchant](
			s.logger,
			merchant_errors.ErrFailedTrashMerchant,
			method,
			span,

			zap.Int("merchant_id", merchant_id),
		)
	}

	logSuccess("Successfully trashed merchant", zap.Int("merchant_id", merchant_id))

	return res, nil
}

func (s *merchantCommandService) RestoreMerchant(ctx context.Context, merchant_id int) (*db.Merchant, error) {
	const method = "RestoreMerchant"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("merchant_id", merchant_id))

	defer func() {
		end(status)
	}()

	s.logger.Debug("Restoring merchant", zap.Int("merchant_id", merchant_id))

	res, err := s.merchantCommandRepository.RestoreMerchant(ctx, merchant_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[*db.Merchant](
			s.logger,
			merchant_errors.ErrFailedRestoreMerchant,
			method,
			span,

			zap.Int("merchant_id", merchant_id),
		)
	}

	logSuccess("Successfully restored merchant", zap.Int("merchant_id", merchant_id))

	return res, nil
}

func (s *merchantCommandService) DeleteMerchantPermanent(ctx context.Context, merchant_id int) (bool, error) {
	const method = "DeleteMerchantPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method,
		attribute.Int("merchant_id", merchant_id))

	defer func() {
		end(status)
	}()

	s.logger.Debug("Deleting merchant permanently", zap.Int("merchant_id", merchant_id))

	_, err := s.merchantCommandRepository.DeleteMerchantPermanent(ctx, merchant_id)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			merchant_errors.ErrFailedDeleteMerchant,
			method,
			span,

			zap.Int("merchant_id", merchant_id),
		)
	}

	logSuccess("Successfully deleted merchant permanently", zap.Int("merchant_id", merchant_id))

	return true, nil
}

func (s *merchantCommandService) RestoreAllMerchant(ctx context.Context) (bool, error) {
	const method = "RestoreAllMerchant"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status)
	}()

	s.logger.Debug("Restoring all merchants")

	_, err := s.merchantCommandRepository.RestoreAllMerchant(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			merchant_errors.ErrFailedRestoreAllMerchants,
			method,
			span,
		)
	}

	logSuccess("Successfully restored all merchants")
	return true, nil
}

func (s *merchantCommandService) DeleteAllMerchantPermanent(ctx context.Context) (bool, error) {
	const method = "DeleteAllMerchantPermanent"

	ctx, span, end, status, logSuccess := s.observability.StartTracingAndLogging(ctx, method)

	defer func() {
		end(status)
	}()

	s.logger.Debug("Permanently deleting all merchants")

	_, err := s.merchantCommandRepository.DeleteAllMerchantPermanent(ctx)
	if err != nil {
		status = "error"
		return errorhandler.HandleError[bool](
			s.logger,
			merchant_errors.ErrFailedDeleteAllMerchants,
			method,
			span,
		)
	}

	logSuccess("Successfully deleted all merchants permanently")
	return true, nil
}
