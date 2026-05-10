package postgres

import (
	"context"
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/stretchr/testify/assert"

	"movie-service/internal/module/movie/entity"
)

func setupMovieDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { sqldb.Close() })
	return bun.NewDB(sqldb, pgdialect.New()), mock
}

func newMovieRepo(db *bun.DB) *Repository {
	return &Repository{db: db, roDb: db}
}

// ---------------------------------------------------------------------------
// Tests P3_085 - P3_094 (Movie Repository)
// ---------------------------------------------------------------------------

// P3_085: movieRepository.GetByID() - Found
func TestMovieRepo_P3_085_GetByID_Found(t *testing.T) {
	db, mock := setupMovieDB(t)
	repo := newMovieRepo(db)
	mock.ExpectQuery(`SELECT`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "title", "status"}).
			AddRow("m1", "Test", "SHOWING"),
	)
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"movie_id", "genre_id"}))
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

	res, err := repo.GetByID(context.Background(), "m1")
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

// P3_086: movieRepository.GetByID() - NotFound
func TestMovieRepo_P3_086_GetByID_NotFound(t *testing.T) {
	db, mock := setupMovieDB(t)
	repo := newMovieRepo(db)
	mock.ExpectQuery(`SELECT`).WillReturnError(sql.ErrNoRows)
	res, err := repo.GetByID(context.Background(), "bad")
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, res)
}

// P3_087: movieRepository.GetByID() - DBError
func TestMovieRepo_P3_087_GetByID_DBError(t *testing.T) {
	db, mock := setupMovieDB(t)
	repo := newMovieRepo(db)
	mock.ExpectQuery(`SELECT`).WillReturnError(sql.ErrConnDone)
	_, err := repo.GetByID(context.Background(), "m1")
	assert.ErrorIs(t, err, sql.ErrConnDone)
}

// P3_088: movieRepository.GetMany() - Pagination
func TestMovieRepo_P3_088_GetMany_Pagination(t *testing.T) {
	db, mock := setupMovieDB(t)
	repo := newMovieRepo(db)
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("m1"))
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	res, err := repo.GetMany(context.Background(), 10, 0, "", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, res)
}

// P3_089: movieRepository.GetMany() - FilterByStatus
func TestMovieRepo_P3_089_GetMany_FilterByStatus(t *testing.T) {
	db, mock := setupMovieDB(t)
	repo := newMovieRepo(db)
	mock.ExpectQuery(`SELECT.*status.*=.*SHOWING`).WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow("m1", "SHOWING"))
	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	res, err := repo.GetMany(context.Background(), 10, 0, "", "SHOWING")
	assert.NoError(t, err)
	assert.Equal(t, entity.MovieStatusShowing, res[0].Status)
}

// P3_090: movieRepository.Create() - Success
func TestMovieRepo_P3_090_Create_Success(t *testing.T) {
	db, mock := setupMovieDB(t)
	repo := newMovieRepo(db)
	mock.ExpectQuery(`INSERT`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("m1"))
	mock.ExpectExec(`INSERT`).WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.Create(context.Background(), &entity.Movie{Id: "m1", Title: "T"}, []string{"g1"})
	assert.NoError(t, err)
}

// P3_091: movieRepository.Create() - DBError
func TestMovieRepo_P3_091_Create_DBError(t *testing.T) {
	db, mock := setupMovieDB(t)
	repo := newMovieRepo(db)
	mock.ExpectQuery(`INSERT`).WillReturnError(sql.ErrConnDone)
	err := repo.Create(context.Background(), &entity.Movie{Id: "m1"}, []string{"g1"})
	assert.Error(t, err)
}

// P3_092: movieRepository.Update() - Success
func TestMovieRepo_P3_092_Update_Success(t *testing.T) {
	db, mock := setupMovieDB(t)
	repo := newMovieRepo(db)
	mock.ExpectExec(`UPDATE`).WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.Update(context.Background(), &entity.Movie{Id: "m1", Title: "U"}, nil)
	assert.NoError(t, err)
}

// P3_093: movieRepository.Update() - DBError
func TestMovieRepo_P3_093_Update_DBError(t *testing.T) {
	db, mock := setupMovieDB(t)
	repo := newMovieRepo(db)
	mock.ExpectExec(`UPDATE`).WillReturnError(sql.ErrTxDone)
	err := repo.Update(context.Background(), &entity.Movie{Id: "m1"}, nil)
	assert.Error(t, err)
}

// P3_094: movieRepository.Delete() - HardDelete
func TestMovieRepo_P3_094_Delete_HardDelete(t *testing.T) {
	db, mock := setupMovieDB(t)
	repo := newMovieRepo(db)
	mock.ExpectExec(`DELETE FROM.*movies.*WHERE.*id`).WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.Delete(context.Background(), "m1")
	assert.NoError(t, err)
}
