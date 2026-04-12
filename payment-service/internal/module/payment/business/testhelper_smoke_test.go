// Smoke test for testhelper — verifies that the test DB is reachable and that
// a write inside a transaction gets rolled back at defer.
package business

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment-service/internal/module/payment/entity"
)

// TC-HELPER-SMOKE-1: the DB helper can open a tx, run an INSERT, and
// guarantee that a ROLLBACK returns the DB to its prior state.
func TestTestHelper_RollbackLeavesNoTrace(t *testing.T) {
	ctx := context.Background()

	db := openTestDB(t)
	defer db.Close()

	const sentinelID = "smoke-helper-sentinel-id"

	// ---- Pre-condition: the sentinel row must not exist yet ----
	var countBefore int
	err := db.NewSelect().
		Model((*entity.Payment)(nil)).
		Where("id = ?", sentinelID).
		ColumnExpr("COUNT(*)").
		Scan(ctx, &countBefore)
	require.NoError(t, err)
	require.Equal(t, 0, countBefore, "DB must start clean for this test")

	// ---- Open a test transaction, insert a row, verify it is visible inside
	//      the tx, then rollback ----
	tx := beginTestTx(t, db)

	_, err = tx.NewInsert().Model(&entity.Payment{
		Id:          sentinelID,
		BookingId:   "smoke-booking",
		Amount:      1.0,
		PaymentDate: nowForTest(),
		Status:      entity.PaymentStatusPending,
		CreatedAt:   nowForTest(),
	}).Exec(ctx)
	require.NoError(t, err, "INSERT inside test tx must succeed (FK disabled by SET LOCAL)")

	var inTxCount int
	err = tx.NewSelect().
		Model((*entity.Payment)(nil)).
		Where("id = ?", sentinelID).
		ColumnExpr("COUNT(*)").
		Scan(ctx, &inTxCount)
	require.NoError(t, err)
	assert.Equal(t, 1, inTxCount, "row must be visible inside its own tx")

	rollbackTx(tx)

	// ---- Post-condition: after rollback the row must not exist ----
	var countAfter int
	err = db.NewSelect().
		Model((*entity.Payment)(nil)).
		Where("id = ?", sentinelID).
		ColumnExpr("COUNT(*)").
		Scan(ctx, &countAfter)
	require.NoError(t, err)
	assert.Equal(t, 0, countAfter, "rollback must leave DB unchanged")
}
