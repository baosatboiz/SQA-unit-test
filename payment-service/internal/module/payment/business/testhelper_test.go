// Package business — DB-oriented test helpers.
//
// This file provides helpers that let DB-touching unit tests satisfy the
// assignment's two specific requirements:
//
//  1. CheckDB   — verify that the real database actually changed as expected.
//  2. Rollback  — after each test, leave the database in the same state it
//                 had BEFORE the test (no leaked rows).
//
// Strategy:
//   - Connect once to the real PostgreSQL running in docker-compose on :5433
//   - For every test, open a bun.Tx via BeginTx
//   - Inside that tx, set `session_replication_role = replica` so foreign-key
//     triggers are disabled for the duration of the tx. This lets us insert
//     payments without having to seed bookings/users/showtimes/etc.
//   - At defer, call Rollback — PostgreSQL drops every write we made AND the
//     session_replication_role setting (because it was SET LOCAL).
//
// The net effect is that each test has its own isolated sandbox in the real
// DB, and no test ever leaves a trace behind.

package business

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// nowForTest returns the current wall-clock time. Extracted so individual
// tests have a single, readable place to override time if ever needed.
func nowForTest() time.Time { return time.Now() }

// testDSN returns the Postgres connection string used by DB-touching tests.
// It first honors the TEST_DB_DSN env var (so CI can override); otherwise it
// falls back to the docker-compose local URL used by the project.
func testDSN() string {
	if v := os.Getenv("TEST_DB_DSN"); v != "" {
		return v
	}
	// Matches docker-compose.yml: postgres:5432 inside network, :5433 on host.
	return "postgres://postgres:Trang%40051203@localhost:5433/cinema_app?sslmode=disable"
}

// openTestDB opens a *bun.DB pointing at the test PostgreSQL instance.
// The caller owns the returned handle and must Close() it.
func openTestDB(t *testing.T) *bun.DB {
	t.Helper()
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(testDSN())))
	db := bun.NewDB(sqldb, pgdialect.New())

	// Fail fast if the DB isn't reachable — tells the developer to start
	// docker-compose instead of leaving an ugly error mid-test.
	require.NoError(t, db.PingContext(context.Background()),
		"cannot reach test PostgreSQL at %s — is `docker start postgres` running?", testDSN())
	return db
}

// beginTestTx opens a transaction and disables foreign-key enforcement for its
// duration. It returns the transaction; the caller MUST `defer tx.Rollback()`.
//
// Why SET LOCAL session_replication_role = replica?
//   The payments table has an FK to bookings(id). Creating a real booking would
//   in turn require users, showtimes, movies, rooms, seats, etc. — an expensive
//   fixture for a unit test. Setting session_replication_role to 'replica' tells
//   Postgres to skip session-level triggers and FK checks for this transaction
//   only. It is the standard Postgres-level isolation trick used in Rails/Django
//   test suites. After ROLLBACK the setting is gone.
func beginTestTx(t *testing.T, db *bun.DB) bun.Tx {
	t.Helper()
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err, "BeginTx failed")

	_, err = tx.ExecContext(ctx, "SET LOCAL session_replication_role = 'replica'")
	require.NoError(t, err, "could not disable FK triggers for test tx")
	return tx
}

// rollbackTx is a defer-friendly wrapper that ignores errors from Rollback.
// A rollback can error if the connection is already closed; for unit tests we
// simply don't care.
func rollbackTx(tx bun.Tx) {
	_ = tx.Rollback()
}
