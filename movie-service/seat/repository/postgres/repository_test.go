package postgres

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/stretchr/testify/assert"

	"movie-service/internal/module/seat/entity"
)

func setupSeatDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { sqldb.Close() })
	return bun.NewDB(sqldb, pgdialect.New()), mock
}

func newSeatRepo(db *bun.DB) *Repository {
	return &Repository{db: db, roDb: db}
}

// ---------------------------------------------------------------------------
// Tests P3_109 - P3_115 (Seat Repository)
// ---------------------------------------------------------------------------

// P3_109: seatRepository.GetByID() - Success
func TestSeatRepo_P3_109_GetByID_Success(t *testing.T) {
	db, mock := setupSeatDB(t)
	repo := newSeatRepo(db)
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("s1"))
	result, err := repo.GetByID(context.Background(), "s1")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// P3_110: seatRepository.GetMany() - Success (Lấy seat của room)
func TestSeatRepo_P3_110_GetMany_Success(t *testing.T) {
	db, mock := setupSeatDB(t)
	repo := newSeatRepo(db)
	mock.ExpectQuery(`SELECT.*room_id.*=.*r1`).WillReturnRows(sqlmock.NewRows([]string{"id", "room_id"}).AddRow("s1", "r1"))
	results, err := repo.GetMany(context.Background(), 10, 0, "", "r1", "", "", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, results)
}

// P3_111: seatRepository.GetMany() - EmptyRoom
func TestSeatRepo_P3_111_GetMany_EmptyRoom(t *testing.T) {
	db, mock := setupSeatDB(t)
	repo := newSeatRepo(db)
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	results, err := repo.GetMany(context.Background(), 10, 0, "", "r2", "", "", "")
	assert.NoError(t, err)
	assert.Len(t, results, 0)
}

// P3_112: seatRepository.GetByIDs() - Batch
func TestSeatRepo_P3_112_GetByIDs_Batch(t *testing.T) {
	db, mock := setupSeatDB(t)
	repo := newSeatRepo(db)
	mock.ExpectQuery(`SELECT.*id.*IN`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("s1").AddRow("s2"))
	results, err := repo.GetByIDs(context.Background(), []string{"s1", "s2"})
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

// P3_113: seatRepository.Create() - Success
func TestSeatRepo_P3_113_Create_Success(t *testing.T) {
	db, mock := setupSeatDB(t)
	repo := newSeatRepo(db)
	mock.ExpectQuery(`INSERT`).WillReturnRows(sqlmock.NewRows([]string{"seat_type", "status"}).AddRow("REGULAR", "AVAILABLE"))
	err := repo.Create(context.Background(), &entity.Seat{Id: "s1", RoomId: "r1"})
	assert.NoError(t, err)
}

// P3_114: seatRepository.Update() - Success
func TestSeatRepo_P3_114_Update_Success(t *testing.T) {
	db, mock := setupSeatDB(t)
	repo := newSeatRepo(db)
	mock.ExpectExec(`UPDATE`).WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.Update(context.Background(), &entity.Seat{Id: "s1", SeatType: "VIP"})
	assert.NoError(t, err)
}

// P3_115: seatRepository.Delete() - Success
func TestSeatRepo_P3_115_Delete_Success(t *testing.T) {
	db, mock := setupSeatDB(t)
	repo := newSeatRepo(db)
	mock.ExpectExec(`DELETE`).WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.Delete(context.Background(), "s1")
	assert.NoError(t, err)
}
