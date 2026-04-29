package apps

import (
	"fmt"
	"strings"

	pbai "github.com/MamangRust/microservice-payment-gateway-grpc/pb/ai_security"
	pbcard "github.com/MamangRust/microservice-payment-gateway-grpc/pb/card"
	pbmerchant "github.com/MamangRust/microservice-payment-gateway-grpc/pb/merchant"
	pbsaldo "github.com/MamangRust/microservice-payment-gateway-grpc/pb/saldo"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/transaction"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/server"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/handler"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/transaction/service"
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

	connMerchant, err := grpc.NewClient(viper.GetString("GRPC_MERCHANT_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Merchant service: %w", err)
	}

	connAI, err := grpc.NewClient(viper.GetString("GRPC_AI_SECURITY_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to AI Security service: %w", err)
	}

	saldoClientQuery := pbsaldo.NewSaldoQueryServiceClient(connSaldo)
	saldoClientCmd := pbsaldo.NewSaldoCommandServiceClient(connSaldo)
	cardClientQuery := pbcard.NewCardQueryServiceClient(connCard)
	cardClientCmd := pbcard.NewCardCommandServiceClient(connCard)
	merchantClientQuery := pbmerchant.NewMerchantQueryServiceClient(connMerchant)
	aiClient := pbai.NewAISecurityServiceClient(connAI)

	saldoAdapter := adapter.NewSaldoAdapter(saldoClientQuery, saldoClientCmd)
	cardAdapter := adapter.NewCardAdapter(cardClientQuery, cardClientCmd)
	merchantAdapter := adapter.NewMerchantAdapter(merchantClientQuery)

	repos := repository.NewRepositories(srv.DB, saldoAdapter, cardAdapter, merchantAdapter)
	kafkaBrokers := strings.Split(viper.GetString("KAFKA_BROKERS"), ",")
	myKafka := kafka.NewKafka(srv.Logger, kafkaBrokers)
	svc := service.NewService(&service.Deps{
		Kafka:            myKafka,
		Repositories:     repos,
		Logger:           srv.Logger,
		Cache:            srv.CacheStore,
		AISecurityClient: aiClient,
		MerchantAdapter:  merchantAdapter,
		CardAdapter:      cardAdapter,
		SaldoAdapter:     saldoAdapter,
	})
	h := handler.NewHandler(svc)

	srv.RegisterServices = func(gs *grpc.Server) {
		pb.RegisterTransactionQueryServiceServer(gs, h)
		pb.RegisterTransactionCommandServiceServer(gs, h)
	}

	return srv, nil
}
