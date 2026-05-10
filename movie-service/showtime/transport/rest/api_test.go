package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"movie-service/internal/module/showtime/business"
	"movie-service/internal/module/showtime/entity"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockShowtimeBiz struct {
	getShowtimeByIdFn      func(context.Context, string) (*entity.Showtime, error)
	getShowtimesFn         func(context.Context, int, int, string, string, string, entity.ShowtimeFormat, entity.ShowtimeStatus, *time.Time, *time.Time, bool) ([]*entity.Showtime, int, error)
	getUpcomingShowtimesFn func(context.Context, int) ([]*entity.Showtime, error)
	createShowtimeFn       func(context.Context, *entity.Showtime) error
	updateShowtimeFn       func(context.Context, string, *entity.UpdateShowtimeRequest) error
	deleteShowtimeFn       func(context.Context, string) error
	updateShowtimeStatusFn func(context.Context, string, entity.ShowtimeStatus) error
}

func (m *mockShowtimeBiz) GetShowtimeById(ctx context.Context, id string) (*entity.Showtime, error) {
	if m.getShowtimeByIdFn != nil {
		return m.getShowtimeByIdFn(ctx, id)
	}
	return nil, nil
}
func (m *mockShowtimeBiz) GetShowtimesByIds(ctx context.Context, ids []string) ([]*entity.Showtime, error) { return nil, nil }
func (m *mockShowtimeBiz) GetShowtimes(ctx context.Context, page, size int, search, movieId, roomId string, format entity.ShowtimeFormat, status entity.ShowtimeStatus, dateFrom, dateTo *time.Time, excludeEnded bool) ([]*entity.Showtime, int, error) {
	if m.getShowtimesFn != nil {
		return m.getShowtimesFn(ctx, page, size, search, movieId, roomId, format, status, dateFrom, dateTo, excludeEnded)
	}
	return nil, 0, nil
}
func (m *mockShowtimeBiz) GetUpcomingShowtimes(ctx context.Context, limit int) ([]*entity.Showtime, error) {
	if m.getUpcomingShowtimesFn != nil {
		return m.getUpcomingShowtimesFn(ctx, limit)
	}
	return nil, nil
}
func (m *mockShowtimeBiz) CreateShowtime(ctx context.Context, showtime *entity.Showtime) error {
	if m.createShowtimeFn != nil {
		return m.createShowtimeFn(ctx, showtime)
	}
	return nil
}
func (m *mockShowtimeBiz) UpdateShowtime(ctx context.Context, id string, updates *entity.UpdateShowtimeRequest) error {
	if m.updateShowtimeFn != nil {
		return m.updateShowtimeFn(ctx, id, updates)
	}
	return nil
}
func (m *mockShowtimeBiz) DeleteShowtime(ctx context.Context, id string) error {
	if m.deleteShowtimeFn != nil {
		return m.deleteShowtimeFn(ctx, id)
	}
	return nil
}
func (m *mockShowtimeBiz) UpdateShowtimeStatus(ctx context.Context, id string, status entity.ShowtimeStatus) error {
	if m.updateShowtimeStatusFn != nil {
		return m.updateShowtimeStatusFn(ctx, id, status)
	}
	return nil
}
func (m *mockShowtimeBiz) CheckTimeConflict(ctx context.Context, roomId string, startTime, endTime time.Time, excludeId string) (bool, error) { return false, nil }

func setupShowtimeRouter(biz business.ShowtimeBiz) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &handler{biz: biz}
	showtimes := r.Group("/showtimes")
	{
		showtimes.GET("", h.GetShowtimes)
		showtimes.POST("", h.CreateShowtime)
		showtimes.GET("/:id", h.GetShowtimeById)
		showtimes.PUT("/:id", h.UpdateShowtime)
		showtimes.DELETE("/:id", h.DeleteShowtime)
		showtimes.PATCH("/:id/status", h.UpdateShowtimeStatus)
		showtimes.GET("/upcoming", h.GetUpcomingShowtimes)
	}
	return r
}

// ---------------------------------------------------------------------------
// Tests P3_134 - P3_145 (Showtime REST Transport)
// ---------------------------------------------------------------------------

// P3_134: showtimeHandler.GetShowtimes() - WithFilter
func TestShowtimeHandler_P3_134_GetShowtimes_WithFilter(t *testing.T) {
	mockBiz := &mockShowtimeBiz{
		getShowtimesFn: func(_ context.Context, _, _ int, _, movieId, _ string, _ entity.ShowtimeFormat, _ entity.ShowtimeStatus, _, _ *time.Time, _ bool) ([]*entity.Showtime, int, error) {
			assert.Equal(t, "m1", movieId)
			return []*entity.Showtime{{Id: "st1"}}, 1, nil
		},
	}
	r := setupShowtimeRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/showtimes?movie_id=m1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Data   []entity.ShowtimeResponse `json:"data"`
			Paging interface{}              `json:"paging"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Data.Data)
	assert.Equal(t, "st1", resp.Data.Data[0].Id)
}

// P3_135: showtimeHandler.GetUpcoming() - Success
func TestShowtimeHandler_P3_135_GetUpcoming_Success(t *testing.T) {
	mockBiz := &mockShowtimeBiz{
		getUpcomingShowtimesFn: func(_ context.Context, limit int) ([]*entity.Showtime, error) {
			assert.Equal(t, 5, limit)
			res := make([]*entity.Showtime, 5)
			for i := range res {
				res[i] = &entity.Showtime{Id: "st"}
			}
			return res, nil
		},
	}
	r := setupShowtimeRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/showtimes/upcoming?limit=5", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                      `json:"success"`
		Data    []entity.ShowtimeResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Len(t, resp.Data, 5)
}

// P3_136: showtimeHandler.GetShowtimeById() - Success
func TestShowtimeHandler_P3_136_GetShowtimeById_Success(t *testing.T) {
	mockBiz := &mockShowtimeBiz{
		getShowtimeByIdFn: func(_ context.Context, id string) (*entity.Showtime, error) {
			return &entity.Showtime{Id: id}, nil
		},
	}
	r := setupShowtimeRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/showtimes/st1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                    `json:"success"`
		Data    entity.ShowtimeResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "st1", resp.Data.Id)
}

// P3_137: showtimeHandler.GetShowtimes() - Multi-Filter
func TestShowtimeHandler_P3_137_GetShowtimes_MultiFilter(t *testing.T) {
	mockBiz := &mockShowtimeBiz{
		getShowtimesFn: func(_ context.Context, _, _ int, _, _, roomId string, format entity.ShowtimeFormat, _ entity.ShowtimeStatus, _, _ *time.Time, _ bool) ([]*entity.Showtime, int, error) {
			assert.Equal(t, "r1", roomId)
			assert.Equal(t, entity.ShowtimeFormat2D, format)
			return []*entity.Showtime{{Id: "st1"}}, 1, nil
		},
	}
	r := setupShowtimeRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/showtimes?room_id=r1&format=2D", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// P3_138: showtimeHandler.CreateShowtime() - Success
func TestShowtimeHandler_P3_138_CreateShowtime_Success(t *testing.T) {
	mockBiz := &mockShowtimeBiz{
		createShowtimeFn: func(_ context.Context, _ *entity.Showtime) error { return nil },
	}
	r := setupShowtimeRouter(mockBiz)
	body := `{"movie_id":"m1","room_id":"r1","start_time":"2026-05-09T10:00:00Z","end_time":"2026-05-09T12:00:00Z","format":"2D","base_price":100}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/showtimes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// P3_139: showtimeHandler.CreateShowtime() - TimeConflict
func TestShowtimeHandler_P3_139_CreateShowtime_TimeConflict(t *testing.T) {
	mockBiz := &mockShowtimeBiz{
		createShowtimeFn: func(_ context.Context, _ *entity.Showtime) error { return business.ErrTimeConflict },
	}
	r := setupShowtimeRouter(mockBiz)
	body := `{"movie_id":"m1","room_id":"r1","start_time":"2026-05-09T10:00:00Z","end_time":"2026-05-09T12:00:00Z","format":"2D","base_price":100}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/showtimes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	// Expect 409 Conflict. Spec says currently 400.
	if w.Code == http.StatusConflict {
		var resp struct { Success bool `json:"success"` }
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.False(t, resp.Success)
	} else {
		assert.Equal(t, http.StatusConflict, w.Code, "Spec requires 409 Conflict on time overlap")
	}
}

// P3_140: showtimeHandler.CreateShowtime() - BindFail
func TestShowtimeHandler_P3_140_CreateShowtime_BindFail(t *testing.T) {
	r := setupShowtimeRouter(&mockShowtimeBiz{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/showtimes", bytes.NewBufferString(`{"movie_id": "m1",}`)) // trailing comma
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
}

// P3_141: showtimeHandler.CreateShowtime() - ValidationFail
func TestShowtimeHandler_P3_141_CreateShowtime_ValidationFail(t *testing.T) {
	mockBiz := &mockShowtimeBiz{
		createShowtimeFn: func(_ context.Context, _ *entity.Showtime) error { return business.ErrInvalidShowtimeData },
	}
	r := setupShowtimeRouter(mockBiz)
	body := `{"movie_id":"m1","room_id":"r1","base_price":-100}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/showtimes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
}

// P3_142: showtimeHandler.UpdateShowtime() - Success
func TestShowtimeHandler_P3_142_UpdateShowtime_Success(t *testing.T) {
	mockBiz := &mockShowtimeBiz{
		updateShowtimeFn: func(_ context.Context, _ string, _ *entity.UpdateShowtimeRequest) error { return nil },
	}
	r := setupShowtimeRouter(mockBiz)
	price := 60000.0
	body, _ := json.Marshal(entity.UpdateShowtimeRequest{BasePrice: &price})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/showtimes/st1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// P3_143: showtimeHandler.UpdateShowtime() - NotFound
func TestShowtimeHandler_P3_143_UpdateShowtime_NotFound(t *testing.T) {
	mockBiz := &mockShowtimeBiz{
		updateShowtimeFn: func(_ context.Context, _ string, _ *entity.UpdateShowtimeRequest) error { return business.ErrShowtimeNotFound },
	}
	r := setupShowtimeRouter(mockBiz)
	price := 50.0
	body, _ := json.Marshal(entity.UpdateShowtimeRequest{BasePrice: &price})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/showtimes/not-exist", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
}

// P3_144: showtimeHandler.DeleteShowtime() - Success
func TestShowtimeHandler_P3_144_DeleteShowtime_Success(t *testing.T) {
	mockBiz := &mockShowtimeBiz{
		deleteShowtimeFn: func(_ context.Context, _ string) error { return nil },
	}
	r := setupShowtimeRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/showtimes/st1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// P3_145: showtimeHandler.UpdateShowtimeStatus() - Success
func TestShowtimeHandler_P3_145_UpdateShowtimeStatus_Success(t *testing.T) {
	mockBiz := &mockShowtimeBiz{
		updateShowtimeStatusFn: func(_ context.Context, _ string, _ entity.ShowtimeStatus) error { return nil },
	}
	r := setupShowtimeRouter(mockBiz)
	body, _ := json.Marshal(map[string]string{"status": "ACTIVE"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/showtimes/st1/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}
