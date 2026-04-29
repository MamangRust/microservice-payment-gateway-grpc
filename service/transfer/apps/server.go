package apps

import (
	"fmt"
	pbcard "github.com/MamangRust/microservice-payment-gateway-grpc/pb/card"
	pbsaldo "github.com/MamangRust/microservice-payment-gateway-grpc/pb/saldo"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/transfer"
	pbai "github.com/MamangRust/microservice-payment-gateway-grpc/pb/ai_security"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/server"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/handler"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transfer/service"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewServer(cfg *server.Config) (*server.GRPCServer, error) {
	srv, err := server.New(cfg)
	if err != nil {
		return nil, err
	}

	// gRPC Clients for cross-service communication
	connSaldo, err := grpc.NewClient(viper.GetString("GRPC_SALDO_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Saldo service: %w", err)
	}

	connCard, err := grpc.NewClient(viper.GetString("GRPC_CARD_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Card service: %w", err)
	}

	connAI, err := grpc.NewClient(viper.GetString("GRPC_AI_SECURITY_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to AI Security service: %w", err)
	}

	saldoClientQuery := pbsaldo.NewSaldoQueryServiceClient(connSaldo)
	saldoClientCmd := pbsaldo.NewSaldoCommandServiceClient(connSaldo)
	cardClientQuery := pbcard.NewCardQueryServiceClient(connCard)
	cardClientCmd := pbcard.NewCardCommandServiceClient(connCard)
	aiClient := pbai.NewAISecurityServiceClient(connAI)

	saldoAdapter := adapter.NewSaldoAdapter(saldoClientQuery, saldoClientCmd)
	cardAdapter := adapter.NewCardAdapter(cardClientQuery, cardClientCmd)

	repos := repository.NewRepositories(srv.DB, saldoAdapter, cardAdapter)
	myKafka := kafka.NewKafka(srv.Logger, []string{viper.GetString("KAFKA_BROKERS")})

	svc := service.NewService(&service.Deps{
		Kafka:            myKafka,
		Cache:            srv.CacheStore,
		Logger:           srv.Logger,
		Repositories:     repos,
		AISecurityClient: aiClient,
		CardAdapter:      cardAdapter,
		SaldoAdapter:     saldoAdapter,
	})
	h := handler.NewHandler(svc)

	srv.RegisterServices = func(gs *grpc.Server) {
		pb.RegisterTransferQueryServiceServer(gs, h)
		pb.RegisterTransferCommandServiceServer(gs, h)
	}

	return srv, nil
}
