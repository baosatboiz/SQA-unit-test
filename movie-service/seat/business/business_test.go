package business

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	gocache "github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"

	"movie-service/internal/module/seat/entity"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// mockSeatRepository — spy + function injection
// ---------------------------------------------------------------------------

type mockSeatRepository struct {
	getByIDFn         func(context.Context, string) (*entity.Seat, error)
	getByIDsFn        func(context.Context, []string) ([]*entity.Seat, error)
	getManyFn         func(context.Context, int, int, string, string, string, entity.SeatType, entity.SeatStatus) ([]*entity.Seat, error)
	getTotalCountFn   func(context.Context, string, string, string, entity.SeatType, entity.SeatStatus) (int, error)
	createFn          func(context.Context, *entity.Seat) error
	updateFn          func(context.Context, *entity.Seat) error
	deleteFn          func(context.Context, string) error
	existsBySeatPosFn func(context.Context, string, string, string, string) (bool, error)

	createCalls    int
	updateCalls    int
	deleteCalls    int
	lastCreateSeat *entity.Seat
	lastUpdateSeat *entity.Seat
}

func (m *mockSeatRepository) GetByID(ctx context.Context, id string) (*entity.Seat, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSeatRepository) GetByIDs(ctx context.Context, ids []string) ([]*entity.Seat, error) {
	if m.getByIDsFn != nil {
		return m.getByIDsFn(ctx, ids)
	}
	return nil, nil
}

func (m *mockSeatRepository) GetMany(ctx context.Context, limit, offset int, search, roomId, rowNumber string, seatType entity.SeatType, status entity.SeatStatus) ([]*entity.Seat, error) {
	if m.getManyFn != nil {
		return m.getManyFn(ctx, limit, offset, search, roomId, rowNumber, seatType, status)
	}
	return nil, nil
}

func (m *mockSeatRepository) GetTotalCount(ctx context.Context, search, roomId, rowNumber string, seatType entity.SeatType, status entity.SeatStatus) (int, error) {
	if m.getTotalCountFn != nil {
		return m.getTotalCountFn(ctx, search, roomId, rowNumber, seatType, status)
	}
	return 0, nil
}

func (m *mockSeatRepository) Create(ctx context.Context, seat *entity.Seat) error {
	m.createCalls++
	m.lastCreateSeat = seat
	if m.createFn != nil {
		return m.createFn(ctx, seat)
	}
	return nil
}

func (m *mockSeatRepository) Update(ctx context.Context, seat *entity.Seat) error {
	m.updateCalls++
	m.lastUpdateSeat = seat
	if m.updateFn != nil {
		return m.updateFn(ctx, seat)
	}
	return nil
}

func (m *mockSeatRepository) Delete(ctx context.Context, id string) error {
	m.deleteCalls++
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockSeatRepository) ExistsBySeatPosition(ctx context.Context, roomId, seatNumber, rowNumber, excludeId string) (bool, error) {
	if m.existsBySeatPosFn != nil {
		return m.existsBySeatPosFn(ctx, roomId, seatNumber, rowNumber, excludeId)
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// mockROCache / mockCache
// ---------------------------------------------------------------------------

type mockROCache struct{}

func (m *mockROCache) Get(_ context.Context, _ string, _ any) error {
	return gocache.ErrCacheMiss
}

type mockCache struct {
	*mockROCache
	deleteCalls int
}

func (m *mockCache) Set(_ context.Context, _ string, _ any, _ time.Duration) error { return nil }

func (m *mockCache) Delete(_ context.Context, _ string) error {
	m.deleteCalls++
	return nil
}

// ---------------------------------------------------------------------------
// mockRedisClient — embed interface, implement only Keys + Scan + Del
//   Keys  → getConcurrentLockedSeatsByShowtime / getBookedSeatsByShowtime
//   Scan  → caching.DeleteKeys (invalidateSeatsListCache)
//   Del   → caching.DeleteKeys (invalidateSeatsListCache)
// ---------------------------------------------------------------------------

type mockRedisClient struct {
	redis.UniversalClient            // nil — panic if non-overridden method called
	keysByPattern map[string][]string // pattern → pre-seeded keys
	deletedKeys   []string
	errOnScan     bool
}

func (m *mockRedisClient) Keys(_ context.Context, pattern string) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd(context.Background())
	if m.errOnScan {
		cmd.SetErr(errors.New("redis error"))
		return cmd
	}
	if m.keysByPattern != nil {
		cmd.SetVal(m.keysByPattern[pattern])
	} else {
		cmd.SetVal(nil)
	}
	return cmd
}

func (m *mockRedisClient) Scan(_ context.Context, _ uint64, _ string, _ int64) *redis.ScanCmd {
	cmd := redis.NewScanCmd(context.Background(), nil)
	if m.errOnScan {
		cmd.SetErr(errors.New("redis scan error"))
		return cmd
	}
	cmd.SetVal([]string{}, 0)
	return cmd
}

func (m *mockRedisClient) Del(_ context.Context, keys ...string) *redis.IntCmd {
	m.deletedKeys = append(m.deletedKeys, keys...)
	cmd := redis.NewIntCmd(context.Background())
	cmd.SetVal(int64(len(keys)))
	return cmd
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTestSeatBusiness(t *testing.T, repo *mockSeatRepository, cache *mockCache, rc *mockRedisClient) *business {
	t.Helper()
	if cache == nil {
		cache = &mockCache{mockROCache: &mockROCache{}}
	}
	if rc == nil {
		rc = &mockRedisClient{}
	}
	return &business{
		repository:  repo,
		roCache:     &mockROCache{},
		cache:       cache,
		redisClient: rc,
	}
}

func makeValidSeat(id string) *entity.Seat {
	return &entity.Seat{
		Id:         id,
		RoomId:     "room-1",
		SeatNumber: "A1",
		RowNumber:  "A",
		SeatType:   entity.SeatTypeRegular,
		Status:     entity.SeatStatusAvailable,
	}
}

// ---------------------------------------------------------------------------
// Tests P3_060 - P3_074 (Seat Business)
// ---------------------------------------------------------------------------

// P3_060: GetSeatById - Success
func TestSeatBusiness_P3_060_GetSeatById_Success(t *testing.T) {
	repo := &mockSeatRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Seat, error) {
			return makeValidSeat(id), nil
		},
	}
	biz := newTestSeatBusiness(t, repo, nil, nil)
	seat, err := biz.GetSeatById(context.Background(), "seat-1")
	assert.NoError(t, err)
	assert.NotNil(t, seat)
	assert.Equal(t, "seat-1", seat.Id)
}

// P3_061: GetSeatById - NotFound
func TestSeatBusiness_P3_061_GetSeatById_NotFound(t *testing.T) {
	repo := &mockSeatRepository{
		getByIDFn: func(_ context.Context, _ string) (*entity.Seat, error) {
			return nil, sql.ErrNoRows
		},
	}
	biz := newTestSeatBusiness(t, repo, nil, nil)
	_, err := biz.GetSeatById(context.Background(), "bad-id")
	assert.ErrorIs(t, err, ErrSeatNotFound)
}

// P3_062: GetSeatsByIds - BatchFetch
func TestSeatBusiness_P3_062_GetSeatsByIds_BatchFetch(t *testing.T) {
	repo := &mockSeatRepository{
		getByIDsFn: func(_ context.Context, ids []string) ([]*entity.Seat, error) {
			result := make([]*entity.Seat, len(ids))
			for i, id := range ids { result[i] = makeValidSeat(id) }
			return result, nil
		},
	}
	biz := newTestSeatBusiness(t, repo, nil, nil)
	seats, err := biz.GetSeatsByIds(context.Background(), []string{"s1", "s2"})
	assert.NoError(t, err)
	assert.Len(t, seats, 2)
}

// P3_063: GetSeats - FilterByRoom
func TestSeatBusiness_P3_063_GetSeats_FilterByRoom(t *testing.T) {
	const targetRoom = "room-r1"
	repo := &mockSeatRepository{
		getManyFn: func(_ context.Context, _, _ int, _, roomId, _ string, _ entity.SeatType, _ entity.SeatStatus) ([]*entity.Seat, error) {
			if roomId == targetRoom {
				return []*entity.Seat{{Id: "s1", RoomId: targetRoom}, {Id: "s2", RoomId: targetRoom}}, nil
			}
			return nil, nil
		},
		getTotalCountFn: func(_ context.Context, _, roomId, _ string, _ entity.SeatType, _ entity.SeatStatus) (int, error) {
			if roomId == targetRoom { return 2, nil }
			return 0, nil
		},
	}
	biz := newTestSeatBusiness(t, repo, nil, nil)
	seats, total, err := biz.GetSeats(context.Background(), 1, 10, "", targetRoom, "", "", "")
	assert.NoError(t, err)
	assert.Len(t, seats, 2)
	assert.Equal(t, 2, total)
}

// P3_064: GetSeats - FilterByType
func TestSeatBusiness_P3_064_GetSeats_FilterByType(t *testing.T) {
	repo := &mockSeatRepository{
		getManyFn: func(_ context.Context, _, _ int, _, _, _ string, seatType entity.SeatType, _ entity.SeatStatus) ([]*entity.Seat, error) {
			if seatType == entity.SeatTypeVIP {
				return []*entity.Seat{{Id: "vip-1", SeatType: entity.SeatTypeVIP}}, nil
			}
			return nil, nil
		},
		getTotalCountFn: func(_ context.Context, _, _, _ string, seatType entity.SeatType, _ entity.SeatStatus) (int, error) {
			if seatType == entity.SeatTypeVIP { return 1, nil }
			return 0, nil
		},
	}
	biz := newTestSeatBusiness(t, repo, nil, nil)
	seats, total, err := biz.GetSeats(context.Background(), 1, 10, "", "", "", entity.SeatTypeVIP, "")
	assert.NoError(t, err)
	assert.Len(t, seats, 1)
	assert.Equal(t, 1, total)
}

// P3_065: GetLockedSeatsByShowtime - WithLockedSeats
func TestSeatBusiness_P3_065_GetLockedSeatsByShowtime_WithLockedSeats(t *testing.T) {
	rc := &mockRedisClient{
		keysByPattern: map[string][]string{
			fmt.Sprintf("seat:concurrent_lock:%s:*", "show-1"): {"seat:concurrent_lock:show-1:seat-A"},
			fmt.Sprintf("seat_lock:%s:*", "show-1"): {"seat_lock:show-1:seat-C"},
		},
	}
	biz := newTestSeatBusiness(t, &mockSeatRepository{}, nil, rc)
	resp, err := biz.GetLockedSeatsByShowtime(context.Background(), "show-1")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, append(resp.LockedSeatIds, resp.BookedSeatIds...))
}

// P3_066: GetLockedSeatsByShowtime - NoLocks
func TestSeatBusiness_P3_066_GetLockedSeatsByShowtime_NoLocks(t *testing.T) {
	biz := newTestSeatBusiness(t, &mockSeatRepository{}, nil, nil)
	resp, err := biz.GetLockedSeatsByShowtime(context.Background(), "show-empty")
	assert.NoError(t, err)
	assert.Empty(t, resp.LockedSeatIds)
	assert.Empty(t, resp.BookedSeatIds)
}

// P3_067: GetLockedSeatsByShowtime - RedisError
func TestSeatBusiness_P3_067_GetLockedSeatsByShowtime_RedisError(t *testing.T) {
	rc := &mockRedisClient{
		errOnScan: true,
	}
	biz := newTestSeatBusiness(t, &mockSeatRepository{}, nil, rc)
	_, err := biz.GetLockedSeatsByShowtime(context.Background(), "st1")
	assert.Error(t, err)
}

// P3_068: CreateSeat - Success
func TestSeatBusiness_P3_068_CreateSeat_Success(t *testing.T) {
	repo := &mockSeatRepository{
		existsBySeatPosFn: func(_ context.Context, _, _, _, _ string) (bool, error) { return false, nil },
	}
	biz := newTestSeatBusiness(t, repo, nil, nil)
	err := biz.CreateSeat(context.Background(), makeValidSeat(""))
	assert.NoError(t, err)
}

// P3_069: CreateSeat - DuplicateInRoom
func TestSeatBusiness_P3_069_CreateSeat_DuplicateInRoom(t *testing.T) {
	repo := &mockSeatRepository{
		existsBySeatPosFn: func(_ context.Context, _, _, _, _ string) (bool, error) { return true, nil },
	}
	biz := newTestSeatBusiness(t, repo, nil, nil)
	err := biz.CreateSeat(context.Background(), makeValidSeat(""))
	assert.ErrorIs(t, err, ErrSeatPositionExists)
}

// P3_070: CreateSeat - RepoError
func TestSeatBusiness_P3_070_CreateSeat_RepoError(t *testing.T) {
	repo := &mockSeatRepository{
		existsBySeatPosFn: func(_ context.Context, _, _, _, _ string) (bool, error) { return false, nil },
		createFn: func(_ context.Context, _ *entity.Seat) error { return errors.New("db error") },
	}
	biz := newTestSeatBusiness(t, repo, nil, nil)
	err := biz.CreateSeat(context.Background(), makeValidSeat(""))
	assert.Error(t, err)
}

// P3_071: UpdateSeat - Success
func TestSeatBusiness_P3_071_UpdateSeat_Success(t *testing.T) {
	repo := &mockSeatRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Seat, error) { return makeValidSeat(id), nil },
	}
	biz := newTestSeatBusiness(t, repo, nil, nil)
	newType := entity.SeatTypeVIP
	err := biz.UpdateSeat(context.Background(), "seat-1", &entity.UpdateSeatRequest{SeatType: &newType})
	assert.NoError(t, err)
}

// P3_072: UpdateSeat - NotFound
func TestSeatBusiness_P3_072_UpdateSeat_NotFound(t *testing.T) {
	repo := &mockSeatRepository{
		getByIDFn: func(_ context.Context, _ string) (*entity.Seat, error) { return nil, sql.ErrNoRows },
	}
	biz := newTestSeatBusiness(t, repo, nil, nil)
	newType := entity.SeatTypeVIP
	err := biz.UpdateSeat(context.Background(), "bad", &entity.UpdateSeatRequest{SeatType: &newType})
	assert.ErrorIs(t, err, ErrSeatNotFound)
}

// P3_073: DeleteSeat - Success
func TestSeatBusiness_P3_073_DeleteSeat_Success(t *testing.T) {
	repo := &mockSeatRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Seat, error) { return makeValidSeat(id), nil },
	}
	biz := newTestSeatBusiness(t, repo, nil, nil)
	err := biz.DeleteSeat(context.Background(), "seat-1")
	assert.NoError(t, err)
}

// P3_074: UpdateSeatStatus - Disable
func TestSeatBusiness_P3_074_UpdateSeatStatus_Disable(t *testing.T) {
	repo := &mockSeatRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Seat, error) { return makeValidSeat(id), nil },
	}
	biz := newTestSeatBusiness(t, repo, nil, nil)
	err := biz.UpdateSeatStatus(context.Background(), "seat-1", entity.SeatStatusBlocked)
	assert.NoError(t, err)
}
