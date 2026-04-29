package handler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/IBM/sarama"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/email/mailer"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/email/metrics"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	traceunic "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/trace_unic"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// emailHandler is a struct that implements the sarama.ConsumerGroupHandler interface
type emailHandler struct {
	ctx             context.Context
	trace           trace.Tracer
	logger          logger.LoggerInterface
	Mailer          mailer.MailerInterface
	requestCounter  *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

// NewEmailHandler returns a new instance of the emailHandler struct.
// It initializes the Prometheus metrics for counting and tracking request durations.
// It takes a context.Context, a logger.LoggerInterface, and a pointer to mailer.Mailer as input.
// The returned emailHandler instance is ready to be used for handling Kafka messages.
func NewEmailHandler(ctx context.Context, logger logger.LoggerInterface, mailer mailer.MailerInterface) *emailHandler {
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "email_service_requests_total",
			Help: "Total number of requests to the EmailService",
		},
		[]string{"method", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "email_service_request_duration_seconds",
			Help:    "Histogram of request durations for the EmailService",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status"},
	)

	prometheus.MustRegister(requestCounter, requestDuration)

	return &emailHandler{
		ctx:             ctx,
		logger:          logger,
		Mailer:          mailer,
		trace:           otel.Tracer("email-handler"),
		requestCounter:  requestCounter,
		requestDuration: requestDuration,
	}
}

func (h *emailHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *emailHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

// ConsumeClaim processes incoming Kafka messages from the specified consumer group claim.
//
// It unmarshals each message into a payload map and extracts the email address, subject, and body.
// It calls the Send method of the mailer.Mailer instance with the extracted data.
// If the Send method returns an error, it logs the error and records the error in the span.
// It also sets the status to "failed_send_email".
// If the Send method succeeds, it increments the EmailSent metric.
// Each message is marked as processed in the consumer group session.
//
// It also records the request duration and counts the total number of requests
// to the EmailService using Prometheus metrics.
func (h *emailHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	start := time.Now()
	status := "success"

	defer func() {
		h.recordMetrics("ConsumeClaim", status, start)
	}()

	_, span := h.trace.Start(h.ctx, "ConsumeClaim")
	defer span.End()

	for msg := range claim.Messages() {
		var payload map[string]interface{}
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			traceID := traceunic.GenerateTraceID("FAILED_UNMARSHAL_MESSAGE")

			h.logger.Error("Failed to unmarshal message", zap.Error(err))

			span.SetAttributes(attribute.String("trace.id", traceID))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to unmarshal message")
			status = "failed_unmarshal_message"

			continue
		}

		email := payload["email"].(string)
		subject := payload["subject"].(string)
		body := payload["body"].(string)

		err := h.Mailer.Send(email, subject, body)
		if err != nil {
			traceID := traceunic.GenerateTraceID("FAILED_SEND_EMAIL")

			h.logger.Error("Failed to send email", zap.Error(err))
			span.SetAttributes(attribute.String("trace.id", traceID))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to send email")
			status = "failed_send_email"

			metrics.EmailFailed.Inc()
		} else {
			metrics.EmailSent.Inc()
		}

		sess.MarkMessage(msg, "")
	}

	h.logger.Info("ConsumeClaim finished", zap.Int("messages", len(claim.Messages())))

	return nil
}

// recordMetrics records Prometheus metrics for the given method and status.
// It increments the request counter and observes the request duration
// for the given method and status, using the provided start time.
func (s *emailHandler) recordMetrics(method string, status string, start time.Time) {
	s.requestCounter.WithLabelValues(method, status).Inc()
	s.requestDuration.WithLabelValues(method, status).Observe(time.Since(start).Seconds())
}
