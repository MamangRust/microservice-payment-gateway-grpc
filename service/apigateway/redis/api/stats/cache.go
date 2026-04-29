package stats_cache

import (
	"context"

	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
)

type statsCache struct {
	store *cache.CacheStore
}

func NewStatsCache(store *cache.CacheStore) StatsCache {
	return &statsCache{store: store}
}

func (c *statsCache) GetCache(ctx context.Context, key string) (interface{}, bool) {
	result, found := cache.GetFromCache[interface{}](ctx, c.store, key)
	if !found || result == nil {
		return nil, false
	}
	return *result, true
}

func (c *statsCache) SetCache(ctx context.Context, key string, data interface{}) {
	if data == nil {
		return
	}
	cache.SetToCache(ctx, c.store, key, &data, ttlDefault)
}

func (c *statsCache) DeleteCache(ctx context.Context, key string) {
	cache.DeleteFromCache(ctx, c.store, key)
}
