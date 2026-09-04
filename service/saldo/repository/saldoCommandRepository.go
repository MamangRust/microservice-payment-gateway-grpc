package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	database "github.com/MamangRust/microservice-payment-gateway-grpc/pkg/database"
	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/saldo/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

type saldoCommandRepository struct {
	db *db.Queries
}

func NewSaldoCommandRepository(db *db.Queries) SaldoCommandRepository {
	return &saldoCommandRepository{
		db: db,
	}
}

func (r *saldoCommandRepository) CreateSaldo(ctx context.Context, request *requests.CreateSaldoRequest) (*db.CreateSaldoRow, error) {
	req := db.CreateSaldoParams{
		CardNumber:   request.CardNumber,
		TotalBalance: amountToInt64(request.TotalBalance),
	}
	res, err := r.db.CreateSaldo(ctx, req)

	if err != nil {
		// The SQL is an atomic idempotent upsert guarded by the partial unique
		// index idx_saldos_card_number_active (card_number WHERE deleted_at IS
		// NULL). If a unique violation still surfaces (e.g. constraint mismatch
		// with the index predicate), map it to 409 instead of a generic 500.
		if database.IsUniqueViolation(err) {
			return nil, sharedErrors.NewConflictError("saldo record already exists").WithInternal(err)
		}

		// READ COMMITTED snapshot race: the CTE's INSERT ... ON CONFLICT DO
		// NOTHING blocks on the unique index while the concurrent card-created
		// Kafka consumer commits, then skips (no inserted row), but the
		// UNION-ALL SELECT still uses the statement-start snapshot and cannot
		// see the just-committed row -> no rows in result set. Re-read the row
		// in a fresh statement to return the winner's record.
		if errors.Is(err, pgx.ErrNoRows) {
			existing, getErr := r.db.GetSaldoByCardNumber(ctx, request.CardNumber)
			if getErr != nil {
				return nil, sharedErrors.ErrFailed("create saldo record").WithInternal(getErr)
			}
			return &db.CreateSaldoRow{
				SaldoID:        existing.SaldoID,
				CardNumber:     existing.CardNumber,
				TotalBalance:   existing.TotalBalance,
				WithdrawAmount: existing.WithdrawAmount,
				WithdrawTime:   existing.WithdrawTime,
				CreatedAt:      existing.CreatedAt,
				UpdatedAt:      existing.UpdatedAt,
			}, nil
		}

		return nil, sharedErrors.ErrFailed("create saldo record").WithInternal(err)
	}

	return res, nil
}

func (r *saldoCommandRepository) CreateSaldoIfNotExists(ctx context.Context, request *requests.CreateSaldoRequest) error {
	return r.db.CreateSaldoIfNotExists(ctx, db.CreateSaldoIfNotExistsParams{
		CardNumber:   request.CardNumber,
		TotalBalance: amountToInt64(request.TotalBalance),
	})
}

func (r *saldoCommandRepository) UpdateSaldo(ctx context.Context, request *requests.UpdateSaldoRequest) (*db.UpdateSaldoRow, error) {
	if request.SaldoID == nil {
		return nil, sharedErrors.NewBadRequestError("saldo ID is required")
	}

	current, err := r.db.GetSaldoByID(ctx, int32(*request.SaldoID))
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "saldo record", "update saldo record")
	}
	// Changing the card association of an active saldo is not allowed; balance
	// changes must go through the immutable ledger adjustment path.
	if current.CardNumber != request.CardNumber {
		return nil, sharedErrors.NewConflictError("cannot change card number of an existing saldo").WithInternal(fmt.Errorf("card mismatch: %s != %s", current.CardNumber, request.CardNumber))
	}

	target := amountToInt64(request.TotalBalance)
	delta := target - current.TotalBalance
	if delta == 0 {
		return &db.UpdateSaldoRow{
			SaldoID: current.SaldoID, CardNumber: current.CardNumber, TotalBalance: current.TotalBalance,
			WithdrawAmount: current.WithdrawAmount, WithdrawTime: current.WithdrawTime,
			CreatedAt: current.CreatedAt, UpdatedAt: current.UpdatedAt,
		}, nil
	}

	// The operation identity includes the target balance so a retry with the
	// same target is a replay no-op, while a legitimate second update with a
	// different target produces a distinct immutable adjustment entry.
	adjusted, err := r.db.ApplyLedgerAdjustment(ctx, db.LedgerAdjustmentParams{
		CardNumber:  request.CardNumber,
		Delta:       delta,
		OperationID: fmt.Sprintf("saldo-update:%d:%d", *request.SaldoID, target),
		SourceType:  "saldo_update",
		SourceID:    fmt.Sprintf("%d", *request.SaldoID),
		Note:        "legacy absolute update routed through immutable ledger adjustment",
	})
	if err != nil {
		if errors.Is(err, db.ErrLedgerOperationConflict) {
			return nil, sharedErrors.NewConflictError("saldo update operation ID reused with different details").WithInternal(err)
		}
		return nil, sharedErrors.ErrFailed("update saldo record").WithInternal(err)
	}

	return &db.UpdateSaldoRow{
		SaldoID: adjusted.SaldoID, CardNumber: adjusted.CardNumber, TotalBalance: adjusted.TotalBalance,
		WithdrawAmount: current.WithdrawAmount, WithdrawTime: current.WithdrawTime,
		CreatedAt: current.CreatedAt, UpdatedAt: current.UpdatedAt,
	}, nil
}

func (r *saldoCommandRepository) UpdateSaldoBalance(ctx context.Context, request *requests.UpdateSaldoBalance) (*db.UpdateSaldoBalanceRow, error) {
	req := db.UpdateSaldoBalanceParams{
		CardNumber:   request.CardNumber,
		TotalBalance: amountToInt64(request.TotalBalance),
	}

	res, err := r.db.UpdateSaldoBalance(ctx, req)

	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "saldo record", "update saldo balance")
	}

	return res, nil
}

func (r *saldoCommandRepository) DebitSaldo(ctx context.Context, request *requests.DebitSaldoRequest) (*db.DebitSaldoRow, error) {
	amount := amountToInt64(request.Amount)

	operationID := request.OperationID
	if operationID == "" {
		operationID = uuid.NewString()
	}
	sourceType := request.SourceType
	if sourceType == "" {
		sourceType = "unknown"
	}
	res, err := r.db.DebitSaldoWithLedger(ctx, db.LedgerMutationParams{
		CardNumber:  request.CardNumber,
		Amount:      amount,
		OperationID: operationID,
		SourceType:  sourceType,
		SourceID:    request.SourceID,
	})
	if err != nil {
		if errors.Is(err, db.ErrLedgerOperationConflict) {
			return nil, sharedErrors.NewConflictError("ledger operation ID was reused with different mutation details").WithInternal(err)
		}
		// A no-rows result here means the atomic predicate total_balance >= $2
		// rejected the debit: the card exists (callers resolve it first) but the
		// balance is insufficient. Surface that as a 409 domain error instead of
		// a misleading 404 not-found. Any other error stays a generic failure.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sharedErrors.NewConflictError("insufficient balance").WithInternal(err)
		}
		return nil, sharedErrors.ErrFailed("debit saldo").WithInternal(err)
	}
	return res, nil
}

func (r *saldoCommandRepository) CreditSaldo(ctx context.Context, request *requests.CreditSaldoRequest) (*db.CreditSaldoRow, error) {
	amount := amountToInt64(request.Amount)

	operationID := request.OperationID
	if operationID == "" {
		operationID = uuid.NewString()
	}
	sourceType := request.SourceType
	if sourceType == "" {
		sourceType = "unknown"
	}
	res, err := r.db.CreditSaldoWithLedger(ctx, db.LedgerMutationParams{
		CardNumber:  request.CardNumber,
		Amount:      amount,
		OperationID: operationID,
		SourceType:  sourceType,
		SourceID:    request.SourceID,
	})
	if err != nil {
		if errors.Is(err, db.ErrLedgerOperationConflict) {
			return nil, sharedErrors.NewConflictError("ledger operation ID was reused with different mutation details").WithInternal(err)
		}
		// No-rows here means the card has no active saldo record (amount is
		// validated > 0 upstream), so keep the not-found mapping.
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "saldo record", "credit saldo")
	}
	return res, nil
}

func (r *saldoCommandRepository) ApplySaldoAdjustment(ctx context.Context, request *requests.ApplySaldoAdjustmentRequest) (*db.SaldoAdjustmentRow, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	res, err := r.db.ApplyLedgerAdjustment(ctx, db.LedgerAdjustmentParams{
		CardNumber:  request.CardNumber,
		Delta:       request.Delta,
		OperationID: request.OperationID,
		SourceType:  request.SourceType,
		SourceID:    request.SourceID,
		Note:        request.Note,
	})
	if err != nil {
		if errors.Is(err, db.ErrLedgerOperationConflict) {
			return nil, sharedErrors.NewConflictError("adjustment operation ID was reused with different mutation details").WithInternal(err)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sharedErrors.NewConflictError("saldo adjustment rejected").WithInternal(err)
		}
		return nil, sharedErrors.ErrFailed("apply saldo adjustment").WithInternal(err)
	}
	return res, nil
}

func (r *saldoCommandRepository) ResolveReconciliation(ctx context.Context, queueID int64, operationID, note string) error {
	if queueID <= 0 || operationID == "" {
		return sharedErrors.NewBadRequestError("queue ID and operation ID are required")
	}
	if err := r.db.ResolveReconciliation(ctx, queueID, operationID, note); err != nil {
		return sharedErrors.ErrNoRowsOrFailed(err, "reconciliation queue item", "resolve reconciliation")
	}
	return nil
}

func (r *saldoCommandRepository) UpdateSaldoWithdraw(ctx context.Context, request *requests.UpdateSaldoWithdraw) (*db.UpdateSaldoWithdrawRow, error) {
	withdrawAmount := optionalAmountToInt64(request.WithdrawAmount)

	if withdrawAmount == nil || request.WithdrawTime == nil {
		return nil, sharedErrors.NewBadRequestError("withdraw amount and time are required for ledger-backed withdrawal")
	}

	var withdrawTime pgtype.Timestamp
	withdrawTime = pgtype.Timestamp{
		Time:  *request.WithdrawTime,
		Valid: true,
	}

	operationID := "withdraw:" + request.CardNumber + ":" + request.WithdrawTime.UTC().Format("20060102150405.000000000") + ":" + fmt.Sprint(*withdrawAmount)
	debit, err := r.DebitSaldo(ctx, &requests.DebitSaldoRequest{
		CardNumber:  request.CardNumber,
		Amount:      int(*withdrawAmount),
		OperationID: operationID,
		SourceType:  "withdraw",
		SourceID:    request.CardNumber,
	})
	if err != nil {
		return nil, err
	}
	res, err := r.db.UpdateSaldoWithdrawMetadata(ctx, db.UpdateSaldoWithdrawMetadataParams{
		CardNumber:     request.CardNumber,
		WithdrawAmount: withdrawAmount,
		WithdrawTime:   withdrawTime,
	})
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "saldo record", "update saldo withdrawal metadata")
	}
	return &db.UpdateSaldoWithdrawRow{
		SaldoID: debit.SaldoID, CardNumber: debit.CardNumber, TotalBalance: debit.TotalBalance,
		WithdrawAmount: res.WithdrawAmount, WithdrawTime: res.WithdrawTime,
		CreatedAt: res.CreatedAt, UpdatedAt: res.UpdatedAt,
	}, nil
}

func (r *saldoCommandRepository) TrashedSaldo(ctx context.Context, saldo_id int) (*db.Saldo, error) {
	res, err := r.db.TrashSaldo(ctx, int32(saldo_id))
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "saldo record", "trash saldo record")
	}
	return res, nil
}

func (r *saldoCommandRepository) RestoreSaldo(ctx context.Context, saldo_id int) (*db.Saldo, error) {
	res, err := r.db.RestoreSaldo(ctx, int32(saldo_id))
	if err != nil {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "saldo record", "restore saldo record")
	}
	return res, nil
}

func (r *saldoCommandRepository) DeleteSaldoPermanent(ctx context.Context, saldo_id int) (bool, error) {
	err := r.db.DeleteSaldoPermanently(ctx, int32(saldo_id))
	if err != nil {
		return false, sharedErrors.ErrNoRowsOrFailed(err, "saldo record", "delete saldo record permanently")
	}
	return true, nil
}

func (r *saldoCommandRepository) RestoreAllSaldo(ctx context.Context) (bool, error) {
	err := r.db.RestoreAllSaldos(ctx)

	if err != nil {
		return false, sharedErrors.ErrFailed("restore all saldo records").WithInternal(err)
	}

	return true, nil
}

func (r *saldoCommandRepository) DeleteAllSaldoPermanent(ctx context.Context) (bool, error) {
	err := r.db.DeleteAllPermanentSaldos(ctx)

	if err != nil {
		return false, sharedErrors.ErrFailed("delete all saldo records permanently").WithInternal(err)
	}

	return true, nil
}
