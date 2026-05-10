package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"movie-service/internal/module/news/business"
	"movie-service/internal/module/news/entity"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var ErrNewsNotFound = errors.New("news not found")

type mockNewsBiz struct {
	getNewsSummariesFn     func(context.Context, string, int, int, bool) ([]*entity.NewsSummaryWithSources, int, error)
	getNewsSummaryByIDFn   func(context.Context, string) (*entity.NewsSummaryWithSources, error)
	updateNewsSummaryFn    func(context.Context, string, string, string) error
	toggleNewsSummaryActiveFn func(context.Context, string, bool) error
}

func (m *mockNewsBiz) GetNewsSummaries(ctx context.Context, category string, page, pageSize int, includeInactive bool) ([]*entity.NewsSummaryWithSources, int, error) {
	if m.getNewsSummariesFn != nil {
		return m.getNewsSummariesFn(ctx, category, page, pageSize, includeInactive)
	}
	return nil, 0, nil
}
func (m *mockNewsBiz) GetNewsSummaryByID(ctx context.Context, id string) (*entity.NewsSummaryWithSources, error) {
	if m.getNewsSummaryByIDFn != nil {
		return m.getNewsSummaryByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockNewsBiz) UpdateNewsSummary(ctx context.Context, id, title, summary string) error {
	if m.updateNewsSummaryFn != nil {
		return m.updateNewsSummaryFn(ctx, id, title, summary)
	}
	return nil
}
func (m *mockNewsBiz) ToggleNewsSummaryActive(ctx context.Context, id string, isActive bool) error {
	if m.toggleNewsSummaryActiveFn != nil {
		return m.toggleNewsSummaryActiveFn(ctx, id, isActive)
	}
	return nil
}

func setupNewsRouter(biz business.NewsBusiness) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	a := &API{biz: biz}
	news := r.Group("/news")
	{
		news.GET("", a.GetNewsSummaries)
		news.GET("/:id", a.GetNewsSummaryByID)
		news.PUT("/:id", a.UpdateNewsSummary)
		news.PATCH("/:id/active", a.ToggleNewsSummaryActive)
	}
	return r
}

// ---------------------------------------------------------------------------
// Tests P3_164 - P3_169 (News REST Transport)
// ---------------------------------------------------------------------------

// P3_164: newsHandler.GetSummaries() - Success
func TestNewsHandler_P3_164_GetSummaries_Success(t *testing.T) {
	mockBiz := &mockNewsBiz{
		getNewsSummariesFn: func(_ context.Context, _ string, _, _ int, _ bool) ([]*entity.NewsSummaryWithSources, int, error) {
			return []*entity.NewsSummaryWithSources{{NewsSummary: entity.NewsSummary{ID: "n1"}}}, 1, nil
		},
	}
	r := setupNewsRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/news?page=1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data       []entity.NewsSummaryWithSources `json:"data"`
		Pagination interface{}                    `json:"pagination"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEmpty(t, resp.Data)
	assert.Equal(t, "n1", resp.Data[0].NewsSummary.ID)
}

// P3_165: newsHandler.GetSummaries() - NoNewsFound
func TestNewsHandler_P3_165_GetSummaries_NoNewsFound(t *testing.T) {
	mockBiz := &mockNewsBiz{
		getNewsSummariesFn: func(_ context.Context, cat string, _, _ int, _ bool) ([]*entity.NewsSummaryWithSources, int, error) {
			assert.Equal(t, "empty", cat)
			return []*entity.NewsSummaryWithSources{}, 0, nil
		},
	}
	r := setupNewsRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/news?category=empty", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data []entity.NewsSummaryWithSources `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Empty(t, resp.Data)
}

// P3_166: newsHandler.GetSummaryByID() - Success
func TestNewsHandler_P3_166_GetSummaryByID_Success(t *testing.T) {
	mockBiz := &mockNewsBiz{
		getNewsSummaryByIDFn: func(_ context.Context, id string) (*entity.NewsSummaryWithSources, error) {
			return &entity.NewsSummaryWithSources{NewsSummary: entity.NewsSummary{ID: id}}, nil
		},
	}
	r := setupNewsRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/news/n1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Data entity.NewsSummaryWithSources `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "n1", resp.Data.NewsSummary.ID)
}

// P3_167: newsHandler.GetSummaryByID() - NotFound
func TestNewsHandler_P3_167_GetSummaryByID_NotFound(t *testing.T) {
	mockBiz := &mockNewsBiz{
		getNewsSummaryByIDFn: func(_ context.Context, _ string) (*entity.NewsSummaryWithSources, error) {
			return nil, ErrNewsNotFound
		},
	}
	r := setupNewsRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/news/bad", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// P3_168: newsHandler.UpdateSummary() - Success
func TestNewsHandler_P3_168_UpdateSummary_Success(t *testing.T) {
	mockBiz := &mockNewsBiz{
		updateNewsSummaryFn: func(_ context.Context, _, _, _ string) error { return nil },
	}
	r := setupNewsRouter(mockBiz)
	body, _ := json.Marshal(map[string]string{"title": "New Title", "summary": "New Summary"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/news/n1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Message string `json:"message"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "News updated successfully", resp.Message)
}

// P3_169: newsHandler.ToggleActive() - Success
func TestNewsHandler_P3_169_ToggleActive_Success(t *testing.T) {
	mockBiz := &mockNewsBiz{
		toggleNewsSummaryActiveFn: func(_ context.Context, _ string, _ bool) error { return nil },
	}
	r := setupNewsRouter(mockBiz)
	body, _ := json.Marshal(map[string]bool{"is_active": true})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/news/n1/active", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Message string `json:"message"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "News status updated successfully", resp.Message)
}
