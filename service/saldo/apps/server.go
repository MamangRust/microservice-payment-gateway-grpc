package apps

import (
	"context"
	"fmt"
	"time"

	pbcard "github.com/MamangRust/microservice-payment-gateway-grpc/pb/card"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/saldo"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/resilience"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/server"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/handler"
	saldokafka "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/redis"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/service"
	ledgerworker "github.com/MamangRust/microservice-payment-gateway-grpc/shared/ledger"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/outbox"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewServer(cfg *server.Config) (*server.GRPCServer, error) {
	srv, err := server.New(cfg)
	if err != nil {
		return nil, err
	}

	queries := db.New(srv.Pool)

	// gRPC Clients for cross-service communication
	connCard, err := grpc.NewClient(viper.GetString("GRPC_CARD_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Card service: %w", err)
	}

	cardClientQuery := pbcard.NewCardQueryServiceClient(connCard)
	cardClientCmd := pbcard.NewCardCommandServiceClient(connCard)
	cardGuard := resilience.NewDependencyGuard("card", 5, 30, 100, 3*time.Second, srv.Logger)
	cardAdapter := adapter.NewCardAdapter(cardClientQuery, cardClientCmd, adapter.WithDependencyGuard(cardGuard))

	repos := repository.NewRepositories(queries, cardAdapter)

	mykafka, err := kafka.NewKafka(srv.Logger, []string{viper.GetString("KAFKA_BROKERS")})
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	svc := service.NewService(&service.Deps{
		Cache:        srv.CacheStore,
		Logger:       srv.Logger,
		Repositories: repos,
		CardAdapter:  cardAdapter,
		Kafka:        mykafka,
		Outbox:       outbox.NewStore(queries.InsertOutbox),
	})

	kafkaHandler := saldokafka.NewSaldoKafkaHandler(svc, srv.Logger, context.Background())
	err = mykafka.StartConsumers([]string{"saldo-service-topic-create-saldo"}, "saldo-service-group", kafkaHandler)
	if err != nil {
		srv.Logger.Error("Failed to start kafka consumers", zap.Error(err))
	}

	// Phase 3: durable outbox publisher - saldo.changed events are sent to
	// stats-topic-saldo-events (stats-writer) and consumed by the cache
	// invalidator below.
	outboxPub := outbox.NewPublisher(srv.Pool, mykafka, srv.Logger)
	go outboxPub.Start(srv.Ctx)

	// P1.4: durable cache invalidation. The invalidator runs in its own consumer
	// group so it never competes with stats-writer for offsets, and it is safe
	// to replay (deleting a missing cache key is a no-op).
	saldoCache := mencache.NewMencache(srv.CacheStore)
	cacheInvalidator := saldokafka.NewSaldoCacheInvalidator(saldoCache, srv.Logger)
	if err := mykafka.StartConsumers([]string{"stats-topic-saldo-events"}, "saldo-cache-invalidator", cacheInvalidator); err != nil {
		srv.Logger.Error("Failed to start saldo cache invalidator consumer", zap.Error(err))
	}

	h := handler.NewHandler(svc)

	srv.RegisterServices = func(gs *grpc.Server) {
		pb.RegisterSaldoQueryServiceServer(gs, h)
		pb.RegisterSaldoCommandServiceServer(gs, h)
	}

	// Phase 4: start reconciliation only after all saldo dependencies and
	// consumers have initialized successfully. It follows srv.Ctx and exits
	// during graceful shutdown.
	reconciler := ledgerworker.NewReconciler(queries, srv.Logger, 5*time.Minute, 100)
	go reconciler.Start(srv.Ctx)

	return srv, nil
}
