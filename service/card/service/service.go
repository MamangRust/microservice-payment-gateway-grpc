package service

import (
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/adapter"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/kafka"
	"github.com/MamangRust/microservice-payment-gateway-grpc/pkg/logger"
	mencache "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/redis"
	"github.com/MamangRust/microservice-payment-gateway-grpc/service/card/repository"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/observability"
)

type Service interface {
	CardQueryService
	CardCommandService
}

type service struct {
	CardQueryService
	CardCommandService
}

type Deps struct {
	Cache        *cache.CacheStore
	Repositories *repository.Repositories
	UserAdapter  adapter.UserAdapter
	Logger       logger.LoggerInterface
	Kafka        *kafka.Kafka
}

func NewService(deps *Deps) Service {
	observability, _ := observability.NewObservability("card-server", deps.Logger)

	cache := mencache.NewMencache(deps.Cache)

	return &service{
		CardQueryService:     newCardQuery(deps, observability, cache),
		CardCommandService:   newCardCommand(deps, observability, cache),
	}
}

// newCardQuery initializes a new instance of the CardQueryService.
// It takes a pointer to Deps and a mapper for CardResponse.
// It returns a pointer to CardQueryService.
func newCardQuery(deps *Deps, observability observability.TraceLoggerObservability, cache mencache.Mencache) CardQueryService {
	return NewCardQueryService(&cardQueryServiceDeps{
		Cache:               cache,
		CardQueryRepository: deps.Repositories.CardQuery,
		Logger:              deps.Logger,
		Observability:       observability,
	})
}



// newCardCommand initializes a new instance of the CardCommandService.
// It takes a pointer to Deps and a mapper for CardResponse.
// It returns a pointer to CardCommandService.
func newCardCommand(deps *Deps, observability observability.TraceLoggerObservability, cache mencache.Mencache) CardCommandService {
	return NewCardCommandService(&cardCommandServiceDeps{
		Cache:                 cache,
		Kafka:                 deps.Kafka,
		UserAdapter:           deps.UserAdapter,
		CardCommandRepository: deps.Repositories.CardCommand,
		Logger:                deps.Logger,
		Observability:         observability,
	})
}
