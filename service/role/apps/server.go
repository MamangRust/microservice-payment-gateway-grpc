package apps

import (
	"strings"
	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb/role"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/server"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/role/handler"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/role/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/role/service"
	"github.com/spf13/viper"
	"context"
	myhandlerkafka "github.com/MamangRust/microservice-payment-gateway-grpc/service/role/kafka"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func NewServer(cfg *server.Config) (*server.GRPCServer, error) {
	srv, err := server.New(cfg)
	if err != nil {
		return nil, err
	}

	repos := repository.NewRepositories(srv.DB)
	kafkaBrokers := strings.Split(viper.GetString("KAFKA_BROKERS"), ",")
	mykafka := kafka.NewKafka(srv.Logger, kafkaBrokers)

	svc := service.NewService(&service.Deps{
		Cache:        srv.CacheStore,
		Logger:       srv.Logger,
		Repositories: repos,
	})

	kafkaHandler := myhandlerkafka.NewRoleKafkaHandler(svc.RoleQuery, mykafka, srv.Logger, context.Background())
	err = mykafka.StartConsumers([]string{"request-role"}, "role-service-group", kafkaHandler)
	if err != nil {
		srv.Logger.Error("Failed to start kafka consumers", zap.Error(err))
	}

	h := handler.NewHandler(svc)

	srv.RegisterServices = func(gs *grpc.Server) {
		pb.RegisterRoleQueryServiceServer(gs, h.RoleQuery)
		pb.RegisterRoleCommandServiceServer(gs, h.RoleCommand)
	}

	return srv, nil
}
