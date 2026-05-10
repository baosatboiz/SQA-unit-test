package postgres

import (
	"context"
	"testing"
	"time"

	"movie-service/internal/module/showtime/entity"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func setupShowtimeDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { sqldb.Close() })
	return bun.NewDB(sqldb, pgdialect.New()), mock
}

func newShowtimeRepo(db *bun.DB) *Repository {
	return &Repository{db: db, roDb: db}
}

// ---------------------------------------------------------------------------
// Tests P3_095 - P3_100 (Showtime Repository)
// ---------------------------------------------------------------------------

// P3_095: showtimeRepository.CheckConflict() - NoOverlap
func TestShowtimeRepo_P3_095_CheckConflict_NoOverlap(t *testing.T) {
	db, mock := setupShowtimeDB(t)
	repo := newShowtimeRepo(db)
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	conflict, err := repo.CheckConflict(context.Background(), "r1", time.Now(), time.Now().Add(time.Hour), "")
	assert.NoError(t, err)
	assert.False(t, conflict)
}

// P3_096: showtimeRepository.CheckConflict() - Overlap
func TestShowtimeRepo_P3_096_CheckConflict_Overlap(t *testing.T) {
	db, mock := setupShowtimeDB(t)
	repo := newShowtimeRepo(db)
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	conflict, err := repo.CheckConflict(context.Background(), "r1", time.Now(), time.Now().Add(time.Hour), "")
	assert.NoError(t, err)
	assert.True(t, conflict)
}

// P3_097: showtimeRepository.GetByID() - Success
func TestShowtimeRepo_P3_097_GetByID_Success(t *testing.T) {
	db, mock := setupShowtimeDB(t)
	repo := newShowtimeRepo(db)
	mock.ExpectQuery(`SELECT`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "movie_id", "room_id", "start_time", "end_time", "format", "base_price", "status", "created_at", "updated_at"}).
			AddRow("st1", "m1", "r1", time.Now(), time.Now().Add(2*time.Hour), "2D", 100.0, "SCHEDULED", time.Now(), time.Now()),
	)
	// Relation mocks for Movie and Room
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("m1"))
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("r1"))
	// Seat mock
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	result, err := repo.GetByID(context.Background(), "st1")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "st1", result.Id)
}

// P3_098: showtimeRepository.GetMany() - Range
func TestShowtimeRepo_P3_098_GetMany_Range(t *testing.T) {
	db, mock := setupShowtimeDB(t)
	repo := newShowtimeRepo(db)
	from := time.Now(); to := from.Add(24*time.Hour)
	mock.ExpectQuery(`SELECT.*start_time.*>=.*AND.*start_time.*<=`).WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow("st-1"),
	)
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id"})) // Movie
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id"})) // Room
	results, err := repo.GetMany(context.Background(), 10, 0, "", "", "", "", "", &from, &to, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, results)
}

// P3_099: showtimeRepository.Create() - Success
func TestShowtimeRepo_P3_099_Create_Success(t *testing.T) {
	db, mock := setupShowtimeDB(t)
	repo := newShowtimeRepo(db)
	mock.ExpectQuery(`INSERT`).WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("SCHEDULED"))
	err := repo.Create(context.Background(), &entity.Showtime{Id: "st-1", MovieId: "m1", RoomId: "r1"})
	assert.NoError(t, err)
}

// P3_100: showtimeRepository.Update() - Success
func TestShowtimeRepo_P3_100_Update_Success(t *testing.T) {
	db, mock := setupShowtimeDB(t)
	repo := newShowtimeRepo(db)
	mock.ExpectExec(`UPDATE`).WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.Update(context.Background(), &entity.Showtime{Id: "st-1", Format: "IMAX"})
	assert.NoError(t, err)
}

// P3_101: showtimeRepository.Delete() - Success
func TestShowtimeRepo_P3_101_Delete_Success(t *testing.T) {
	db, mock := setupShowtimeDB(t)
	repo := newShowtimeRepo(db)
	mock.ExpectExec(`DELETE FROM.*showtimes.*WHERE.*id`).WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.Delete(context.Background(), "st1")
	assert.NoError(t, err)
}
