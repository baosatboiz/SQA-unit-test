package repository

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/stretchr/testify/assert"
)

func setupNewsDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { sqldb.Close() })
	return bun.NewDB(sqldb, pgdialect.New()), mock
}

// ---------------------------------------------------------------------------
// Tests P3_116 - P3_121 (News Repository)
// ---------------------------------------------------------------------------

// P3_116: newsRepository.GetNewsSummaries() - WithCategory
func TestNewsRepo_P3_116_GetNewsSummaries_WithCategory(t *testing.T) {
	db, mock := setupNewsDB(t)
	repo := NewNewsRepository(db)

	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id", "category"}).AddRow("n1", "movie"))
	res, err := repo.GetNewsSummaries(context.Background(), "movie", 10, 0, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, res)
}

// P3_117: newsRepository.GetNewsSummaryByID() - Found
func TestNewsRepo_P3_117_GetNewsSummaryByID_Found(t *testing.T) {
	db, mock := setupNewsDB(t)
	repo := NewNewsRepository(db)

	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow("n1", "Title"))
	res, err := repo.GetNewsSummaryByID(context.Background(), "n1")
	assert.NoError(t, err)
	assert.NotNil(t, res)
}

// P3_118: newsRepository.GetArticlesByIDs() - Batch
func TestNewsRepo_P3_118_GetArticlesByIDs_Batch(t *testing.T) {
	db, mock := setupNewsDB(t)
	repo := NewNewsRepository(db)

	mock.ExpectQuery(`SELECT`).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow("a1", "A1").AddRow("a2", "A2"))
	res, err := repo.GetArticlesByIDs(context.Background(), []string{"a1", "a2"})
	assert.NoError(t, err)
	assert.Len(t, res, 2)
}

// P3_119: newsRepository.CountSummaries() - WithFilter
func TestNewsRepo_P3_119_CountSummaries_WithFilter(t *testing.T) {
	db, mock := setupNewsDB(t)
	repo := NewNewsRepository(db)

	mock.ExpectQuery(`(?i)SELECT.*count`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	count, err := repo.CountSummaries(context.Background(), "movie", false)
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}

// P3_120: newsRepository.UpdateNewsSummary() - Success
func TestNewsRepo_P3_120_UpdateNewsSummary_Success(t *testing.T) {
	db, mock := setupNewsDB(t)
	repo := NewNewsRepository(db)

	mock.ExpectExec(`UPDATE`).WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.UpdateNewsSummary(context.Background(), "n1", "T", "S")
	assert.NoError(t, err)
}

// P3_121: newsRepository.UpdateNewsSummaryIsActive() - Toggle
func TestNewsRepo_P3_121_UpdateNewsSummaryIsActive_Toggle(t *testing.T) {
	db, mock := setupNewsDB(t)
	repo := NewNewsRepository(db)

	mock.ExpectExec(`UPDATE`).WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.UpdateNewsSummaryIsActive(context.Background(), "n1", true)
	assert.NoError(t, err)
}
