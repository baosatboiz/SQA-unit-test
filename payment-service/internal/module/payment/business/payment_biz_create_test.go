// Package business — CreatePayment unit tests (PAY-TC-001 → PAY-TC-005).
//
// These tests exercise paymentBiz.CreatePayment using the REAL PostgreSQL
// database so that the CheckDB and Rollback requirements are literally
// satisfied: every assertion that says "the row exists" is a live SELECT,
// and every test rolls back its transaction on exit.
//
// The repository is constructed around a *bun.Tx (not *bun.DB), so all INSERTs
// are contained inside the test's transaction. OutboxClient and BlockchainService
// are mocked because they are not the subject of these test cases.

package business

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"payment-service/internal/module/payment/entity"
	repository "payment-service/internal/module/payment/repository/postgres"
)

// -----------------------------------------------------------------------------
// CreatePayment — PAY-TC-001 through PAY-TC-005
// -----------------------------------------------------------------------------

// TC-001 — Idempotency: if a payment already exists for the given bookingId,
// CreatePayment must return the EXISTING record and NOT insert a new one.
// This matches the "return existing payment if already exists" contract.
func TestCreatePayment_TC001_ReturnsExistingPaymentForDuplicateBookingId(t *testing.T) {
	ctx := context.Background()

	// ---- Arrange: real tx + seed an existing payment row ----
	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	existingBookingID := "booking-tc001"
	seededPaymentID := uuid.New().String()
	seededPayment := &entity.Payment{
		Id:          seededPaymentID,
		BookingId:   existingBookingID,
		Amount:      200000,
		PaymentDate: time.Now(),
		Status:      entity.PaymentStatusPending,
		CreatedAt:   time.Now(),
	}
	_, err := tx.NewInsert().Model(seededPayment).Exec(ctx)
	require.NoError(t, err, "seeding existing payment must succeed")

	mockOutbox := new(MockOutboxClient)
	mockBC := new(MockBlockchainService)
	biz := &paymentBiz{
		db:                nil, // unused in CreatePayment
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: mockBC,
	}

	// ---- Act ----
	got, err := biz.CreatePayment(ctx, existingBookingID, 999999 /* different amount */)

	// ---- Assert ----
	require.NoError(t, err)
	assert.Equal(t, seededPaymentID, got.Id, "must return the pre-existing payment's Id")
	assert.Equal(t, 200000.0, got.Amount, "amount must match the existing row, not the new input")

	// CheckDB: verify there is still exactly 1 payment row for this bookingId
	var count int
	err = tx.NewSelect().
		Model((*entity.Payment)(nil)).
		Where("booking_id = ?", existingBookingID).
		ColumnExpr("COUNT(*)").
		Scan(ctx, &count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "no new row must be created when payment already exists")
}

// TC-002 — CreatePayment inserts a new payment whose Id is a valid UUID
// (google/uuid.Parse must succeed) and whose status is PENDING. The row must
// end up physically in the database (CheckDB) and must be rolled back at defer.
func TestCreatePayment_TC002_CreatesNewPaymentWithUUIDAndPendingStatus(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()
	tx := beginTestTx(t, db)
	defer rollbackTx(tx)

	mockOutbox := new(MockOutboxClient)
	mockBC := new(MockBlockchainService)
	biz := &paymentBiz{
		repo:              repository.NewPaymentRepository(tx),
		outboxClient:      mockOutbox,
		blockchainService: mockBC,
	}

	bookingID := "booking-tc002"
	amount := 150000.0

	// ---- Act ----
	payment, err := biz.CreatePayment(ctx, bookingID, amount)

	// ---- Assert (business-level) ----
	require.NoError(t, err)
	require.NotNil(t, payment)
	_, parseErr := uuid.Parse(payment.Id)
	assert.NoError(t, parseErr, "Id must be a valid UUID string")
	assert.Equal(t, entity.PaymentStatusPending, payment.Status)

	// ---- CheckDB: the row must physically exist in payments table ----
	var row entity.Payment
	err = tx.NewSelect().Model(&row).Where("id = ?", payment.Id).Scan(ctx)
	require.NoError(t, err, "payment row must exist in DB after CreatePayment")
	assert.Equal(t, bookingID, row.BookingId)
	assert.Equal(t, 150000.0, row.Amount)
	assert.Equal(t, entity.PaymentStatusPending, row.Status)
}

// TC-003 — PaymentDate must be set to the current time (within a reasonable
// tolerance window so the test is not flaky on slow machines).
func TestCreatePayment_TC003_SetsPaymentDateToNow(t *testing.T) {
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

	before := time.Now()
	payment, err := biz.CreatePayment(ctx, "booking-tc003", 100000)
	after := time.Now()
	require.NoError(t, err)

	// PaymentDate must fall within [before, after] (allowing tiny drift)
	assert.WithinRange(t, payment.PaymentDate, before.Add(-time.Second), after.Add(time.Second),
		"PaymentDate must be approximately time.Now() at CreatePayment call")
}

// TC-004 — CreatedAt must also be set to the current time. This is a separate
// contract from PaymentDate; both are written by the business layer.
func TestCreatePayment_TC004_SetsCreatedAtToNow(t *testing.T) {
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

	before := time.Now()
	payment, err := biz.CreatePayment(ctx, "booking-tc004", 120000)
	after := time.Now()
	require.NoError(t, err)

	assert.WithinRange(t, payment.CreatedAt, before.Add(-time.Second), after.Add(time.Second),
		"CreatedAt must be approximately time.Now() at CreatePayment call")
}

// TC-005 — When the repository's Create() fails (e.g., DB down, constraint
// violation), CreatePayment must wrap the error with the sentinel message
// "failed to create payment". We use a mock repository here to simulate the
// failure deterministically — the test does not touch the real DB at all.
func TestCreatePayment_TC005_ReturnsWrappedErrorWhenRepoCreateFails(t *testing.T) {
	ctx := context.Background()

	mockRepo := new(MockPaymentRepository)
	// FindByBookingId returns nil → CreatePayment proceeds to Create()
	mockRepo.On("FindByBookingId", mock.Anything, "booking-tc005").
		Return(nil, errors.New("not found"))
	// Create() fails — simulate DB connection error
	mockRepo.On("Create", mock.Anything, mock.Anything).
		Return(errors.New("connection refused"))

	biz := &paymentBiz{
		repo:              mockRepo,
		outboxClient:      new(MockOutboxClient),
		blockchainService: new(MockBlockchainService),
	}

	payment, err := biz.CreatePayment(ctx, "booking-tc005", 100000)

	assert.Nil(t, payment, "payment must be nil on error path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create payment",
		"error must carry the 'failed to create payment' sentinel")

	mockRepo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
}
