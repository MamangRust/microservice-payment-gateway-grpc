package apps

import (
	"fmt"
	"strings"

	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/handler"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/merchant/service"

	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/merchant"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pb/user"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/server"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewServer(cfg *server.Config) (*server.GRPCServer, error) {
	srv, err := server.New(cfg)
	if err != nil {
		return nil, err
	}

	// Establish GRPC connection to User service
	userConn, err := grpc.NewClient(viper.GetString("GRPC_USER_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to User service: %w", err)
	}

	userQueryClient := user.NewUserQueryServiceClient(userConn)
	userAdapter := adapter.NewUserAdapter(userQueryClient)

	kafkaBrokers := strings.Split(viper.GetString("KAFKA_BROKERS"), ",")
	mykafka := kafka.NewKafka(srv.Logger, kafkaBrokers)

	repos := repository.NewRepositories(srv.DB, userQueryClient)
	svc := service.NewService(&service.Deps{
		Cache:        srv.CacheStore,
		Logger:       srv.Logger,
		Repositories: repos,
		UserAdapter:  userAdapter,
		Kafka:        mykafka,
	})
	h := handler.NewHandler(svc)

	srv.RegisterServices = func(gs *grpc.Server) {
		pb.RegisterMerchantQueryServiceServer(gs, h)
		pb.RegisterMerchantCommandServiceServer(gs, h)
	}

	return srv, nil
}
