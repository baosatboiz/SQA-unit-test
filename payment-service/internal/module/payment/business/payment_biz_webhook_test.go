// Package business — ProcessSePayWebhook unit tests (PAY-TC-008 → PAY-TC-014).
//
// ProcessSePayWebhook is the most complex method in paymentBiz: it consumes a
// raw bank webhook and must:
//
//   1. Extract the booking UUID from the free-form content/description fields
//      (delegated to extractUUIDNoHyphens — already covered by TC-024+).
//   2. Look up the matching payment row.
//   3. Guarantee idempotency against repeated webhooks for the same transaction.
//   4. Prevent double-payment when a payment is already COMPLETED with a
//      different transactionId.
//   5. Verify that the webhook amount matches the expected amount.
//   6. On success, UPDATE the payment row AND fire a PaymentCompleted event.
//
// Every test below uses a real transaction on the real PostgreSQL database
// (for CheckDB + Rollback) and mocks the outbox client (no live gRPC).

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
// TC-008 — UUID extraction fails  →  error "failed to extract"
// -----------------------------------------------------------------------------

// When the webhook's content/description do NOT contain any 32-char hex UUID,
// the extractor returns "" and ProcessSePayWebhook must fail with a clear
// error. No DB update and no outbox event must occur.
func TestProcessSePayWebhook_TC008_ErrorWhenUUIDExtractionFails(t *testing.T) {
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

	webhook := &entity.SePayWebhook{
		Id:             111,
		Content:        "chuyen khoan linh tinh",
		Description:    "khong co uuid o day",
		TransferAmount: 100000,
	}

	err := biz.ProcessSePayWebhook(ctx, webhook)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to extract booking UUID",
		"error must mention UUID extraction failure")
	mockOutbox.AssertNotCalled(t, "CreateOutboxEvent", mock.Anything, mock.Anything, mock.Anything)
}

// -----------------------------------------------------------------------------
// TC-009 — UUID extracted, but no matching payment row  →  "payment not found"
// -----------------------------------------------------------------------------

// The extracted UUID looks syntactically valid, but no row in `payments` has
// a matching booking_id. The business layer must surface a "payment not found"
// error so the worker can retry/log.
func TestProcessSePayWebhook_TC009_ErrorWhenNoPaymentMatchesExtractedUUID(t *testing.T) {
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

	// 32-char hex UUID that intentionally does not correspond to any seeded row.
	unknownUUID := "DEADBEEFDEADBEEFDEADBEEFDEADBEEF"
	webhook := &entity.SePayWebhook{
		Id:             222,
		Content:        "QH" + unknownUUID,
		TransferAmount: 100000,
	}

	err := biz.ProcessSePayWebhook(ctx, webhook)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "payment not found")
	mockOutbox.AssertNotCalled(t, "CreateOutboxEvent", mock.Anything, mock.Anything, mock.Anything)
}

// -----------------------------------------------------------------------------
// TC-010 — Same webhook replayed: transactionId already recorded  →  nil
// -----------------------------------------------------------------------------

// Idempotency: when the target payment already carries the same transactionId
// as the incoming webhook (e.g., SePay retried), ProcessSePayWebhook must
// return nil without touching the DB and without emitting any outbox event.
func TestProcessSePayWebhook_TC010_IdempotentWhenTransactionAlreadyProcessed(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	// Use a canonical UUID with hyphens as booking_id; the extractor strips
	// hyphens and compares the 32-char hex form.
	bookingUUID := "11111111-2222-3333-4444-555555555555"
	uuidNoHyphens := "11111111222233334444555555555555"
	txID := "222" // must equal fmt.Sprintf("%d", webhook.Id)

	_, err := tx.NewInsert().Model(&entity.Payment{
		Id:            "pay-tc010",
		BookingId:     bookingUUID,
		Amount:        50000,
		PaymentDate:   time.Now(),
		Status:        entity.PaymentStatusPending,
		TransactionId: &txID, // <-- already carries this txId
		CreatedAt:     time.Now(),
	}).Exec(ctx)
	require.NoError(t, err)

	mockOutbox := new(MockOutboxClient)
	biz := &paymentBiz{
		db:                tx,
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: new(MockBlockchainService),
	}

	webhook := &entity.SePayWebhook{
		Id:             222,
		Content:        "QH" + uuidNoHyphens,
		TransferAmount: 50000,
	}

	err = biz.ProcessSePayWebhook(ctx, webhook)

	assert.NoError(t, err, "replayed webhook must be accepted as nil (no-op)")
	mockOutbox.AssertNotCalled(t, "CreateOutboxEvent", mock.Anything, mock.Anything, mock.Anything)
}

// -----------------------------------------------------------------------------
// TC-011 — Payment already COMPLETED with a DIFFERENT txId  →  error
// -----------------------------------------------------------------------------

// Double-payment prevention: if the payment is already COMPLETED but with a
// different transactionId, the service must refuse the new webhook — accepting
// it would imply the customer paid twice.
func TestProcessSePayWebhook_TC011_ErrorWhenAlreadyCompletedWithDifferentTransaction(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	bookingUUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	uuidNoHyphens := "aaaaaaaabbbbccccddddeeeeeeeeeeee"
	previousTxID := "999" // completed earlier by a different webhook

	_, err := tx.NewInsert().Model(&entity.Payment{
		Id:            "pay-tc011",
		BookingId:     bookingUUID,
		Amount:        75000,
		PaymentDate:   time.Now(),
		Status:        entity.PaymentStatusCompleted,
		TransactionId: &previousTxID,
		CreatedAt:     time.Now(),
	}).Exec(ctx)
	require.NoError(t, err)

	mockOutbox := new(MockOutboxClient)
	biz := &paymentBiz{
		db:                tx,
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: new(MockBlockchainService),
	}

	webhook := &entity.SePayWebhook{
		Id:             111, // different from "999"
		Content:        "QH" + uuidNoHyphens,
		TransferAmount: 75000,
	}

	err = biz.ProcessSePayWebhook(ctx, webhook)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already completed",
		"error must mention 'already completed' sentinel")
	mockOutbox.AssertNotCalled(t, "CreateOutboxEvent", mock.Anything, mock.Anything, mock.Anything)
}

// -----------------------------------------------------------------------------
// TC-012 — webhook.TransferAmount != payment.Amount  →  "amount mismatch"
// -----------------------------------------------------------------------------

// The amount transferred by the customer must exactly match the stored amount.
// Any mismatch is a hard error — we do not silently accept partial or excess
// payments.
func TestProcessSePayWebhook_TC012_ErrorWhenAmountMismatch(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	bookingUUID := "12345678-1234-1234-1234-123456789012"
	uuidNoHyphens := "12345678123412341234123456789012"

	_, err := tx.NewInsert().Model(&entity.Payment{
		Id:          "pay-tc012",
		BookingId:   bookingUUID,
		Amount:      100000, // expected
		PaymentDate: time.Now(),
		Status:      entity.PaymentStatusPending,
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

	webhook := &entity.SePayWebhook{
		Id:             333,
		Content:        "QH" + uuidNoHyphens,
		TransferAmount: 99000, // WRONG
	}

	err = biz.ProcessSePayWebhook(ctx, webhook)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "amount mismatch")
	mockOutbox.AssertNotCalled(t, "CreateOutboxEvent", mock.Anything, mock.Anything, mock.Anything)
}

// -----------------------------------------------------------------------------
// TC-013 — Happy path: payment row UPDATE is persisted (CheckDB)
// -----------------------------------------------------------------------------

// On the happy path, ProcessSePayWebhook must update the payment row with:
//
//   - transaction_id = stringified webhook.Id
//   - payload        = JSON-serialized webhook
//   - status         = COMPLETED
//   - payment_method = BANK_TRANSFER
//
// We verify by performing a live SELECT on the test tx (CheckDB requirement).
func TestProcessSePayWebhook_TC013_UpdatesPaymentFieldsOnSuccess(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	bookingUUID := "abababab-cdcd-efef-0101-020202020202"
	uuidNoHyphens := "ababababcdcdefef0101020202020202"

	_, err := tx.NewInsert().Model(&entity.Payment{
		Id:          "pay-tc013",
		BookingId:   bookingUUID,
		Amount:      200000,
		PaymentDate: time.Now(),
		Status:      entity.PaymentStatusPending,
		CreatedAt:   time.Now(),
	}).Exec(ctx)
	require.NoError(t, err)

	mockOutbox := new(MockOutboxClient)
	// Accept any outbox call as success — the webhook test is about the DB write.
	mockOutbox.On("CreateOutboxEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	biz := &paymentBiz{
		db:                tx,
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: new(MockBlockchainService),
	}

	webhook := &entity.SePayWebhook{
		Id:             444,
		Content:        "QH" + uuidNoHyphens,
		TransferAmount: 200000,
		Gateway:        "MB Bank",
		ReferenceCode:  "REF-444",
	}

	err = biz.ProcessSePayWebhook(ctx, webhook)
	require.NoError(t, err)

	// ---- CheckDB: SELECT the updated row and verify every field ----
	var updated entity.Payment
	err = tx.NewSelect().Model(&updated).Where("id = ?", "pay-tc013").Scan(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.PaymentStatusCompleted, updated.Status)
	assert.Equal(t, entity.PaymentMethodBankTransfer, updated.PaymentMethod)
	require.NotNil(t, updated.TransactionId)
	assert.Equal(t, "444", *updated.TransactionId)
	require.NotNil(t, updated.Payload)
	assert.Contains(t, *updated.Payload, "MB Bank",
		"payload must contain the raw webhook JSON")
}

// -----------------------------------------------------------------------------
// TC-014 — Outbox event PaymentCompleted is fired on success
// -----------------------------------------------------------------------------

// The happy path must also publish an outbox event of type PAYMENT_COMPLETED
// so downstream services (booking, notification) can react. We assert the
// mock outbox was called with exactly that event type.
func TestProcessSePayWebhook_TC014_EmitsOutboxPaymentCompletedEvent(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	bookingUUID := "99999999-8888-7777-6666-555555555555"
	uuidNoHyphens := "99999999888877776666555555555555"

	_, err := tx.NewInsert().Model(&entity.Payment{
		Id:          "pay-tc014",
		BookingId:   bookingUUID,
		Amount:      300000,
		PaymentDate: time.Now(),
		Status:      entity.PaymentStatusPending,
		CreatedAt:   time.Now(),
	}).Exec(ctx)
	require.NoError(t, err)

	mockOutbox := new(MockOutboxClient)
	mockOutbox.On("CreateOutboxEvent", mock.Anything,
		string(entity.EventTypePaymentCompleted),
		mock.Anything).Return(nil)

	biz := &paymentBiz{
		db:                tx,
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: new(MockBlockchainService),
	}

	webhook := &entity.SePayWebhook{
		Id:             555,
		Content:        "QH" + uuidNoHyphens,
		TransferAmount: 300000,
	}

	err = biz.ProcessSePayWebhook(ctx, webhook)
	require.NoError(t, err)

	mockOutbox.AssertCalled(t, "CreateOutboxEvent", mock.Anything,
		string(entity.EventTypePaymentCompleted), mock.Anything)
}
