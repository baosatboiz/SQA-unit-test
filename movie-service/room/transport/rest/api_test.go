package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"movie-service/internal/module/room/business"
	"movie-service/internal/module/room/entity"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockRoomBiz struct {
	getRoomByIdFn        func(context.Context, string) (*entity.Room, error)
	getRoomsFn           func(context.Context, int, int, string, entity.RoomType, entity.RoomStatus) ([]*entity.Room, int, error)
	createRoomFn         func(context.Context, *entity.Room) error
	updateRoomFn         func(context.Context, string, *entity.UpdateRoomRequest) error
	deleteRoomFn         func(context.Context, string) error
	updateRoomStatusFn   func(context.Context, string, entity.RoomStatus) error
}

func (m *mockRoomBiz) GetRoomById(ctx context.Context, id string) (*entity.Room, error) {
	if m.getRoomByIdFn != nil {
		return m.getRoomByIdFn(ctx, id)
	}
	return nil, nil
}
func (m *mockRoomBiz) GetRooms(ctx context.Context, page, size int, search string, roomType entity.RoomType, status entity.RoomStatus) ([]*entity.Room, int, error) {
	if m.getRoomsFn != nil {
		return m.getRoomsFn(ctx, page, size, search, roomType, status)
	}
	return nil, 0, nil
}
func (m *mockRoomBiz) CreateRoom(ctx context.Context, room *entity.Room) error {
	if m.createRoomFn != nil {
		return m.createRoomFn(ctx, room)
	}
	return nil
}
func (m *mockRoomBiz) UpdateRoom(ctx context.Context, id string, updates *entity.UpdateRoomRequest) error {
	if m.updateRoomFn != nil {
		return m.updateRoomFn(ctx, id, updates)
	}
	return nil
}
func (m *mockRoomBiz) DeleteRoom(ctx context.Context, id string) error {
	if m.deleteRoomFn != nil {
		return m.deleteRoomFn(ctx, id)
	}
	return nil
}
func (m *mockRoomBiz) UpdateRoomStatus(ctx context.Context, id string, status entity.RoomStatus) error {
	if m.updateRoomStatusFn != nil {
		return m.updateRoomStatusFn(ctx, id, status)
	}
	return nil
}
func (m *mockRoomBiz) ValidateRoomForShowtime(ctx context.Context, roomId string) error { return nil }

func setupRoomRouter(biz business.RoomBiz) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &handler{biz: biz}
	rooms := r.Group("/rooms")
	{
		rooms.GET("", h.GetRooms)
		rooms.POST("", h.CreateRoom)
		rooms.GET("/:id", h.GetRoomById)
		rooms.PUT("/:id", h.UpdateRoom)
		rooms.DELETE("/:id", h.DeleteRoom)
		rooms.PATCH("/:id/status", h.UpdateRoomStatus)
	}
	return r
}

// ---------------------------------------------------------------------------
// Tests P3_146 - P3_150 (Room REST Transport)
// ---------------------------------------------------------------------------

// P3_146: roomHandler.GetRooms() - Success
func TestRoomHandler_P3_146_GetRooms_Success(t *testing.T) {
	mockBiz := &mockRoomBiz{
		getRoomsFn: func(_ context.Context, _, _ int, _ string, _ entity.RoomType, _ entity.RoomStatus) ([]*entity.Room, int, error) {
			return []*entity.Room{{Id: "r1"}}, 1, nil
		},
	}
	r := setupRoomRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/rooms", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Data   []entity.RoomResponse `json:"data"`
			Paging interface{}           `json:"paging"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "r1", resp.Data.Data[0].Id)
}

// P3_147: roomHandler.GetRoomById() - Success
func TestRoomHandler_P3_147_GetRoomById_Success(t *testing.T) {
	mockBiz := &mockRoomBiz{
		getRoomByIdFn: func(_ context.Context, id string) (*entity.Room, error) {
			return &entity.Room{Id: id}, nil
		},
	}
	r := setupRoomRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/rooms/r1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                `json:"success"`
		Data    entity.RoomResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "r1", resp.Data.Id)
}

// P3_148: roomHandler.GetRoomById() - NotFound
func TestRoomHandler_P3_148_GetRoomById_NotFound(t *testing.T) {
	mockBiz := &mockRoomBiz{
		getRoomByIdFn: func(_ context.Context, _ string) (*entity.Room, error) {
			return nil, business.ErrRoomNotFound
		},
	}
	r := setupRoomRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/rooms/r99", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
}

// P3_149: roomHandler.CreateRoom() - DuplicateName (Conflict)
func TestRoomHandler_P3_149_CreateRoom_DuplicateName(t *testing.T) {
	mockBiz := &mockRoomBiz{
		createRoomFn: func(_ context.Context, _ *entity.Room) error { return business.ErrRoomNumberExists },
	}
	r := setupRoomRouter(mockBiz)
	body, _ := json.Marshal(entity.CreateRoomRequest{RoomNumber: 101})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/rooms", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	// Expect 409 Conflict. Spec says currently 400.
	if w.Code == http.StatusConflict {
		var resp struct { Success bool `json:"success"` }
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.False(t, resp.Success)
	} else {
		assert.Equal(t, http.StatusConflict, w.Code, "Spec requires 409 Conflict on duplicate room number")
	}
}

// P3_150: roomHandler.CreateRoom() - BindFail
func TestRoomHandler_P3_150_CreateRoom_BindFail(t *testing.T) {
	r := setupRoomRouter(&mockRoomBiz{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/rooms", bytes.NewBufferString(`{"capacity": 50}`)) // missing room_number
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
}

// P3_151: roomHandler.CreateRoom() - ValidationFail (Negative Capacity)
func TestRoomHandler_P3_151_CreateRoom_ValidationFail(t *testing.T) {
	r := setupRoomRouter(&mockRoomBiz{})
	body, _ := json.Marshal(entity.CreateRoomRequest{RoomNumber: 102, Capacity: -10, RoomType: "STANDARD"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/rooms", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
}

// P3_152: roomHandler.UpdateRoom() - Success
func TestRoomHandler_P3_152_UpdateRoom_Success(t *testing.T) {
	mockBiz := &mockRoomBiz{
		updateRoomFn: func(_ context.Context, _ string, _ *entity.UpdateRoomRequest) error { return nil },
	}
	r := setupRoomRouter(mockBiz)
	capVal := 100
	body, _ := json.Marshal(entity.UpdateRoomRequest{Capacity: &capVal})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/rooms/r1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// P3_153: roomHandler.UpdateRoomStatus() - Success
func TestRoomHandler_P3_153_UpdateRoomStatus_Success(t *testing.T) {
	mockBiz := &mockRoomBiz{
		updateRoomStatusFn: func(_ context.Context, _ string, _ entity.RoomStatus) error { return nil },
	}
	r := setupRoomRouter(mockBiz)
	body, _ := json.Marshal(map[string]string{"status": "MAINTENANCE"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/rooms/r1/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct { Success bool `json:"success"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
}

// P3_154: roomHandler.DeleteRoom() - Success
func TestRoomHandler_P3_154_DeleteRoom_Success(t *testing.T) {
	mockBiz := &mockRoomBiz{
		deleteRoomFn: func(_ context.Context, _ string) error { return nil },
	}
	r := setupRoomRouter(mockBiz)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/rooms/r1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}
