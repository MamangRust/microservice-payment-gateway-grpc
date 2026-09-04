package mencache

import (
	"context"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
)

type SaldoQueryCache interface {
	GetCachedSaldos(ctx context.Context, req *requests.FindAllSaldos) ([]*db.GetSaldosRow, *int, bool)
	SetCachedSaldos(ctx context.Context, req *requests.FindAllSaldos, data []*db.GetSaldosRow, totalRecords *int)

	GetCachedSaldoById(ctx context.Context, saldo_id int) (*db.GetSaldoByIDRow, bool)
	SetCachedSaldoById(ctx context.Context, saldo_id int, data *db.GetSaldoByIDRow)

	GetCachedSaldoByCardNumber(ctx context.Context, card_number string) (*db.Saldo, bool)
	SetCachedSaldoByCardNumber(ctx context.Context, card_number string, data *db.Saldo)

	GetCachedSaldoByActive(ctx context.Context, req *requests.FindAllSaldos) ([]*db.GetActiveSaldosRow, *int, bool)
	SetCachedSaldoByActive(ctx context.Context, req *requests.FindAllSaldos, data []*db.GetActiveSaldosRow, totalRecords *int)

	GetCachedSaldoByTrashed(ctx context.Context, req *requests.FindAllSaldos) ([]*db.GetTrashedSaldosRow, *int, bool)
	SetCachedSaldoByTrashed(ctx context.Context, req *requests.FindAllSaldos, data []*db.GetTrashedSaldosRow, totalRecords *int)
}

type SaldoCommandCache interface {
	DeleteSaldoCache(ctx context.Context, saldo_id int)
	// DeleteSaldoCacheByCardNumber removes the card-number key, which is the
	// entry the financial flows read through FindByCardNumber. Mutations must
	// invalidate it alongside the by-id key to avoid stale balance reads up to
	// the TTL.
	DeleteSaldoCacheByCardNumber(ctx context.Context, card_number string)
}
