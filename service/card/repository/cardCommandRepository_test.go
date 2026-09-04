package repository

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	db "github.com/MamangRust/microservice-payment-gateway-grpc/service/card/database/schema"
	"github.com/MamangRust/microservice-payment-gateway-grpc/shared/domain/requests"
	sharedErrors "github.com/MamangRust/microservice-payment-gateway-grpc/shared/errors"
)

// fakeRow is a minimal pgx.Row whose Scan returns a scripted error. On the
// happy path (err == nil) it returns nil without populating dest, so the
// scanned row is zero-valued — the tests only assert on the returned error,
// not on row contents.
type fakeRow struct {
	err error
}

func (r *fakeRow) Scan(_ ...any) error { return r.err }

// fakeDBTX is a scripted db.DBTX used to drive the repository error paths.
// The card mutation queries (ToggleCardStatus, UpdateCreditLimit,
// RedeemRewardPoints) are UPDATE statements, while the GetCardByID fallback
// used by RedeemPoints is a SELECT — the fake dispatches on that difference.
type fakeDBTX struct {
	queryRowErr   error // error returned by the card mutation query
	getCardErr    error // error returned by the GetCardByID fallback query
	getCardExists bool  // when true, GetCardByID returns a row (nil error)
}

func (f *fakeDBTX) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (f *fakeDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (f *fakeDBTX) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if strings.Contains(sql, "SELECT") {
		if f.getCardExists {
			return &fakeRow{err: nil}
		}
		return &fakeRow{err: f.getCardErr}
	}
	return &fakeRow{err: f.queryRowErr}
}

func newFakeCardCommandRepository(f *fakeDBTX) CardCommandRepository {
	return NewCardCommandRepository(db.New(f))
}

// assertAppErrorCode verifies the returned error is an AppError with the
// expected HTTP code and message.
func assertAppErrorCode(t *testing.T, err error, wantCode int, wantMsg string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *sharedErrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *sharedErrors.AppError, got %T: %v", err, err)
	}
	if appErr.Code != wantCode {
		t.Fatalf("AppError.Code = %d, want %d (message: %s)", appErr.Code, wantCode, appErr.Message)
	}
	if appErr.Message != wantMsg {
		t.Fatalf("AppError.Message = %q, want %q", appErr.Message, wantMsg)
	}
}

func TestToggleCardStatusCardNotFound(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr: pgx.ErrNoRows,
	})

	_, err := repo.ToggleCardStatus(context.Background(), &requests.ToggleCardStatusRequest{CardID: 99})

	assertAppErrorCode(t, err, http.StatusNotFound, "card not found")
}
func TestToggleCardStatusInternalError(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr: errors.New("connection reset"),
	})

	_, err := repo.ToggleCardStatus(context.Background(), &requests.ToggleCardStatusRequest{CardID: 1})

	assertAppErrorCode(t, err, http.StatusInternalServerError, "Failed to update card status")
}

func TestToggleCardStatusSuccess(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{})

	if _, err := repo.ToggleCardStatus(context.Background(), &requests.ToggleCardStatusRequest{CardID: 1}); err != nil {
		t.Fatalf("ToggleCardStatus() unexpected error: %v", err)
	}
}

func TestUpdateCardCardNotFound(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr: pgx.ErrNoRows,
	})

	_, err := repo.UpdateCard(context.Background(), &requests.UpdateCardRequest{
		CardID:       99,
		UserID:       1,
		CardType:     "credit",
		ExpireDate:   time.Date(2028, 12, 1, 0, 0, 0, 0, time.UTC),
		CVV:          "123",
		CardProvider: "visa",
	})

	assertAppErrorCode(t, err, http.StatusNotFound, "card not found")
}

func TestUpdateCardInternalError(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr: errors.New("connection reset"),
	})

	_, err := repo.UpdateCard(context.Background(), &requests.UpdateCardRequest{
		CardID:       1,
		UserID:       1,
		CardType:     "credit",
		ExpireDate:   time.Date(2028, 12, 1, 0, 0, 0, 0, time.UTC),
		CVV:          "123",
		CardProvider: "visa",
	})

	assertAppErrorCode(t, err, http.StatusInternalServerError, "Failed to update card")
}

func TestUpdateCardSuccess(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{})

	if _, err := repo.UpdateCard(context.Background(), &requests.UpdateCardRequest{
		CardID:       1,
		UserID:       1,
		CardType:     "credit",
		ExpireDate:   time.Date(2028, 12, 1, 0, 0, 0, 0, time.UTC),
		CVV:          "123",
		CardProvider: "visa",
	}); err != nil {
		t.Fatalf("UpdateCard() unexpected error: %v", err)
	}
}

func TestTrashedCardCardNotFound(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr: pgx.ErrNoRows,
	})

	_, err := repo.TrashedCard(context.Background(), 99)

	assertAppErrorCode(t, err, http.StatusNotFound, "card not found")
}

func TestTrashedCardInternalError(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr: errors.New("connection reset"),
	})

	_, err := repo.TrashedCard(context.Background(), 1)

	assertAppErrorCode(t, err, http.StatusInternalServerError, "Failed to trash card")
}

func TestTrashedCardSuccess(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{})

	if _, err := repo.TrashedCard(context.Background(), 1); err != nil {
		t.Fatalf("TrashedCard() unexpected error: %v", err)
	}
}

func TestRestoreCardCardNotFound(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr: pgx.ErrNoRows,
	})

	_, err := repo.RestoreCard(context.Background(), 99)

	assertAppErrorCode(t, err, http.StatusNotFound, "card not found")
}

func TestRestoreCardInternalError(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr: errors.New("connection reset"),
	})

	_, err := repo.RestoreCard(context.Background(), 1)

	assertAppErrorCode(t, err, http.StatusInternalServerError, "Failed to restore card")
}

func TestRestoreCardSuccess(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{})

	if _, err := repo.RestoreCard(context.Background(), 1); err != nil {
		t.Fatalf("RestoreCard() unexpected error: %v", err)
	}
}

func TestUpdateCreditLimitCardNotFound(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr: pgx.ErrNoRows,
	})

	_, err := repo.UpdateCreditLimit(context.Background(), &requests.UpdateCreditLimitRequest{
		CardID:      99,
		CreditLimit: 5_000_000,
	})

	assertAppErrorCode(t, err, http.StatusNotFound, "card not found")
}

func TestUpdateCreditLimitInternalError(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr: errors.New("connection reset"),
	})

	_, err := repo.UpdateCreditLimit(context.Background(), &requests.UpdateCreditLimitRequest{
		CardID:      1,
		CreditLimit: 5_000_000,
	})

	assertAppErrorCode(t, err, http.StatusInternalServerError, "Failed to update credit limit")
}

func TestUpdateCreditLimitSuccess(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{})

	if _, err := repo.UpdateCreditLimit(context.Background(), &requests.UpdateCreditLimitRequest{
		CardID:      1,
		CreditLimit: 5_000_000,
	}); err != nil {
		t.Fatalf("UpdateCreditLimit() unexpected error: %v", err)
	}
}

func TestRedeemPointsCardNotFound(t *testing.T) {
	// The redeem query returns no rows AND the card itself does not exist,
	// so the fallback GetCardByID also reports ErrNoRows -> 404.
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr: pgx.ErrNoRows,
		getCardErr:  pgx.ErrNoRows,
	})

	_, err := repo.RedeemPoints(context.Background(), &requests.RedeemPointsRequest{
		CardID: 99,
		Points: 100,
	})

	assertAppErrorCode(t, err, http.StatusNotFound, "card not found")
}

func TestRedeemPointsInsufficientRewardPoints(t *testing.T) {
	// The redeem query returns no rows (guard reward_points >= points failed)
	// but the card exists, so the fallback GetCardByID succeeds -> 400.
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr:   pgx.ErrNoRows,
		getCardExists: true,
	})

	_, err := repo.RedeemPoints(context.Background(), &requests.RedeemPointsRequest{
		CardID: 1,
		Points: 100,
	})

	assertAppErrorCode(t, err, http.StatusBadRequest, "insufficient reward points")
}

func TestRedeemPointsFallbackInternalError(t *testing.T) {
	// The redeem query returns no rows, but the GetCardByID fallback itself
	// fails with a non-ErrNoRows error (e.g. connection reset) -> 500.
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr: pgx.ErrNoRows,
		getCardErr:  errors.New("connection reset"),
	})

	_, err := repo.RedeemPoints(context.Background(), &requests.RedeemPointsRequest{
		CardID: 1,
		Points: 100,
	})

	assertAppErrorCode(t, err, http.StatusInternalServerError, "Failed to redeem reward points")
}

func TestRedeemPointsInternalError(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{
		queryRowErr: errors.New("connection reset"),
	})

	_, err := repo.RedeemPoints(context.Background(), &requests.RedeemPointsRequest{
		CardID: 1,
		Points: 100,
	})

	assertAppErrorCode(t, err, http.StatusInternalServerError, "Failed to redeem reward points")
}

func TestRedeemPointsSuccess(t *testing.T) {
	repo := newFakeCardCommandRepository(&fakeDBTX{})

	if _, err := repo.RedeemPoints(context.Background(), &requests.RedeemPointsRequest{
		CardID: 1,
		Points: 100,
	}); err != nil {
		t.Fatalf("RedeemPoints() unexpected error: %v", err)
	}
}
