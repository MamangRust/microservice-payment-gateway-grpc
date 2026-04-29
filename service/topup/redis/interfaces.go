package mencache

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

type TopupQueryCache interface {
	GetCachedTopupsCache(ctx context.Context, req *requests.FindAllTopups) ([]*db.GetTopupsRow, *int, bool)
	SetCachedTopupsCache(ctx context.Context, req *requests.FindAllTopups, data []*db.GetTopupsRow, total *int)

	GetCacheTopupByCardCache(ctx context.Context, req *requests.FindAllTopupsByCardNumber) ([]*db.GetTopupsByCardNumberRow, *int, bool)
	SetCacheTopupByCardCache(ctx context.Context, req *requests.FindAllTopupsByCardNumber, data []*db.GetTopupsByCardNumberRow, total *int)

	GetCachedTopupActiveCache(ctx context.Context, req *requests.FindAllTopups) ([]*db.GetActiveTopupsRow, *int, bool)
	SetCachedTopupActiveCache(ctx context.Context, req *requests.FindAllTopups, data []*db.GetActiveTopupsRow, total *int)

	GetCachedTopupTrashedCache(ctx context.Context, req *requests.FindAllTopups) ([]*db.GetTrashedTopupsRow, *int, bool)
	SetCachedTopupTrashedCache(ctx context.Context, req *requests.FindAllTopups, data []*db.GetTrashedTopupsRow, total *int)

	GetCachedTopupCache(ctx context.Context, id int) (*db.GetTopupByIDRow, bool)
	SetCachedTopupCache(ctx context.Context, data *db.GetTopupByIDRow)
}

type TopupCommandCache interface {
	DeleteCachedTopupCache(ctx context.Context, id int)
}
