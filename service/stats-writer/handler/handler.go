package handler

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/stats-writer/usecase"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/events"
	"go.uber.org/zap"
)

type StatsHandler struct {
	useCase usecase.UseCase
	log     logger.LoggerInterface
}

func NewStatsHandler(useCase usecase.UseCase, log logger.LoggerInterface) *StatsHandler {
	return &StatsHandler{
		useCase: useCase,
		log:     log,
	}
}

func (h *StatsHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *StatsHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *StatsHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		switch msg.Topic {
		case "payment.transaction.created", "stats-topic-transaction-events":
			var event events.TransactionEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				h.log.Error("Failed to unmarshal transaction event", zap.Error(err))
				continue
			}
			if err := h.useCase.SaveTransactionEvent(context.Background(), event); err != nil {
				h.log.Error("Failed to save transaction event", zap.Error(err))
				continue
			}
		case "stats-topic-topup-events":
			var event events.TopupEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				h.log.Error("Failed to unmarshal topup event", zap.Error(err))
				continue
			}
			if err := h.useCase.SaveTopupEvent(context.Background(), event); err != nil {
				h.log.Error("Failed to save topup event", zap.Error(err))
				continue
			}
		case "stats-topic-transfer-events":
			var event events.TransferEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				h.log.Error("Failed to unmarshal transfer event", zap.Error(err))
				continue
			}
			if err := h.useCase.SaveTransferEvent(context.Background(), event); err != nil {
				h.log.Error("Failed to save transfer event", zap.Error(err))
				continue
			}
		case "stats-topic-withdraw-events":
			var event events.WithdrawEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				h.log.Error("Failed to unmarshal withdraw event", zap.Error(err))
				continue
			}
			if err := h.useCase.SaveWithdrawEvent(context.Background(), event); err != nil {
				h.log.Error("Failed to save withdraw event", zap.Error(err))
				continue
			}
		case "stats-topic-saldo-events":
			var event events.SaldoEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				h.log.Error("Failed to unmarshal saldo event", zap.Error(err))
				continue
			}
			if err := h.useCase.SaveSaldoEvent(context.Background(), event); err != nil {
				h.log.Error("Failed to save saldo event", zap.Error(err))
				continue
			}
		case "stats-topic-merchant-events":
			var event events.MerchantEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				h.log.Error("Failed to unmarshal merchant event", zap.Error(err))
				continue
			}
			if err := h.useCase.SaveMerchantEvent(context.Background(), event); err != nil {
				h.log.Error("Failed to save merchant event", zap.Error(err))
				continue
			}
		case "stats-topic-card-events":
			var event events.CardEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				h.log.Error("Failed to unmarshal card event", zap.Error(err))
				continue
			}
			if err := h.useCase.SaveCardEvent(context.Background(), event); err != nil {
				h.log.Error("Failed to save card event", zap.Error(err))
				continue
			}
		}

		session.MarkMessage(msg, "")
	}
	return nil
}
