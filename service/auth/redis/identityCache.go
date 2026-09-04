package mencache

import (
	"context"
	"fmt"
	"time"

	userdb "github.com/MamangRust/microservice-payment-gateway-grpc/service/user/database/schema"
	sharedcachehelpers "github.com/MamangRust/microservice-payment-gateway-grpc/shared/cache"
)

var (
	keyIdentityRefreshToken = "identity:refresh_token:%s"
	keyIdentityUserInfo     = "identity:user_info:%s"
)

type identityCache struct {
	store *sharedcachehelpers.CacheStore
}

func NewidentityCache(store *sharedcachehelpers.CacheStore) *identityCache {
	return &identityCache{store: store}
}

func (c *identityCache) SetRefreshToken(ctx context.Context, token string, expiration time.Duration) {
	key := keyIdentityRefreshToken
	key = fmt.Sprintf(key, token)

	sharedcachehelpers.SetToCache(ctx, c.store, key, &token, expiration)
}

func (c *identityCache) GetRefreshToken(ctx context.Context, token string) (string, bool) {
	key := keyIdentityRefreshToken
	key = fmt.Sprintf(key, token)

	result, found := sharedcachehelpers.GetFromCache[string](ctx, c.store, key)
	if !found || result == nil {
		return "", false
	}
	return *result, true
}

func (c *identityCache) DeleteRefreshToken(ctx context.Context, token string) {
	key := fmt.Sprintf(keyIdentityRefreshToken, token)
	sharedcachehelpers.DeleteFromCache(ctx, c.store, key)
}

func (c *identityCache) SetCachedUserInfo(ctx context.Context, user *userdb.GetUserByIDRow, expiration time.Duration) {
	if user == nil {
		return
	}

	key := fmt.Sprintf(keyIdentityUserInfo, user.UserID)

	sharedcachehelpers.SetToCache(ctx, c.store, key, user, expiration)
}

func (c *identityCache) GetCachedUserInfo(ctx context.Context, userId string) (*userdb.GetUserByIDRow, bool) {
	key := fmt.Sprintf(keyIdentityUserInfo, userId)

	return sharedcachehelpers.GetFromCache[userdb.GetUserByIDRow](ctx, c.store, key)
}

func (c *identityCache) DeleteCachedUserInfo(ctx context.Context, userId string) {
	key := fmt.Sprintf(keyIdentityUserInfo, userId)

	sharedcachehelpers.DeleteFromCache(ctx, c.store, key)
}
