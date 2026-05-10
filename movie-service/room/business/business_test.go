package business

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/cache/v9"
	"github.com/stretchr/testify/assert"
	"movie-service/internal/module/room/entity"
)

var (
	errConflict = errors.New("conflict")
)

// ============ Mock Repository ============
type mockRoomRepository struct {
	getByIDFn        func(ctx context.Context, id string) (*entity.Room, error)
	getManFn         func(ctx context.Context, limit, offset int, search string, roomType entity.RoomType, status entity.RoomStatus) ([]*entity.Room, error)
	getTotalFn       func(ctx context.Context, search string, roomType entity.RoomType, status entity.RoomStatus) (int, error)
	createFn         func(ctx context.Context, room *entity.Room) error
	updateFn         func(ctx context.Context, room *entity.Room) error
	deleteFn         func(ctx context.Context, id string) error
	existsByRoomNumFn func(ctx context.Context, roomNumber int, excludeId string) (bool, error)
}

func (m *mockRoomRepository) GetByID(ctx context.Context, id string) (*entity.Room, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, sql.ErrNoRows
}

func (m *mockRoomRepository) GetMany(ctx context.Context, limit, offset int, search string, roomType entity.RoomType, status entity.RoomStatus) ([]*entity.Room, error) {
	if m.getManFn != nil {
		return m.getManFn(ctx, limit, offset, search, roomType, status)
	}
	return []*entity.Room{}, nil
}

func (m *mockRoomRepository) GetTotalCount(ctx context.Context, search string, roomType entity.RoomType, status entity.RoomStatus) (int, error) {
	if m.getTotalFn != nil {
		return m.getTotalFn(ctx, search, roomType, status)
	}
	return 0, nil
}

func (m *mockRoomRepository) Create(ctx context.Context, room *entity.Room) error {
	if m.createFn != nil {
		return m.createFn(ctx, room)
	}
	return nil
}

func (m *mockRoomRepository) Update(ctx context.Context, room *entity.Room) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, room)
	}
	return nil
}

func (m *mockRoomRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockRoomRepository) ExistsByRoomNumber(ctx context.Context, roomNumber int, excludeId string) (bool, error) {
	if m.existsByRoomNumFn != nil {
		return m.existsByRoomNumFn(ctx, roomNumber, excludeId)
	}
	return false, nil
}

// ============ Mock Cache ============
type mockROCache struct {
}

func (m *mockROCache) Get(ctx context.Context, key string, dest interface{}) error {
	return cache.ErrCacheMiss
}

type mockCache struct {
	*mockROCache
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	return nil
}

// ============ Helper Functions ============
func makeValidRoom(id string) *entity.Room {
	return &entity.Room{
		Id:         id,
		RoomNumber: 101,
		Capacity:   50,
		RoomType:   entity.RoomTypeStandard,
		Status:     entity.RoomStatusActive,
	}
}

func newTestRoomBusiness(t *testing.T, repo RoomRepository) RoomBiz {
	return &business{
		repository:  repo,
		cache:       &mockCache{&mockROCache{}},
		roCache:     &mockROCache{},
		redisClient: nil,
	}
}

// ---------------------------------------------------------------------------
// Tests P3_044 - P3_059 (Room Business)
// ---------------------------------------------------------------------------

// P3_044: GetRoomById - Success
func TestRoomBusiness_P3_044_GetRoomById_Success(t *testing.T) {
	repo := &mockRoomRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Room, error) {
			return makeValidRoom(id), nil
		},
	}
	biz := newTestRoomBusiness(t, repo)
	room, err := biz.GetRoomById(context.Background(), "room-1")
	assert.NoError(t, err)
	assert.NotNil(t, room)
	assert.Equal(t, "room-1", room.Id)
}

// P3_045: GetRoomById - NotFound
func TestRoomBusiness_P3_045_GetRoomById_NotFound(t *testing.T) {
	repo := &mockRoomRepository{
		getByIDFn: func(_ context.Context, _ string) (*entity.Room, error) {
			return nil, sql.ErrNoRows
		},
	}
	biz := newTestRoomBusiness(t, repo)
	_, err := biz.GetRoomById(context.Background(), "bad")
	assert.ErrorIs(t, err, ErrRoomNotFound)
}

// P3_046: GetRooms - Success
func TestRoomBusiness_P3_046_GetRooms_Success(t *testing.T) {
	repo := &mockRoomRepository{
		getManFn: func(_ context.Context, _, _ int, _ string, _ entity.RoomType, _ entity.RoomStatus) ([]*entity.Room, error) {
			return []*entity.Room{{Id: "room-1"}, {Id: "room-2"}}, nil
		},
		getTotalFn: func(_ context.Context, _ string, _ entity.RoomType, _ entity.RoomStatus) (int, error) { return 2, nil },
	}
	biz := newTestRoomBusiness(t, repo)
	rooms, total, err := biz.GetRooms(context.Background(), 1, 10, "", "", "")
	assert.NoError(t, err)
	assert.Len(t, rooms, 2)
	assert.Equal(t, 2, total)
}

// P3_047: GetRooms - FilterByType
func TestRoomBusiness_P3_047_GetRooms_FilterByType(t *testing.T) {
	repo := &mockRoomRepository{
		getManFn: func(_ context.Context, _, _ int, _ string, roomType entity.RoomType, _ entity.RoomStatus) ([]*entity.Room, error) {
			if roomType == entity.RoomTypeVIP { return []*entity.Room{{Id: "vip-1", RoomType: entity.RoomTypeVIP}}, nil }
			return nil, nil
		},
		getTotalFn: func(_ context.Context, _ string, roomType entity.RoomType, _ entity.RoomStatus) (int, error) {
			if roomType == entity.RoomTypeVIP { return 1, nil }
			return 0, nil
		},
	}
	biz := newTestRoomBusiness(t, repo)
	rooms, total, err := biz.GetRooms(context.Background(), 1, 10, "", entity.RoomTypeVIP, "")
	assert.NoError(t, err)
	assert.Len(t, rooms, 1)
	assert.Equal(t, 1, total)
}

// P3_048: GetRooms - FilterByStatus
func TestRoomBusiness_P3_048_GetRooms_FilterByStatus(t *testing.T) {
	repo := &mockRoomRepository{
		getManFn: func(_ context.Context, _, _ int, _ string, _ entity.RoomType, status entity.RoomStatus) ([]*entity.Room, error) {
			if status == entity.RoomStatusActive { return []*entity.Room{{Id: "active-1", Status: entity.RoomStatusActive}}, nil }
			return nil, nil
		},
		getTotalFn: func(_ context.Context, _ string, _ entity.RoomType, status entity.RoomStatus) (int, error) {
			if status == entity.RoomStatusActive { return 1, nil }
			return 0, nil
		},
	}
	biz := newTestRoomBusiness(t, repo)
	rooms, _, err := biz.GetRooms(context.Background(), 1, 10, "", "", entity.RoomStatusActive)
	assert.NoError(t, err)
	assert.NotEmpty(t, rooms)
}

// P3_049: CreateRoom - Success
func TestRoomBusiness_P3_049_CreateRoom_Success(t *testing.T) {
	repo := &mockRoomRepository{
		existsByRoomNumFn: func(_ context.Context, _ int, _ string) (bool, error) { return false, nil },
	}
	biz := newTestRoomBusiness(t, repo)
	err := biz.CreateRoom(context.Background(), makeValidRoom("room-1"))
	assert.NoError(t, err)
}

// P3_050: CreateRoom - DuplicateName
func TestRoomBusiness_P3_050_CreateRoom_DuplicateName(t *testing.T) {
	repo := &mockRoomRepository{
		existsByRoomNumFn: func(_ context.Context, _ int, _ string) (bool, error) { return true, nil },
	}
	biz := newTestRoomBusiness(t, repo)
	err := biz.CreateRoom(context.Background(), makeValidRoom("room-1"))
	assert.ErrorIs(t, err, ErrRoomNumberExists)
}

// P3_051: CreateRoom - InvalidCapacity
func TestRoomBusiness_P3_051_CreateRoom_InvalidCapacity(t *testing.T) {
	biz := newTestRoomBusiness(t, &mockRoomRepository{})
	room := makeValidRoom("room-1")
	room.Capacity = -10
	err := biz.CreateRoom(context.Background(), room)
	assert.ErrorIs(t, err, ErrInvalidRoomData)
}

// P3_052: UpdateRoom - Success
func TestRoomBusiness_P3_052_UpdateRoom_Success(t *testing.T) {
	repo := &mockRoomRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Room, error) { return makeValidRoom(id), nil },
		existsByRoomNumFn: func(_ context.Context, _ int, _ string) (bool, error) { return false, nil },
	}
	biz := newTestRoomBusiness(t, repo)
	cap := 100
	err := biz.UpdateRoom(context.Background(), "room-1", &entity.UpdateRoomRequest{Capacity: &cap})
	assert.NoError(t, err)
}

// P3_053: UpdateRoom - RepoError
func TestRoomBusiness_P3_053_UpdateRoom_RepoError(t *testing.T) {
	repo := &mockRoomRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Room, error) { return makeValidRoom(id), nil },
		updateFn: func(_ context.Context, _ *entity.Room) error { return errors.New("failed to update") },
	}
	biz := newTestRoomBusiness(t, repo)
	cap := 100
	err := biz.UpdateRoom(context.Background(), "room-1", &entity.UpdateRoomRequest{Capacity: &cap})
	assert.Error(t, err)
}

// P3_054: DeleteRoom - Success
func TestRoomBusiness_P3_054_DeleteRoom_Success(t *testing.T) {
	repo := &mockRoomRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Room, error) { return makeValidRoom(id), nil },
	}
	biz := newTestRoomBusiness(t, repo)
	err := biz.DeleteRoom(context.Background(), "room-1")
	assert.NoError(t, err)
}

// P3_055: DeleteRoom - HasShowtimes
func TestRoomBusiness_P3_055_DeleteRoom_HasShowtimes(t *testing.T) {
	repo := &mockRoomRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Room, error) { return makeValidRoom(id), nil },
	}
	biz := newTestRoomBusiness(t, repo)
	// According to spec: Expected Output = "Conflict"
	err := biz.DeleteRoom(context.Background(), "room-with-showtimes")
	assert.ErrorIs(t, err, errConflict)
}

// P3_056: UpdateRoomStatus - Success
func TestRoomBusiness_P3_056_UpdateRoomStatus_Success(t *testing.T) {
	repo := &mockRoomRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Room, error) { return makeValidRoom(id), nil },
	}
	biz := newTestRoomBusiness(t, repo)
	err := biz.UpdateRoomStatus(context.Background(), "room-1", entity.RoomStatusMaintenance)
	assert.NoError(t, err)
}

// P3_057: UpdateRoomStatus - RepoError
func TestRoomBusiness_P3_057_UpdateRoomStatus_RepoError(t *testing.T) {
	repo := &mockRoomRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Room, error) { return makeValidRoom(id), nil },
		updateFn: func(_ context.Context, _ *entity.Room) error { return errors.New("failed to update") },
	}
	biz := newTestRoomBusiness(t, repo)
	err := biz.UpdateRoomStatus(context.Background(), "room-1", entity.RoomStatusActive)
	assert.Error(t, err)
}

// P3_058: ValidateRoomForShowtime - Active
func TestRoomBusiness_P3_058_ValidateRoomForShowtime_Active(t *testing.T) {
	repo := &mockRoomRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Room, error) { return makeValidRoom(id), nil },
	}
	biz := newTestRoomBusiness(t, repo)
	err := biz.ValidateRoomForShowtime(context.Background(), "room-1")
	assert.NoError(t, err)
}

// P3_059: ValidateRoomForShowtime - Maintenance
func TestRoomBusiness_P3_059_ValidateRoomForShowtime_Maintenance(t *testing.T) {
	repo := &mockRoomRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Room, error) {
			r := makeValidRoom(id)
			r.Status = entity.RoomStatusMaintenance
			return r, nil
		},
	}
	biz := newTestRoomBusiness(t, repo)
	err := biz.ValidateRoomForShowtime(context.Background(), "room-maintenance")
	assert.ErrorIs(t, err, ErrRoomNotActive)
}
