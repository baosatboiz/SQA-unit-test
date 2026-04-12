// Package business — VerifyCryptoPayment unit tests (PAY-TC-015 → PAY-TC-018).
//
// VerifyCryptoPayment performs three coordinated steps:
//
//   1. Ask the blockchain service to verify the on-chain transaction.
//   2. Look up the payment row by bookingId and update it with crypto details
//      (transaction_id, payment_method=CRYPTOCURRENCY, status=COMPLETED).
//   3. Emit an outbox event PAYMENT_COMPLETED.
//
// We mock the blockchain service so the test is offline, but the DB update
// runs against the real test PostgreSQL inside a transaction that we roll back.

package business

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"payment-service/internal/module/payment/entity"
	repository "payment-service/internal/module/payment/repository/postgres"
)

// -----------------------------------------------------------------------------
// TC-015 — blockchain verification fails  →  wrapped error
// -----------------------------------------------------------------------------

// If blockchainService.VerifyTransaction returns an error (invalid tx hash,
// tx not found, amount mismatch, etc.), VerifyCryptoPayment must NOT touch
// the database and must return an error that starts with "blockchain
// verification failed".
func TestVerifyCryptoPayment_TC015_ErrorWhenBlockchainVerificationFails(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	mockBC := new(MockBlockchainService)
	mockBC.On("VerifyTransaction",
		mock.Anything, "0xInvalid", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("invalid transaction hash format"))

	mockOutbox := new(MockOutboxClient)

	biz := &paymentBiz{
		db:                tx,
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: mockBC,
	}

	req := &entity.CryptoVerificationRequest{
		BookingId:   "booking-tc015",
		TxHash:      "0xInvalid",
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",
		AmountEth:   "0.1",
		AmountVnd:   2000000,
	}

	err := biz.VerifyCryptoPayment(ctx, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "blockchain verification failed")
	mockOutbox.AssertNotCalled(t, "CreateOutboxEvent", mock.Anything, mock.Anything, mock.Anything)
}

// -----------------------------------------------------------------------------
// TC-016 — payment not found for the given bookingId  →  repo error surfaced
// -----------------------------------------------------------------------------

// When blockchain verification succeeds but there is NO payment row for the
// given bookingId, the repository's not-found error must be surfaced. We do
// not want to create phantom payments from the crypto path.
func TestVerifyCryptoPayment_TC016_ErrorWhenPaymentNotFound(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	mockBC := new(MockBlockchainService)
	mockBC.On("VerifyTransaction",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil) // chain check passes

	biz := &paymentBiz{
		db:                tx,
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      new(MockOutboxClient),
		blockchainService: mockBC,
	}

	req := &entity.CryptoVerificationRequest{
		BookingId:   "booking-tc016-does-not-exist",
		TxHash:      "0x0000000000000000000000000000000000000000000000000000000000000001",
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",
		AmountEth:   "0.1",
	}

	err := biz.VerifyCryptoPayment(ctx, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "payment not found",
		"repo's 'payment not found' error must be surfaced")
}

// -----------------------------------------------------------------------------
// TC-017 — Happy path: payment row is updated with crypto details (CheckDB)
// -----------------------------------------------------------------------------

// On success, VerifyCryptoPayment must update the payment row so that:
//
//   - transaction_id = request.TxHash
//   - payment_method = CRYPTOCURRENCY
//   - status         = COMPLETED
//
// We verify with a live SELECT from the tx (CheckDB).
func TestVerifyCryptoPayment_TC017_UpdatesPaymentWithCryptoDetails(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	// Seed a pending payment tied to our test bookingId
	const bookingID = "booking-tc017"
	const paymentID = "pay-tc017"
	_, err := tx.NewInsert().Model(&entity.Payment{
		Id:          paymentID,
		BookingId:   bookingID,
		Amount:      500000,
		PaymentDate: time.Now(),
		Status:      entity.PaymentStatusPending,
		CreatedAt:   time.Now(),
	}).Exec(ctx)
	require.NoError(t, err)

	mockBC := new(MockBlockchainService)
	mockBC.On("VerifyTransaction",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	mockOutbox := new(MockOutboxClient)
	mockOutbox.On("CreateOutboxEvent",
		mock.Anything, mock.Anything, mock.Anything).Return(nil)

	biz := &paymentBiz{
		db:                tx,
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: mockBC,
	}

	const txHash = "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	req := &entity.CryptoVerificationRequest{
		BookingId:   bookingID,
		TxHash:      txHash,
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",
		AmountEth:   "0.01",
		AmountVnd:   500000,
	}

	err = biz.VerifyCryptoPayment(ctx, req)
	require.NoError(t, err)

	// ---- CheckDB ----
	var updated entity.Payment
	err = tx.NewSelect().Model(&updated).Where("id = ?", paymentID).Scan(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.PaymentStatusCompleted, updated.Status)
	assert.Equal(t, entity.PaymentMethodCryptoCurrency, updated.PaymentMethod)
	require.NotNil(t, updated.TransactionId)
	assert.Equal(t, txHash, *updated.TransactionId, "transaction_id must equal tx hash")
}

// -----------------------------------------------------------------------------
// TC-018 — Outbox event fired with tx_hash field
// -----------------------------------------------------------------------------

// After the DB update, VerifyCryptoPayment must publish an outbox event of
// type PAYMENT_COMPLETED whose payload carries `tx_hash`. We verify this by
// asserting the mock was called with a payload map that contains tx_hash.
func TestVerifyCryptoPayment_TC018_EmitsOutboxEventWithTxHash(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	// Seed the payment so the repo lookup succeeds.
	const bookingID = "booking-tc018"
	_, err := tx.NewInsert().Model(&entity.Payment{
		Id:          "pay-tc018",
		BookingId:   bookingID,
		Amount:      800000,
		PaymentDate: time.Now(),
		Status:      entity.PaymentStatusPending,
		CreatedAt:   time.Now(),
	}).Exec(ctx)
	require.NoError(t, err)

	mockBC := new(MockBlockchainService)
	mockBC.On("VerifyTransaction",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	mockOutbox := new(MockOutboxClient)
	// Accept any event and record its arguments so we can introspect them.
	mockOutbox.On("CreateOutboxEvent",
		mock.Anything,
		string(entity.EventTypePaymentCompleted),
		mock.MatchedBy(func(data interface{}) bool {
			// data is map[string]interface{}; check that tx_hash key is set
			m, ok := data.(map[string]interface{})
			if !ok {
				return false
			}
			txHash, has := m["tx_hash"]
			return has && txHash != nil
		})).Return(nil)

	biz := &paymentBiz{
		db:                tx,
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: mockBC,
	}

	req := &entity.CryptoVerificationRequest{
		BookingId:   bookingID,
		TxHash:      "0xaabbccddeeff00112233445566778899aabbccddeeff00112233445566778899",
		FromAddress: "0x1111111111111111111111111111111111111111",
		ToAddress:   "0x2222222222222222222222222222222222222222",
		AmountEth:   "0.016",
		AmountVnd:   800000,
	}

	err = biz.VerifyCryptoPayment(ctx, req)
	require.NoError(t, err)

	mockOutbox.AssertExpectations(t)
}
