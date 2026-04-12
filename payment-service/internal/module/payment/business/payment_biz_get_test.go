// Package business — GetPaymentByBookingId unit tests (PAY-TC-006 → PAY-TC-007).
//
// GetPaymentByBookingId is a thin wrapper over repo.FindByBookingId, so the
// important behaviors to cover are:
//
//   1. The happy-path SELECT returns a fully-populated Payment row.
//   2. When no row matches the given bookingId, the repository's "not found"
//      error is surfaced unchanged.
//
// TC-006 uses the real database (the row is seeded inside the test tx and
// rolled back at defer). TC-007 also uses the real DB and relies on the fact
// that inside a freshly-opened tx no booking matches our unique sentinel.

package business

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment-service/internal/module/payment/entity"
	repository "payment-service/internal/module/payment/repository/postgres"
)

// TC-006 — Happy path: GetPaymentByBookingId returns the payment object that
// matches the given bookingId. We seed a row inside the test tx first, then
// call the business method and verify the returned object matches.
func TestGetPaymentByBookingId_TC006_ReturnsPaymentWhenFound(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	// Seed an existing payment so FindByBookingId has something to find.
	const bookingID = "booking-tc006"
	const paymentID = "pay-tc006-id"
	_, err := tx.NewInsert().Model(&entity.Payment{
		Id:          paymentID,
		BookingId:   bookingID,
		Amount:      88000,
		PaymentDate: time.Now(),
		Status:      entity.PaymentStatusPending,
		CreatedAt:   time.Now(),
	}).Exec(ctx)
	require.NoError(t, err)

	biz := &paymentBiz{
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      new(MockOutboxClient),
		blockchainService: new(MockBlockchainService),
	}

	// ---- Act ----
	got, err := biz.GetPaymentByBookingId(ctx, bookingID)

	// ---- Assert ----
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, paymentID, got.Id)
	assert.Equal(t, bookingID, got.BookingId)
	assert.Equal(t, 88000.0, got.Amount)
	assert.Equal(t, entity.PaymentStatusPending, got.Status)
}

// TC-007 — Not-found path: when the bookingId does not correspond to any
// payment row, GetPaymentByBookingId must return an error whose message
// signals "payment not found" (the underlying repository's not-found sentinel).
func TestGetPaymentByBookingId_TC007_ReturnsErrorWhenNotFound(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	biz := &paymentBiz{
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      new(MockOutboxClient),
		blockchainService: new(MockBlockchainService),
	}

	// Unique sentinel that is guaranteed not to exist inside this tx.
	got, err := biz.GetPaymentByBookingId(ctx, "booking-tc007-definitely-not-exist")

	assert.Nil(t, got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "payment not found",
		"error must mention 'payment not found' sentinel")
}
