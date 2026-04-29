package myhandlerkafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/IBM/sarama"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/service"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/response"
	"go.uber.org/zap"
)

// merchantKafkaHandler is a struct that implements the sarama.ConsumerGroupHandler interface
type merchantKafkaHandler struct {
	logger          logger.LoggerInterface
	merchantService service.MerchantQueryService
	kafka           *kafka.Kafka
}

// NewMerchantKafkaHandler creates a new Kafka consumer group handler for processing merchant API key validation responses.
//
// It takes a merchant query service, a Kafka producer, and a logger as parameters.
// The handler is used to process incoming Kafka messages from the merchant API key validation response topic.
// It implements the sarama.ConsumerGroupHandler interface to handle consumer group lifecycle events.
func NewMerchantKafkaHandler(merchantService service.MerchantQueryService, kafka *kafka.Kafka, logger logger.LoggerInterface) sarama.ConsumerGroupHandler {
	return &merchantKafkaHandler{
		merchantService: merchantService,
		kafka:           kafka,
		logger:          logger,
	}
}

// Setup is called at the beginning of a new Kafka consumer group session.
//
// It can be used to initialize resources or state before message consumption begins.
// In this implementation, it performs no setup and returns nil.
func (m *merchantKafkaHandler) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is called at the end of a Kafka consumer group session.
//
// It can be used to release resources allocated during Setup or message consumption.
// In this implementation, it performs no cleanup and returns nil.
func (m *merchantKafkaHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim processes incoming Kafka messages from the specified consumer group claim.
// It unmarshals each message into a payload map and retrieves the correlation ID.
// If a valid correlation ID is found, it checks if the API key is valid by calling
// the FindByApiKey method of the merchantService. If the API key is valid, it sends
// a valid response to the corresponding Kafka topic. Each message is marked as processed
// in the consumer group session.
func (m *merchantKafkaHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for msg := range claim.Messages() {
		var payload requests.MerchantRequestPayload
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			m.logger.Error("Failed to unmarshal merchant request", zap.Error(err))
			continue
		}

		resp := response.MerchantResponsePayload{
			CorrelationID: payload.CorrelationID,
			Valid:         false,
		}

		merchant, err := m.merchantService.FindByApiKey(ctx, payload.ApiKey)
		if err == nil && merchant != nil {
			resp.Valid = true
			resp.MerchantID = int64(merchant.MerchantID)
		}

		respBytes, _ := json.Marshal(resp)
		sendErr := m.kafka.SendMessage(payload.ReplyTopic, payload.CorrelationID, respBytes)
		if sendErr != nil {
			m.logger.Error("Failed to send Kafka response", zap.Error(sendErr))
		}

		session.MarkMessage(msg, "")
	}
	return nil
}
