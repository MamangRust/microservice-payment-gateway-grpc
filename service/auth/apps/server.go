package apps

import (
	"fmt"
	"strings"
	"time"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/auth/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/auth/handler"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/auth/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/auth/service"

	pb "github.com/MamangRust/microservice-payment-gateway-grpc/pb"
	pb_role "github.com/MamangRust/microservice-payment-gateway-grpc/pb/role"
	pbuserrole "github.com/MamangRust/microservice-payment-gateway-grpc/pb/user_role"
	pb_user "github.com/MamangRust/microservice-payment-gateway-grpc/pb/user"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/auth"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/hash"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/resilience"
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

	queries := db.New(srv.Pool)

	tokenManager, err := auth.NewManager(viper.GetString("SECRET_KEY"))
	if err != nil {
		return nil, fmt.Errorf("failed to create token manager: %w", err)
	}

	userConn, err := grpc.NewClient(viper.GetString("GRPC_USER_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to User service: %w", err)
	}

	roleConn, err := grpc.NewClient(viper.GetString("GRPC_ROLE_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Role service: %w", err)
	}

	userQueryClient := pb_user.NewUserQueryServiceClient(userConn)
	userCommandClient := pb_user.NewUserCommandServiceClient(userConn)
	roleQueryClient := pb_role.NewRoleQueryServiceClient(roleConn)
	roleCommandClient := pb_role.NewRoleCommandServiceClient(roleConn)
	userRoleClient := pbuserrole.NewUserRoleServiceClient(roleConn)

	userGuard := resilience.NewDependencyGuard("user", 5, 30, 100, 3*time.Second, srv.Logger)
	roleGuard := resilience.NewDependencyGuard("role", 5, 30, 100, 3*time.Second, srv.Logger)
	userRoleGuard := resilience.NewDependencyGuard("user_role", 5, 30, 100, 3*time.Second, srv.Logger)

	userAdapter := adapter.NewAuthUserAdapter(userQueryClient, userCommandClient, adapter.WithDependencyGuard(userGuard))
	roleAdapter := adapter.NewRoleAdapter(roleQueryClient, roleCommandClient, adapter.WithDependencyGuard(roleGuard))
	userRoleAdapter := adapter.NewUserRoleAdapter(userRoleClient, adapter.WithDependencyGuard(userRoleGuard))

	hasher := hash.NewHashingPassword()
	repositories := repository.NewRepositories(
		queries,
		userQueryClient,
		userCommandClient,
		roleQueryClient,
		roleCommandClient,
		userRoleClient,
		repository.GuardOptions{
			User:     []adapter.GuardOption{adapter.WithDependencyGuard(userGuard)},
			Role:     []adapter.GuardOption{adapter.WithDependencyGuard(roleGuard)},
			UserRole: []adapter.GuardOption{adapter.WithDependencyGuard(userRoleGuard)},
		},
	)

	kafkaBrokers := strings.Split(viper.GetString("KAFKA_BROKERS"), ",")
	myKafka, err := kafka.NewKafka(srv.Logger, kafkaBrokers)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	services := service.NewService(&service.Deps{
		Cache:           srv.CacheStore,
		Repositories:    repositories,
		Token:           tokenManager,
		Hash:            hasher,
		Logger:          srv.Logger,
		Kafka:           myKafka,
		UserAdapter:     userAdapter,
		RoleAdapter:     roleAdapter,
		UserRoleAdapter: userRoleAdapter,
	})

	handlers := handler.NewHandler(&handler.Deps{Service: services, Logger: srv.Logger})

	srv.RegisterServices = func(gs *grpc.Server) {
		pb.RegisterAuthServiceServer(gs, handlers.Auth)
	}

	return srv, nil
}
