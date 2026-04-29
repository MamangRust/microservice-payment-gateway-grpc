package mencache

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

type UserQueryCache interface {
	GetCachedUsersCache(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetUsersWithPaginationRow, *int, bool)
	SetCachedUsersCache(ctx context.Context, req *requests.FindAllUsers, data []*db.GetUsersWithPaginationRow, total *int)

	GetCachedUserActiveCache(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetActiveUsersWithPaginationRow, *int, bool)
	SetCachedUserActiveCache(ctx context.Context, req *requests.FindAllUsers, data []*db.GetActiveUsersWithPaginationRow, total *int)

	GetCachedUserTrashedCache(ctx context.Context, req *requests.FindAllUsers) ([]*db.GetTrashedUsersWithPaginationRow, *int, bool)
	SetCachedUserTrashedCache(ctx context.Context, req *requests.FindAllUsers, data []*db.GetTrashedUsersWithPaginationRow, total *int)

	GetCachedUserCache(ctx context.Context, id int) (*db.GetUserByIDRow, bool)
	SetCachedUserCache(ctx context.Context, data *db.GetUserByIDRow)
}

type UserCommandCache interface {
	DeleteUserCache(ctx context.Context, id int)
}
