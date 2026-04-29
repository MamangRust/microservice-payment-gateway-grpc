package apps

import (
	"fmt"
	"strings"

	pbai "github.com/MamangRust/microservice-payment-gateway-grpc/pb/ai_security"
	pbcard "github.com/MamangRust/microservice-payment-gateway-grpc/pb/card"
	pbsaldo "github.com/MamangRust/microservice-payment-gateway-grpc/pb/saldo"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/withdraw"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/server"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/handler"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/withdraw/service"
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

	repos := repository.NewRepositories(srv.DB, cardAdapter, saldoAdapter)
	kafkaBrokers := strings.Split(viper.GetString("KAFKA_BROKERS"), ",")
	myKafka := kafka.NewKafka(srv.Logger, kafkaBrokers)

	svc := service.NewService(&service.Deps{
		Kafka:            myKafka,
		Repositories:     repos,
		Logger:           srv.Logger,
		Cache:            srv.CacheStore,
		AISecurityClient: aiClient,
		CardAdapter:      cardAdapter,
		SaldoAdapter:     saldoAdapter,
	})
	h := handler.NewHandler(svc)

	srv.RegisterServices = func(gs *grpc.Server) {
		pb.RegisterWithdrawQueryServiceServer(gs, h)
		pb.RegisterWithdrawCommandServiceServer(gs, h)
	}

	return srv, nil
}
