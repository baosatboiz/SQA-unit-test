package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"movie-service/internal/module/seat/business"
	"movie-service/internal/module/seat/entity"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockSeatBiz struct {
	getSeatByIdFn                func(context.Context, string) (*entity.Seat, error)
	getSeatsFn                   func(context.Context, int, int, string, string, string, entity.SeatType, entity.SeatStatus) ([]*entity.Seat, int, error)
	getSeatsByIdsFn              func(context.Context, []string) ([]*entity.Seat, error)
	getLockedSeatsByShowtimeFn    func(context.Context, string) (*entity.LockedSeatsResponse, error)
	createSeatFn                 func(context.Context, *entity.Seat) error
	updateSeatFn                 func(context.Context, string, *entity.UpdateSeatRequest) error
	deleteSeatFn                 func(context.Context, string) error
	updateSeatStatusFn           func(context.Context, string, entity.SeatStatus) error
}

func (m *mockSeatBiz) GetSeatById(ctx context.Context, id string) (*entity.Seat, error) {
	if m.getSeatByIdFn != nil {
		return m.getSeatByIdFn(ctx, id)
	}
	return nil, nil
}
func (m *mockSeatBiz) GetSeats(ctx context.Context, page, size int, search, roomId, rowNumber string, seatType entity.SeatType, status entity.SeatStatus) ([]*entity.Seat, int, error) {
	if m.getSeatsFn != nil {
		return m.getSeatsFn(ctx, page, size, search, roomId, rowNumber, seatType, status)
	}
	return nil, 0, nil
}
func (m *mockSeatBiz) GetSeatsByIds(ctx context.Context, ids []string) ([]*entity.Seat, error) {
	if m.getSeatsByIdsFn != nil {
		return m.getSeatsByIdsFn(ctx, ids)
	}
	return nil, nil
}
func (m *mockSeatBiz) GetLockedSeatsByShowtime(ctx context.Context, showtimeId string) (*entity.LockedSeatsResponse, error) {
	if m.getLockedSeatsByShowtimeFn != nil {
		return m.getLockedSeatsByShowtimeFn(ctx, showtimeId)
	}
	return nil, nil
}
func (m *mockSeatBiz) CreateSeat(ctx context.Context, seat *entity.Seat) error {
	if m.createSeatFn != nil {
		return m.createSeatFn(ctx, seat)
	}
	return nil
}
func (m *mockSeatBiz) UpdateSeat(ctx context.Context, id string, updates *entity.UpdateSeatRequest) error {
	if m.updateSeatFn != nil {
		return m.updateSeatFn(ctx, id, updates)
	}
	return nil
}
func (m *mockSeatBiz) DeleteSeat(ctx context.Context, id string) error {
	if m.deleteSeatFn != nil {
		return m.deleteSeatFn(ctx, id)
	}
	return nil
}
func (m *mockSeatBiz) UpdateSeatStatus(ctx context.Context, id string, status entity.SeatStatus) error {
	if m.updateSeatStatusFn != nil {
		return m.updateSeatStatusFn(ctx, id, status)
	}
	return nil
}

func setupSeatRouter(biz business.SeatBiz) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &handler{biz: biz}
	seats := r.Group("/seats")
	{
		seats.GET("", h.GetSeats)
		seats.POST("", h.CreateSeat)
		seats.GET("/:id", h.GetSeatById)
		seats.PUT("/:id", h.UpdateSeat)
		seats.DELETE("/:id", h.DeleteSeat)
		seats.PATCH("/:id/status", h.UpdateSeatStatus)
		seats.GET("/locked", h.GetLockedSeats)
	}
	return r
}

// ---------------------------------------------------------------------------
// Tests P3_155 - P3_163 (Seat REST Transport)
// ---------------------------------------------------------------------------

// P3_155: seatHandler.GetByRoom() - Success
func TestSeatHandler_P3_155_GetByRoom_Success(t *testing.T) {
	mockBiz := &mockSeatBiz{
		getSeatsFn: func(_ context.Context, _, _ int, _, roomId, _ string, _ entity.SeatType, _ entity.SeatStatus) ([]*entity.Seat, int, error) {
			assert.Equal(t, "r1", roomId)
			return []*entity.Seat{{Id: "s1"}}, 1, nil
		},
	}
	r := setupSeatRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/seats?room_id=r1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Data   []entity.SeatResponse `json:"data"`
			Paging interface{}           `json:"paging"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "s1", resp.Data.Data[0].Id)
}

// P3_156: seatHandler.GetLockedSeats() - Success
func TestSeatHandler_P3_156_GetLockedSeats_Success(t *testing.T) {
	mockBiz := &mockSeatBiz{
		getLockedSeatsByShowtimeFn: func(_ context.Context, _ string) (*entity.LockedSeatsResponse, error) {
			return &entity.LockedSeatsResponse{LockedSeatIds: []string{"s1", "s2"}}, nil
		},
	}
	r := setupSeatRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/seats/locked?showtime_id=st1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                       `json:"success"`
		Data    entity.LockedSeatsResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Data.LockedSeatIds, "s1")
}

// P3_157: seatHandler.GetSeatById() - Success
func TestSeatHandler_P3_157_GetSeatById_Success(t *testing.T) {
	mockBiz := &mockSeatBiz{
		getSeatByIdFn: func(_ context.Context, id string) (*entity.Seat, error) {
			return &entity.Seat{Id: id}, nil
		},
	}
	r := setupSeatRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/seats/s1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                `json:"success"`
		Data    entity.SeatResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "s1", resp.Data.Id)
}

// P3_158: seatHandler.GetSeatById() - NotFound
func TestSeatHandler_P3_158_GetSeatById_NotFound(t *testing.T) {
	mockBiz := &mockSeatBiz{
		getSeatByIdFn: func(_ context.Context, _ string) (*entity.Seat, error) {
			return nil, business.ErrSeatNotFound
		},
	}
	r := setupSeatRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/seats/bad", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
}

// P3_159: seatHandler.CreateSeat() - Success
func TestSeatHandler_P3_159_CreateSeat_Success(t *testing.T) {
	mockBiz := &mockSeatBiz{
		createSeatFn: func(_ context.Context, _ *entity.Seat) error { return nil },
	}
	r := setupSeatRouter(mockBiz)
	body := `{"room_id":"r1","seat_number":"A1","row_number":"A","seat_type":"REGULAR"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/seats", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// P3_160: seatHandler.CreateSeat() - ValidationFail
func TestSeatHandler_P3_160_CreateSeat_ValidationFail(t *testing.T) {
	r := setupSeatRouter(&mockSeatBiz{})
	body := `{"room_id":"r1"}` // missing seat_number, row_number
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/seats", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
}

// P3_161: seatHandler.UpdateSeat() - Success
func TestSeatHandler_P3_161_UpdateSeat_Success(t *testing.T) {
	mockBiz := &mockSeatBiz{
		updateSeatFn: func(_ context.Context, _ string, _ *entity.UpdateSeatRequest) error { return nil },
	}
	r := setupSeatRouter(mockBiz)
	body := `{"seat_type":"VIP"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/seats/s1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// P3_162: seatHandler.DeleteSeat() - Success
func TestSeatHandler_P3_162_DeleteSeat_Success(t *testing.T) {
	mockBiz := &mockSeatBiz{
		deleteSeatFn: func(_ context.Context, _ string) error { return nil },
	}
	r := setupSeatRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/seats/s1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// P3_163: seatHandler.UpdateSeatStatus() - Success
func TestSeatHandler_P3_163_UpdateSeatStatus_Success(t *testing.T) {
	mockBiz := &mockSeatBiz{
		updateSeatStatusFn: func(_ context.Context, _ string, _ entity.SeatStatus) error { return nil },
	}
	r := setupSeatRouter(mockBiz)
	body := `{"status":"DISABLED"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/seats/s1/status", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}
