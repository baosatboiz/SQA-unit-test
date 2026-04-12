// Package business — ConfirmPayment unit tests (PAY-TC-019 → PAY-TC-023).
//
// ConfirmPayment is called by a staff member to mark a payment as completed
// through an out-of-band channel (e.g., cash at the counter). The rules are:
//
//   1. If the paymentId does not exist, return a wrapped "payment not found" error.
//   2. If the payment is already COMPLETED, return nil (idempotent — the staff
//      clicked twice).
//   3. If the payment status is neither PENDING nor COMPLETED (e.g., FAILED),
//      reject the confirm with a "cannot confirm" error.
//   4. Happy path — update status → COMPLETED and payment_method = the given
//      method, then emit a PAYMENT_COMPLETED outbox event.

package business

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"payment-service/internal/module/payment/entity"
	repository "payment-service/internal/module/payment/repository/postgres"
)

// -----------------------------------------------------------------------------
// TC-019 — payment not found for the given paymentId  →  error
// -----------------------------------------------------------------------------

// If GetById cannot find a row, the error must be wrapped with the
// "payment not found" prefix so callers can match on it.
func TestConfirmPayment_TC019_ErrorWhenPaymentNotFound(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	mockOutbox := new(MockOutboxClient)
	biz := &paymentBiz{
		db:                tx,
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: new(MockBlockchainService),
	}

	err := biz.ConfirmPayment(ctx, "pay-tc019-missing", entity.PaymentMethodCash)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "payment not found")
	mockOutbox.AssertNotCalled(t, "CreateOutboxEvent", mock.Anything, mock.Anything, mock.Anything)
}

// -----------------------------------------------------------------------------
// TC-020 — payment already COMPLETED  →  nil (idempotent)
// -----------------------------------------------------------------------------

// Staff double-click protection: if the payment is already COMPLETED, return
// nil without updating anything and without emitting a new event.
func TestConfirmPayment_TC020_IdempotentWhenAlreadyCompleted(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	const paymentID = "pay-tc020"
	_, err := tx.NewInsert().Model(&entity.Payment{
		Id:          paymentID,
		BookingId:   "booking-tc020",
		Amount:      10000,
		PaymentDate: time.Now(),
		Status:      entity.PaymentStatusCompleted, // already done
		CreatedAt:   time.Now(),
	}).Exec(ctx)
	require.NoError(t, err)

	mockOutbox := new(MockOutboxClient)
	biz := &paymentBiz{
		db:                tx,
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: new(MockBlockchainService),
	}

	err = biz.ConfirmPayment(ctx, paymentID, entity.PaymentMethodCash)

	assert.NoError(t, err, "already-completed payment must return nil (no-op)")
	mockOutbox.AssertNotCalled(t, "CreateOutboxEvent", mock.Anything, mock.Anything, mock.Anything)
}

// -----------------------------------------------------------------------------
// TC-021 — payment status is FAILED  →  error "cannot confirm"
// -----------------------------------------------------------------------------

// ConfirmPayment must reject a payment whose status is FAILED (or anything
// other than PENDING/COMPLETED). The error must clearly mention "cannot confirm".
func TestConfirmPayment_TC021_ErrorWhenStatusIsFailed(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	const paymentID = "pay-tc021"
	_, err := tx.NewInsert().Model(&entity.Payment{
		Id:          paymentID,
		BookingId:   "booking-tc021",
		Amount:      12000,
		PaymentDate: time.Now(),
		Status:      entity.PaymentStatusFailed, // neither PENDING nor COMPLETED
		CreatedAt:   time.Now(),
	}).Exec(ctx)
	require.NoError(t, err)

	mockOutbox := new(MockOutboxClient)
	biz := &paymentBiz{
		db:                tx,
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: new(MockBlockchainService),
	}

	err = biz.ConfirmPayment(ctx, paymentID, entity.PaymentMethodCash)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot confirm",
		"error must mention 'cannot confirm' sentinel")
	mockOutbox.AssertNotCalled(t, "CreateOutboxEvent", mock.Anything, mock.Anything, mock.Anything)
}

// -----------------------------------------------------------------------------
// TC-022 — Happy path: status→COMPLETED, payment_method→CASH, verify in DB
// -----------------------------------------------------------------------------

// On the happy path for a PENDING payment, ConfirmPayment must update both
// status and payment_method. We verify with a live SELECT (CheckDB).
func TestConfirmPayment_TC022_UpdatesStatusAndMethodOnSuccess(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	const paymentID = "pay-tc022"
	_, err := tx.NewInsert().Model(&entity.Payment{
		Id:          paymentID,
		BookingId:   "booking-tc022",
		Amount:      150000,
		PaymentDate: time.Now(),
		Status:      entity.PaymentStatusPending,
		CreatedAt:   time.Now(),
	}).Exec(ctx)
	require.NoError(t, err)

	mockOutbox := new(MockOutboxClient)
	mockOutbox.On("CreateOutboxEvent",
		mock.Anything, mock.Anything, mock.Anything).Return(nil)

	biz := &paymentBiz{
		db:                tx,
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: new(MockBlockchainService),
	}

	err = biz.ConfirmPayment(ctx, paymentID, entity.PaymentMethodCash)
	require.NoError(t, err)

	// ---- CheckDB ----
	var updated entity.Payment
	err = tx.NewSelect().Model(&updated).Where("id = ?", paymentID).Scan(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.PaymentStatusCompleted, updated.Status)
	assert.Equal(t, entity.PaymentMethodCash, updated.PaymentMethod)
}

// -----------------------------------------------------------------------------
// TC-023 — Outbox event PAYMENT_COMPLETED fired after confirm
// -----------------------------------------------------------------------------

// After a successful confirm, the outbox event must be published so downstream
// services (booking, notification, analytics) can react to the state change.
func TestConfirmPayment_TC023_EmitsOutboxPaymentCompletedEvent(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	const paymentID = "pay-tc023"
	_, err := tx.NewInsert().Model(&entity.Payment{
		Id:          paymentID,
		BookingId:   "booking-tc023",
		Amount:      180000,
		PaymentDate: time.Now(),
		Status:      entity.PaymentStatusPending,
		CreatedAt:   time.Now(),
	}).Exec(ctx)
	require.NoError(t, err)

	mockOutbox := new(MockOutboxClient)
	mockOutbox.On("CreateOutboxEvent",
		mock.Anything,
		string(entity.EventTypePaymentCompleted),
		mock.Anything).Return(nil)

	biz := &paymentBiz{
		db:                tx,
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: new(MockBlockchainService),
	}

	err = biz.ConfirmPayment(ctx, paymentID, entity.PaymentMethodCash)
	require.NoError(t, err)

	mockOutbox.AssertCalled(t, "CreateOutboxEvent", mock.Anything,
		string(entity.EventTypePaymentCompleted), mock.Anything)
}
