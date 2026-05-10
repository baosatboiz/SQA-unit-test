package postgres

import (
	"context"
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"movie-service/internal/module/room/entity"
)

func setupRoomDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { sqldb.Close() })
	return bun.NewDB(sqldb, pgdialect.New()), mock
}

func newRoomRepo(db *bun.DB) *Repository {
	return &Repository{db: db, roDb: db}
}

// ---------------------------------------------------------------------------
// Tests P3_102 - P3_108 (Room Repository)
// ---------------------------------------------------------------------------

// P3_102: roomRepository.GetByID() - NotFound
func TestRoomRepo_P3_102_GetByID_NotFound(t *testing.T) {
	db, mock := setupRoomDB(t)
	repo := newRoomRepo(db)
	mock.ExpectQuery(`SELECT`).WillReturnError(sql.ErrNoRows)
	res, err := repo.GetByID(context.Background(), "bad-id")
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, res)
}

// P3_103: roomRepository.GetByID() - Success
func TestRoomRepo_P3_103_GetByID_Success(t *testing.T) {
	db, mock := setupRoomDB(t)
	repo := newRoomRepo(db)
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id", "room_number"}).AddRow("r1", 101))
	res, err := repo.GetByID(context.Background(), "r1")
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "r1", res.Id)
}

// P3_104: roomRepository.GetMany() - FilterByType
func TestRoomRepo_P3_104_GetMany_FilterByType(t *testing.T) {
	db, mock := setupRoomDB(t)
	repo := newRoomRepo(db)
	mock.ExpectQuery(`SELECT.*room_type.*=.*VIP`).WillReturnRows(sqlmock.NewRows([]string{"id", "room_type"}).AddRow("r-vip", "VIP"))
	res, err := repo.GetMany(context.Background(), 10, 0, "", entity.RoomTypeVIP, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, res)
}

// P3_105: roomRepository.Create() - Success
func TestRoomRepo_P3_105_Create_Success(t *testing.T) {
	db, mock := setupRoomDB(t)
	repo := newRoomRepo(db)
	mock.ExpectQuery(`INSERT`).WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("ACTIVE"))
	err := repo.Create(context.Background(), &entity.Room{Id: "r1", RoomNumber: 101})
	assert.NoError(t, err)
}

// P3_106: roomRepository.Update() - Success
func TestRoomRepo_P3_106_Update_Success(t *testing.T) {
	db, mock := setupRoomDB(t)
	repo := newRoomRepo(db)
	mock.ExpectExec(`UPDATE`).WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.Update(context.Background(), &entity.Room{Id: "r1", Capacity: 100})
	assert.NoError(t, err)
}

// P3_107: roomRepository.Delete() - Success
func TestRoomRepo_P3_107_Delete_Success(t *testing.T) {
	db, mock := setupRoomDB(t)
	repo := newRoomRepo(db)
	mock.ExpectExec(`DELETE`).WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.Delete(context.Background(), "r1")
	assert.NoError(t, err)
}

// P3_108: roomRepository.Delete() - DBError
func TestRoomRepo_P3_108_Delete_DBError(t *testing.T) {
	db, mock := setupRoomDB(t)
	repo := newRoomRepo(db)
	mock.ExpectExec(`DELETE`).WillReturnError(sql.ErrConnDone)
	err := repo.Delete(context.Background(), "r1")
	assert.ErrorIs(t, err, sql.ErrConnDone)
}
