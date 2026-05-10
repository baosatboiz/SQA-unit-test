package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"movie-service/internal/module/movie/business"
	"movie-service/internal/module/movie/entity"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var errConflict = errors.New("conflict")

type mockMovieBiz struct {
	getMovieByIdFn      func(context.Context, string) (*entity.Movie, error)
	getMoviesFn         func(context.Context, int, int, string, string) ([]*entity.Movie, int, error)
	getMovieStatsFn     func(context.Context) ([]*entity.MovieStat, error)
	getGenresFn         func(context.Context) ([]*entity.Genre, error)
	createMovieFn       func(context.Context, *entity.Movie, []string) error
	updateMovieFn       func(context.Context, *entity.Movie, []string) error
	deleteMovieFn       func(context.Context, string) error
	updateMovieStatusFn func(context.Context, string, entity.MovieStatus) error
	validateMovieForShowtimeFn func(context.Context, string) error
}

func (m *mockMovieBiz) GetMovieById(ctx context.Context, id string) (*entity.Movie, error) {
	if m.getMovieByIdFn != nil {
		return m.getMovieByIdFn(ctx, id)
	}
	return &entity.Movie{Id: id, Title: "T"}, nil
}
func (m *mockMovieBiz) GetMovies(ctx context.Context, page, size int, search, status string) ([]*entity.Movie, int, error) {
	if m.getMoviesFn != nil {
		return m.getMoviesFn(ctx, page, size, search, status)
	}
	return nil, 0, nil
}
func (m *mockMovieBiz) GetMovieStats(ctx context.Context) ([]*entity.MovieStat, error) {
	if m.getMovieStatsFn != nil {
		return m.getMovieStatsFn(ctx)
	}
	return nil, nil
}
func (m *mockMovieBiz) GetGenres(ctx context.Context) ([]*entity.Genre, error) {
	if m.getGenresFn != nil {
		return m.getGenresFn(ctx)
	}
	return nil, nil
}
func (m *mockMovieBiz) CreateMovie(ctx context.Context, movie *entity.Movie, genreIds []string) error {
	if m.createMovieFn != nil {
		return m.createMovieFn(ctx, movie, genreIds)
	}
	return nil
}
func (m *mockMovieBiz) UpdateMovie(ctx context.Context, movie *entity.Movie, genreIds []string) error {
	if m.updateMovieFn != nil {
		return m.updateMovieFn(ctx, movie, genreIds)
	}
	return nil
}
func (m *mockMovieBiz) DeleteMovie(ctx context.Context, id string) error {
	if m.deleteMovieFn != nil {
		return m.deleteMovieFn(ctx, id)
	}
	return nil
}
func (m *mockMovieBiz) UpdateMovieStatus(ctx context.Context, id string, status entity.MovieStatus) error {
	if m.updateMovieStatusFn != nil {
		return m.updateMovieStatusFn(ctx, id, status)
	}
	return nil
}
func (m *mockMovieBiz) ValidateMovieForShowtime(ctx context.Context, id string) error {
	if m.validateMovieForShowtimeFn != nil {
		return m.validateMovieForShowtimeFn(ctx, id)
	}
	return nil
}

func setupRouter(biz business.MovieBiz) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &handler{biz: biz}
	movies := r.Group("/movies")
	{
		movies.GET("", h.GetMovies)
		movies.POST("", h.CreateMovie)
		movies.GET("/stats", h.GetMovieStats)
		movies.GET("/:id", h.GetMovieById)
		movies.PUT("/:id", h.UpdateMovie)
		movies.DELETE("/:id", h.DeleteMovie)
		movies.PATCH("/:id/status", h.UpdateMovieStatus)
	}
	r.GET("/genres", h.GetGenres)
	return r
}

// ---------------------------------------------------------------------------
// Tests P3_122 - P3_133 (Movie REST Transport)
// ---------------------------------------------------------------------------

// P3_122: handler.GetMovies() - Success
func TestMovieHandler_P3_122_GetMovies_Success(t *testing.T) {
	mockBiz := &mockMovieBiz{
		getMoviesFn: func(_ context.Context, _, _ int, _, _ string) ([]*entity.Movie, int, error) {
			return []*entity.Movie{{Id: "m1", Title: "T"}}, 1, nil
		},
	}
	r := setupRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/movies?page=1&size=10", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Movies []entity.MovieResponse `json:"movies"`
			Meta   struct {
				Total int `json:"total"`
			} `json:"meta"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, 1, resp.Data.Meta.Total)
	assert.NotEmpty(t, resp.Data.Movies)
	assert.Equal(t, "m1", resp.Data.Movies[0].Id)
}

// P3_123: handler.GetMovies() - InvalidPage fallback
func TestMovieHandler_P3_123_GetMovies_InvalidPage(t *testing.T) {
	mockBiz := &mockMovieBiz{
		getMoviesFn: func(_ context.Context, page, _ int, _, _ string) ([]*entity.Movie, int, error) {
			assert.Equal(t, 1, page)
			return []*entity.Movie{}, 0, nil
		},
	}
	r := setupRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/movies?page=abc", nil)
	r.ServeHTTP(w, req)
	// Expect FAIL: current code returns 400 instead of fallback 200
	if w.Code == http.StatusOK {
		var resp struct { Success bool `json:"success"` }
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.True(t, resp.Success)
	} else {
		assert.Equal(t, http.StatusOK, w.Code, "Spec requires fallback to page 1 and HTTP 200")
	}
}

// P3_124: handler.GetMovieById() - Success
func TestMovieHandler_P3_124_GetMovieById_Success(t *testing.T) {
	mockBiz := &mockMovieBiz{
		getMovieByIdFn: func(_ context.Context, id string) (*entity.Movie, error) {
			return &entity.Movie{Id: id, Title: "T"}, nil
		},
	}
	r := setupRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/movies/m1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                 `json:"success"`
		Data    entity.MovieResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "m1", resp.Data.Id)
}

// P3_125: handler.GetMovieById() - NotFound
func TestMovieHandler_P3_125_GetMovieById_NotFound(t *testing.T) {
	mockBiz := &mockMovieBiz{
		getMovieByIdFn: func(_ context.Context, _ string) (*entity.Movie, error) {
			return nil, business.ErrMovieNotFound
		},
	}
	r := setupRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/movies/bad", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
}

// P3_126: handler.CreateMovie() - Success
func TestMovieHandler_P3_126_CreateMovie_Success(t *testing.T) {
	mockBiz := &mockMovieBiz{
		createMovieFn: func(_ context.Context, _ *entity.Movie, _ []string) error { return nil },
	}
	r := setupRouter(mockBiz)
	body, _ := json.Marshal(entity.CreateMovieRequest{Title: "New", Genres: []string{"g1"}, Duration: 120})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/movies", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Success bool                 `json:"success"`
		Data    entity.MovieResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "New", resp.Data.Title)
}

// P3_127: handler.CreateMovie() - ValidationFail
func TestMovieHandler_P3_127_CreateMovie_ValidationFail(t *testing.T) {
	r := setupRouter(&mockMovieBiz{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/movies", bytes.NewBufferString("{invalid}"))
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
}

// P3_128: handler.UpdateMovie() - Success
func TestMovieHandler_P3_128_UpdateMovie_Success(t *testing.T) {
	mockBiz := &mockMovieBiz{
		updateMovieFn: func(_ context.Context, _ *entity.Movie, _ []string) error { return nil },
		getMovieByIdFn: func(_ context.Context, id string) (*entity.Movie, error) {
			return &entity.Movie{Id: id, Title: "U"}, nil
		},
	}
	r := setupRouter(mockBiz)
	body, _ := json.Marshal(entity.UpdateMovieRequest{Title: "U", Duration: 120})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/movies/m1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                 `json:"success"`
		Data    entity.MovieResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "U", resp.Data.Title)
}

// P3_129: handler.DeleteMovie() - Success
func TestMovieHandler_P3_129_DeleteMovie_Success(t *testing.T) {
	mockBiz := &mockMovieBiz{
		deleteMovieFn: func(_ context.Context, _ string) error { return nil },
	}
	r := setupRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/movies/m1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// P3_130: handler.DeleteMovie() - HasShowtimes (Conflict)
func TestMovieHandler_P3_130_DeleteMovie_HasShowtimes(t *testing.T) {
	mockBiz := &mockMovieBiz{
		deleteMovieFn: func(_ context.Context, _ string) error { return errConflict },
	}
	r := setupRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/movies/m1", nil)
	r.ServeHTTP(w, req)
	// Expect 409 Conflict. Spec says currently 500.
	if w.Code != http.StatusConflict {
		assert.Equal(t, http.StatusConflict, w.Code, "Spec requires 409 Conflict when movie has showtimes")
	} else {
		var resp struct { Success bool `json:"success"` }
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.False(t, resp.Success)
	}
}

// P3_131: handler.UpdateMovieStatus() - Success
func TestMovieHandler_P3_131_UpdateMovieStatus_Success(t *testing.T) {
	mockBiz := &mockMovieBiz{
		updateMovieStatusFn: func(_ context.Context, _ string, _ entity.MovieStatus) error { return nil },
		getMovieByIdFn: func(_ context.Context, id string) (*entity.Movie, error) {
			return &entity.Movie{Id: id, Title: "T"}, nil
		},
	}
	r := setupRouter(mockBiz)
	body, _ := json.Marshal(map[string]string{"status": "showing"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/movies/m1/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                 `json:"success"`
		Data    entity.MovieResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// P3_132: handler.GetGenres() - Success
func TestMovieHandler_P3_132_GetGenres_Success(t *testing.T) {
	mockBiz := &mockMovieBiz{
		getGenresFn: func(_ context.Context) ([]*entity.Genre, error) {
			return []*entity.Genre{{Id: "g1", Name: "Action"}}, nil
		},
	}
	r := setupRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/genres", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                     `json:"success"`
		Data    []map[string]interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Data)
}

// P3_133: handler.GetMovieStats() - Success
func TestMovieHandler_P3_133_GetMovieStats_Success(t *testing.T) {
	mockBiz := &mockMovieBiz{
		getMovieStatsFn: func(_ context.Context) ([]*entity.MovieStat, error) {
			return []*entity.MovieStat{{Status: "SHOWING", Count: 5}}, nil
		},
	}
	r := setupRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/movies/stats", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool               `json:"success"`
		Data    []entity.MovieStat `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, int64(5), resp.Data[0].Count)
}
