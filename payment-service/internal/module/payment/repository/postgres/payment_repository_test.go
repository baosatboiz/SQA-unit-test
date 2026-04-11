package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"payment-service/internal/module/payment/entity"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// newMockPaymentDB creates a fresh isolated sqlmock database for each test case.
// This isolation is the unit-test equivalent of restoring database state between tests.
func newMockPaymentDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	cleanup := func() {
		mock.ExpectClose()
		require.NoError(t, bunDB.Close())
	}

	return bunDB, mock, cleanup
}

// samplePayment returns stable test data used across repository tests.
func samplePayment() *entity.Payment {
	now := time.Date(2026, time.April, 11, 10, 30, 0, 0, time.UTC)
	transactionID := "tx-001"
	payload := "{\"source\":\"test\"}"

	return &entity.Payment{
		Id:            "pay-001",
		BookingId:     "booking-123",
		Amount:        150000.00,
		PaymentDate:   now,
		PaymentMethod: entity.PaymentMethodCash,
		TransactionId: &transactionID,
		Status:        entity.PaymentStatusPending,
		Payload:       &payload,
		CreatedAt:     now,
	}
}

// newPaymentRowResult builds a single-row result set that mimics the payments table shape.
func newPaymentRowResult(payment *entity.Payment) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "booking_id", "amount", "payment_date", "payment_method",
		"transaction_id", "status", "payload", "created_at", "updated_at",
	}).AddRow(
		payment.Id, payment.BookingId, payment.Amount, payment.PaymentDate,
		nil, payment.TransactionId, nil, payment.Payload, payment.CreatedAt, nil,
	)
}

// Test Case ID: PAY-TC-054
// Purpose: Verify Create executes INSERT against DB (CheckDB by SQL expectation).
// Rollback: With sqlmock there is no real persisted state; each test uses isolated mock DB and closes it in cleanup.
func TestPaymentRepository_Create_Success(t *testing.T) {
	db, mock, cleanup := newMockPaymentDB(t)
	defer cleanup()

	expectedPayment := samplePayment()

	// CheckDB: Ensure repository issues INSERT statement.
	mock.ExpectQuery("INSERT INTO").
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(nil))

	repo := NewPaymentRepository(db)
	err := repo.Create(context.Background(), expectedPayment)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Test Case ID: PAY-TC-055
// Purpose: Verify GetById reads expected payment data from DB and maps fields correctly.
func TestPaymentRepository_GetById_Success(t *testing.T) {
	db, mock, cleanup := newMockPaymentDB(t)
	defer cleanup()

	expectedPayment := samplePayment()
	mockRows := newPaymentRowResult(expectedPayment)

	// CheckDB: Ensure SELECT is executed and returns expected database row.
	mock.ExpectQuery("SELECT").
		WillReturnRows(mockRows)

	repo := NewPaymentRepository(db)
	actualPayment, err := repo.GetById(context.Background(), expectedPayment.Id)

	require.NoError(t, err)
	assert.Equal(t, expectedPayment.Id, actualPayment.Id)
	assert.Equal(t, expectedPayment.BookingId, actualPayment.BookingId)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Test Case ID: PAY-TC-057
// Purpose: Verify GetById returns not-found error when DB has no matching record.
func TestPaymentRepository_GetById_NotFound(t *testing.T) {
	db, mock, cleanup := newMockPaymentDB(t)
	defer cleanup()

	// CheckDB: SELECT is still expected, but database returns no rows.
	mock.ExpectQuery("SELECT").
		WillReturnError(sql.ErrNoRows)

	repo := NewPaymentRepository(db)
	actualPayment, err := repo.GetById(context.Background(), "missing-id")

	assert.Nil(t, actualPayment)
	assert.EqualError(t, err, "payment not found")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Test Case ID: PAY-TC-056
// Purpose: Verify FindByBookingId reads expected payment data from DB.
func TestPaymentRepository_FindByBookingId_Success(t *testing.T) {
	db, mock, cleanup := newMockPaymentDB(t)
	defer cleanup()

	expectedPayment := samplePayment()
	mockRows := newPaymentRowResult(expectedPayment)

	// CheckDB: Ensure SELECT by booking_id is executed.
	mock.ExpectQuery("SELECT").
		WillReturnRows(mockRows)

	repo := NewPaymentRepository(db)
	actualPayment, err := repo.FindByBookingId(context.Background(), expectedPayment.BookingId)

	require.NoError(t, err)
	assert.Equal(t, expectedPayment.BookingId, actualPayment.BookingId)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Test Case ID: PAY-TC-058
// Purpose: Verify FindByShortCode accesses DB and maps returned payment correctly.
func TestPaymentRepository_FindByShortCode_Success(t *testing.T) {
	db, mock, cleanup := newMockPaymentDB(t)
	defer cleanup()

	expectedPayment := samplePayment()
	mockRows := newPaymentRowResult(expectedPayment)

	// CheckDB: Ensure SELECT query is issued for short-code lookup.
	mock.ExpectQuery("SELECT").
		WillReturnRows(mockRows)

	repo := NewPaymentRepository(db)
	actualPayment, err := repo.FindByShortCode(context.Background(), "C1234567")

	require.NoError(t, err)
	assert.Equal(t, expectedPayment.BookingId, actualPayment.BookingId)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Test Case ID: PAY-TC-059
// Purpose: Verify FindByUUIDNoHyphens accesses DB and returns expected payment.
func TestPaymentRepository_FindByUUIDNoHyphens_Success(t *testing.T) {
	db, mock, cleanup := newMockPaymentDB(t)
	defer cleanup()

	expectedPayment := samplePayment()
	mockRows := newPaymentRowResult(expectedPayment)

	// CheckDB: Ensure SELECT query is issued for UUID-no-hyphens lookup.
	mock.ExpectQuery("SELECT").
		WillReturnRows(mockRows)

	repo := NewPaymentRepository(db)
	actualPayment, err := repo.FindByUUIDNoHyphens(context.Background(), "BOOKING123")

	require.NoError(t, err)
	assert.Equal(t, expectedPayment.BookingId, actualPayment.BookingId)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Test Case ID: PAY-TC-060
// Purpose: Verify UpdatePaymentFields executes UPDATE against DB (CheckDB by SQL expectation).
// Rollback: Test starts a transaction explicitly and rolls it back to ensure no state leakage.
func TestPaymentRepository_UpdatePaymentFields_Success(t *testing.T) {
	db, mock, cleanup := newMockPaymentDB(t)
	defer cleanup()

	// Begin test transaction (this is test-controlled rollback scope).
	mock.ExpectBegin()
	testTx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	// CheckDB: Ensure UPDATE statement is executed.
	mock.ExpectExec("UPDATE").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Ensure test transaction is rolled back at the end.
	mock.ExpectRollback()

	repo := NewPaymentRepository(db)
	err = repo.UpdatePaymentFields(context.Background(), testTx, "pay-001", map[string]interface{}{
		"status":     entity.PaymentStatusCompleted,
		"updated_at": time.Now(),
	})

	require.NoError(t, err)
	require.NoError(t, testTx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}
