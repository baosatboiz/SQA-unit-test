package business

import (
	"context"
	"errors"
	"testing"

	"movie-service/internal/module/news/entity"
	"movie-service/internal/module/news/repository"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// mockNewsRepository — implements repository.NewsRepository in-memory
// ---------------------------------------------------------------------------

type mockNewsRepository struct {
	getNewsSummariesFn          func(context.Context, string, int, int, bool) ([]*entity.NewsSummary, error)
	getNewsSummaryByIDFn        func(context.Context, string) (*entity.NewsSummary, error)
	getArticlesByIDsFn          func(context.Context, []string) ([]*entity.NewsArticle, error)
	countSummariesFn            func(context.Context, string, bool) (int, error)
	updateNewsSummaryFn         func(context.Context, string, string, string) error
	updateNewsSummaryIsActiveFn func(context.Context, string, bool) error
}

// compile-time check: mockNewsRepository implements repository.NewsRepository
var _ repository.NewsRepository = (*mockNewsRepository)(nil)

func (m *mockNewsRepository) GetNewsSummaries(ctx context.Context, category string, limit, offset int, includeInactive bool) ([]*entity.NewsSummary, error) {
	if m.getNewsSummariesFn != nil {
		return m.getNewsSummariesFn(ctx, category, limit, offset, includeInactive)
	}
	return nil, nil
}

func (m *mockNewsRepository) GetNewsSummaryByID(ctx context.Context, id string) (*entity.NewsSummary, error) {
	if m.getNewsSummaryByIDFn != nil {
		return m.getNewsSummaryByIDFn(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockNewsRepository) GetArticlesByIDs(ctx context.Context, ids []string) ([]*entity.NewsArticle, error) {
	if m.getArticlesByIDsFn != nil {
		return m.getArticlesByIDsFn(ctx, ids)
	}
	return nil, nil
}

func (m *mockNewsRepository) CountSummaries(ctx context.Context, category string, includeInactive bool) (int, error) {
	if m.countSummariesFn != nil {
		return m.countSummariesFn(ctx, category, includeInactive)
	}
	return 0, nil
}

func (m *mockNewsRepository) UpdateNewsSummary(ctx context.Context, id, title, summary string) error {
	if m.updateNewsSummaryFn != nil {
		return m.updateNewsSummaryFn(ctx, id, title, summary)
	}
	return nil
}

func (m *mockNewsRepository) UpdateNewsSummaryIsActive(ctx context.Context, id string, isActive bool) error {
	if m.updateNewsSummaryIsActiveFn != nil {
		return m.updateNewsSummaryIsActiveFn(ctx, id, isActive)
	}
	return nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTestNewsBusiness(repo repository.NewsRepository) NewsBusiness {
	return NewNewsBusiness(repo)
}

func makeValidNewsSummary(id string, isActive bool) *entity.NewsSummary {
	return &entity.NewsSummary{
		ID:       id,
		Title:    "Test News Title",
		Summary:  "Test news summary content",
		IsActive: isActive,
		Category: "technology",
	}
}

// ---------------------------------------------------------------------------
// Tests P3_075 - P3_084 (News Business)
// ---------------------------------------------------------------------------

// P3_075: GetNewsSummaries - Success
func TestNewsBusiness_P3_075_GetNewsSummaries_Success(t *testing.T) {
	repo := &mockNewsRepository{
		getNewsSummariesFn: func(_ context.Context, _ string, _, _ int, _ bool) ([]*entity.NewsSummary, error) {
			return []*entity.NewsSummary{makeValidNewsSummary("n1", true), makeValidNewsSummary("n2", true)}, nil
		},
		countSummariesFn: func(_ context.Context, _ string, _ bool) (int, error) { return 2, nil },
	}
	biz := newTestNewsBusiness(repo)
	summaries, total, err := biz.GetNewsSummaries(context.Background(), "all", 1, 10, false)
	assert.NoError(t, err)
	assert.Len(t, summaries, 2)
	assert.Equal(t, 2, total)
}

// P3_076: GetNewsSummaries - IncludeInactive
func TestNewsBusiness_P3_076_GetNewsSummaries_IncludeInactive(t *testing.T) {
	repo := &mockNewsRepository{
		getNewsSummariesFn: func(_ context.Context, _ string, _, _ int, includeInactive bool) ([]*entity.NewsSummary, error) {
			if includeInactive {
				return []*entity.NewsSummary{makeValidNewsSummary("n1", true), makeValidNewsSummary("n2", false)}, nil
			}
			return []*entity.NewsSummary{makeValidNewsSummary("n1", true)}, nil
		},
	}
	biz := newTestNewsBusiness(repo)
	summaries, _, _ := biz.GetNewsSummaries(context.Background(), "all", 1, 10, true)
	assert.Len(t, summaries, 2)
}

// P3_077: GetNewsSummaries - RepoError
func TestNewsBusiness_P3_077_GetNewsSummaries_RepoError(t *testing.T) {
	repo := &mockNewsRepository{
		countSummariesFn: func(_ context.Context, _ string, _ bool) (int, error) {
			return 0, errors.New("failed to count")
		},
	}
	biz := newTestNewsBusiness(repo)
	_, _, err := biz.GetNewsSummaries(context.Background(), "all", 1, 10, false)
	assert.Error(t, err)
}

// P3_078: GetNewsSummaryByID - Success
func TestNewsBusiness_P3_078_GetNewsSummaryByID_Success(t *testing.T) {
	repo := &mockNewsRepository{
		getNewsSummaryByIDFn: func(_ context.Context, id string) (*entity.NewsSummary, error) {
			return makeValidNewsSummary(id, true), nil
		},
		getArticlesByIDsFn: func(_ context.Context, _ []string) ([]*entity.NewsArticle, error) {
			return []*entity.NewsArticle{{ID: "art-1"}}, nil
		},
	}
	biz := newTestNewsBusiness(repo)
	res, err := biz.GetNewsSummaryByID(context.Background(), "n1")
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "n1", res.ID)
}

// P3_079: GetNewsSummaryByID - NotFound
func TestNewsBusiness_P3_079_GetNewsSummaryByID_NotFound(t *testing.T) {
	repo := &mockNewsRepository{
		getNewsSummaryByIDFn: func(_ context.Context, _ string) (*entity.NewsSummary, error) {
			return nil, errors.New("not found")
		},
	}
	biz := newTestNewsBusiness(repo)
	_, err := biz.GetNewsSummaryByID(context.Background(), "bad")
	assert.Error(t, err)
}

// P3_080: UpdateNewsSummary - Success
func TestNewsBusiness_P3_080_UpdateNewsSummary_Success(t *testing.T) {
	repo := &mockNewsRepository{}
	biz := newTestNewsBusiness(repo)
	err := biz.UpdateNewsSummary(context.Background(), "n1", "New Title", "New Summary")
	assert.NoError(t, err)
}

// P3_081: UpdateNewsSummary - EmptyContent
func TestNewsBusiness_P3_081_UpdateNewsSummary_EmptyContent(t *testing.T) {
	biz := newTestNewsBusiness(&mockNewsRepository{})
	// Expecting FAIL as per spec: "Không validate input trước khi gọi repo"
	err := biz.UpdateNewsSummary(context.Background(), "n1", "", "S")
	assert.Error(t, err, "Expected validation error for empty title")
}

// P3_082: ToggleNewsSummaryActive - Activate
func TestNewsBusiness_P3_082_ToggleNewsSummaryActive_Activate(t *testing.T) {
	repo := &mockNewsRepository{}
	biz := newTestNewsBusiness(repo)
	err := biz.ToggleNewsSummaryActive(context.Background(), "n1", true)
	assert.NoError(t, err)
}

// P3_083: ToggleNewsSummaryActive - Deactivate
func TestNewsBusiness_P3_083_ToggleNewsSummaryActive_Deactivate(t *testing.T) {
	repo := &mockNewsRepository{}
	biz := newTestNewsBusiness(repo)
	err := biz.ToggleNewsSummaryActive(context.Background(), "n1", false)
	assert.NoError(t, err)
}

// P3_084: ToggleNewsSummaryActive - RepoError
func TestNewsBusiness_P3_084_ToggleNewsSummaryActive_RepoError(t *testing.T) {
	repo := &mockNewsRepository{
		updateNewsSummaryIsActiveFn: func(_ context.Context, _ string, _ bool) error {
			return errors.New("failed to toggle")
		},
	}
	biz := newTestNewsBusiness(repo)
	err := biz.ToggleNewsSummaryActive(context.Background(), "n1", true)
	assert.Error(t, err)
}
