package business

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/cache/v9"
	"github.com/stretchr/testify/assert"
	movieEntity "movie-service/internal/module/movie/entity"
	roomEntity "movie-service/internal/module/room/entity"
	"movie-service/internal/module/showtime/entity"
)

var (
	errConflict = errors.New("conflict")
)

// ============ Mock Repository ============
type mockShowtimeRepository struct {
	getByIDFn       func(ctx context.Context, id string) (*entity.Showtime, error)
	getByIDsFn      func(ctx context.Context, ids []string) ([]*entity.Showtime, error)
	getManFn        func(ctx context.Context, limit, offset int, search, movieId, roomId string, format entity.ShowtimeFormat, status entity.ShowtimeStatus, dateFrom, dateTo *time.Time, excludeEnded bool) ([]*entity.Showtime, error)
	getTotalFn      func(ctx context.Context, search, movieId, roomId string, format entity.ShowtimeFormat, status entity.ShowtimeStatus, dateFrom, dateTo *time.Time, excludeEnded bool) (int, error)
	getUpcomingFn   func(ctx context.Context, limit int) ([]*entity.Showtime, error)
	checkConflictFn func(ctx context.Context, roomId string, startTime, endTime time.Time, excludeId string) (bool, error)
	createFn        func(ctx context.Context, s *entity.Showtime) error
	updateFn        func(ctx context.Context, s *entity.Showtime) error
	deleteFn        func(ctx context.Context, id string) error
	updateStatusFn  func(ctx context.Context, id string, status entity.ShowtimeStatus) error
}

func (m *mockShowtimeRepository) GetByID(ctx context.Context, id string) (*entity.Showtime, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, sql.ErrNoRows
}

func (m *mockShowtimeRepository) GetByIds(ctx context.Context, ids []string) ([]*entity.Showtime, error) {
	if m.getByIDsFn != nil {
		return m.getByIDsFn(ctx, ids)
	}
	return []*entity.Showtime{}, nil
}

func (m *mockShowtimeRepository) GetMany(ctx context.Context, limit, offset int, search, movieId, roomId string, format entity.ShowtimeFormat, status entity.ShowtimeStatus, dateFrom, dateTo *time.Time, excludeEnded bool) ([]*entity.Showtime, error) {
	if m.getManFn != nil {
		return m.getManFn(ctx, limit, offset, search, movieId, roomId, format, status, dateFrom, dateTo, excludeEnded)
	}
	return []*entity.Showtime{}, nil
}

func (m *mockShowtimeRepository) GetTotalCount(ctx context.Context, search, movieId, roomId string, format entity.ShowtimeFormat, status entity.ShowtimeStatus, dateFrom, dateTo *time.Time, excludeEnded bool) (int, error) {
	if m.getTotalFn != nil {
		return m.getTotalFn(ctx, search, movieId, roomId, format, status, dateFrom, dateTo, excludeEnded)
	}
	return 0, nil
}

func (m *mockShowtimeRepository) GetByMovie(ctx context.Context, movieId string) ([]*entity.Showtime, error) {
	return []*entity.Showtime{}, nil
}

func (m *mockShowtimeRepository) Create(ctx context.Context, s *entity.Showtime) error {
	if m.createFn != nil { return m.createFn(ctx, s) }
	return nil
}

func (m *mockShowtimeRepository) Update(ctx context.Context, s *entity.Showtime) error {
	if m.updateFn != nil { return m.updateFn(ctx, s) }
	return nil
}

func (m *mockShowtimeRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil { return m.deleteFn(ctx, id) }
	return nil
}

func (m *mockShowtimeRepository) UpdateStatus(ctx context.Context, id string, status entity.ShowtimeStatus) error {
	if m.updateStatusFn != nil { return m.updateStatusFn(ctx, id, status) }
	return nil
}

func (m *mockShowtimeRepository) GetUpcoming(ctx context.Context, limit int) ([]*entity.Showtime, error) {
	if m.getUpcomingFn != nil { return m.getUpcomingFn(ctx, limit) }
	return nil, nil
}

func (m *mockShowtimeRepository) CheckConflict(ctx context.Context, roomId string, startTime, endTime time.Time, excludeId string) (bool, error) {
	if m.checkConflictFn != nil { return m.checkConflictFn(ctx, roomId, startTime, endTime, excludeId) }
	return false, nil
}

// ============ Mock Movie Biz ============
type mockMovieBiz struct {
	validateFn func(ctx context.Context, movieId string) error
}

func (m *mockMovieBiz) GetMovieById(ctx context.Context, id string) (*movieEntity.Movie, error) {
	return nil, nil
}
func (m *mockMovieBiz) GetMovies(ctx context.Context, page, size int, search string, status string) ([]*movieEntity.Movie, int, error) {
	return nil, 0, nil
}
func (m *mockMovieBiz) GetMovieStats(ctx context.Context) ([]*movieEntity.MovieStat, error) {
	return nil, nil
}
func (m *mockMovieBiz) GetGenres(ctx context.Context) ([]*movieEntity.Genre, error) {
	return nil, nil
}
func (m *mockMovieBiz) CreateMovie(ctx context.Context, movie *movieEntity.Movie, genreIds []string) error {
	return nil
}
func (m *mockMovieBiz) UpdateMovie(ctx context.Context, movie *movieEntity.Movie, genreIds []string) error {
	return nil
}
func (m *mockMovieBiz) DeleteMovie(ctx context.Context, id string) error {
	return nil
}
func (m *mockMovieBiz) UpdateMovieStatus(ctx context.Context, id string, status movieEntity.MovieStatus) error {
	return nil
}
func (m *mockMovieBiz) ValidateMovieForShowtime(ctx context.Context, movieId string) error {
	if m.validateFn != nil {
		return m.validateFn(ctx, movieId)
	}
	return nil
}

// ============ Mock Room Biz ============
type mockRoomBiz struct {
	validateFn func(ctx context.Context, roomId string) error
}

func (m *mockRoomBiz) GetRoomById(ctx context.Context, id string) (*roomEntity.Room, error) {
	return nil, nil
}
func (m *mockRoomBiz) GetRooms(ctx context.Context, page, size int, search string, roomType roomEntity.RoomType, status roomEntity.RoomStatus) ([]*roomEntity.Room, int, error) {
	return nil, 0, nil
}
func (m *mockRoomBiz) CreateRoom(ctx context.Context, room *roomEntity.Room) error {
	return nil
}
func (m *mockRoomBiz) UpdateRoom(ctx context.Context, id string, updates *roomEntity.UpdateRoomRequest) error {
	return nil
}
func (m *mockRoomBiz) DeleteRoom(ctx context.Context, id string) error {
	return nil
}
func (m *mockRoomBiz) UpdateRoomStatus(ctx context.Context, id string, status roomEntity.RoomStatus) error {
	return nil
}
func (m *mockRoomBiz) ValidateRoomForShowtime(ctx context.Context, roomId string) error {
	if m.validateFn != nil {
		return m.validateFn(ctx, roomId)
	}
	return nil
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
func makeValidShowtime(id string) *entity.Showtime {
	return &entity.Showtime{
		Id:        id,
		MovieId:   "movie-1",
		RoomId:    "room-1",
		StartTime: time.Now().Add(2 * time.Hour),
		EndTime:   time.Now().Add(4 * time.Hour),
		Format:    entity.ShowtimeFormat2D,
		BasePrice: 10.0,
		Status:    entity.ShowtimeStatusScheduled,
	}
}

func newTestShowtimeBusiness(t *testing.T, repo ShowtimeRepository) ShowtimeBiz {
	return &business{
		repository:  repo,
		movieBiz:    &mockMovieBiz{},
		roomBiz:     &mockRoomBiz{},
		cache:       &mockCache{&mockROCache{}},
		roCache:     &mockROCache{},
		redisClient: nil,
	}
}

// ---------------------------------------------------------------------------
// Tests P3_024 - P3_043 (Showtime Business)
// ---------------------------------------------------------------------------

// P3_024: GetShowtimeById - Success
func TestShowtimeBusiness_P3_024_GetShowtimeById_Success(t *testing.T) {
	repo := &mockShowtimeRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Showtime, error) {
			return makeValidShowtime(id), nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	showtime, err := biz.GetShowtimeById(context.Background(), "showtime-uuid")
	assert.NoError(t, err)
	assert.NotNil(t, showtime)
	assert.Equal(t, "showtime-uuid", showtime.Id)
}

// P3_025: GetShowtimeById - NotFound
func TestShowtimeBusiness_P3_025_GetShowtimeById_NotFound(t *testing.T) {
	repo := &mockShowtimeRepository{
		getByIDFn: func(_ context.Context, _ string) (*entity.Showtime, error) {
			return nil, sql.ErrNoRows
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	_, err := biz.GetShowtimeById(context.Background(), "bad-id")
	assert.ErrorIs(t, err, ErrShowtimeNotFound)
}

// P3_026: GetShowtimesByIds - BatchFetch
func TestShowtimeBusiness_P3_026_GetShowtimesByIds_BatchFetch(t *testing.T) {
	repo := &mockShowtimeRepository{
		getByIDsFn: func(_ context.Context, ids []string) ([]*entity.Showtime, error) {
			var result []*entity.Showtime
			for _, id := range ids {
				result = append(result, makeValidShowtime(id))
			}
			return result, nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	ids := []string{"showtime-1", "showtime-2"}
	showtimes, err := biz.GetShowtimesByIds(context.Background(), ids)
	assert.NoError(t, err)
	assert.Len(t, showtimes, 2)
}

// P3_027: GetShowtimesByIds - PartialFound
func TestShowtimeBusiness_P3_027_GetShowtimesByIds_PartialFound(t *testing.T) {
	repo := &mockShowtimeRepository{
		getByIDsFn: func(_ context.Context, ids []string) ([]*entity.Showtime, error) {
			if len(ids) > 0 {
				return []*entity.Showtime{makeValidShowtime(ids[0])}, nil
			}
			return nil, nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	showtimes, err := biz.GetShowtimesByIds(context.Background(), []string{"good-id", "bad-id"})
	assert.NoError(t, err)
	assert.Len(t, showtimes, 1)
}

// P3_028: GetShowtimes - WithMovieFilter
func TestShowtimeBusiness_P3_028_GetShowtimes_WithMovieFilter(t *testing.T) {
	repo := &mockShowtimeRepository{
		getManFn: func(_ context.Context, _, _ int, _, movieId, _ string, _ entity.ShowtimeFormat, _ entity.ShowtimeStatus, _, _ *time.Time, _ bool) ([]*entity.Showtime, error) {
			if movieId == "m1" {
				return []*entity.Showtime{{Id: "st-1", MovieId: "m1"}}, nil
			}
			return nil, nil
		},
		getTotalFn: func(_ context.Context, _, movieId, _ string, _ entity.ShowtimeFormat, _ entity.ShowtimeStatus, _, _ *time.Time, _ bool) (int, error) {
			if movieId == "m1" { return 1, nil }
			return 0, nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	showtimes, total, err := biz.GetShowtimes(context.Background(), 1, 10, "", "m1", "", "", "", nil, nil, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, showtimes)
	assert.Equal(t, 1, total)
}

// P3_029: GetShowtimes - WithDateRangeFilter
func TestShowtimeBusiness_P3_029_GetShowtimes_WithDateRangeFilter(t *testing.T) {
	dateFrom := time.Now()
	dateTo := time.Now().Add(24 * time.Hour)
	repo := &mockShowtimeRepository{
		getManFn: func(_ context.Context, _, _ int, _, _, _ string, _ entity.ShowtimeFormat, _ entity.ShowtimeStatus, df, dt *time.Time, _ bool) ([]*entity.Showtime, error) {
			if df != nil && dt != nil { return []*entity.Showtime{{Id: "st-1"}}, nil }
			return nil, nil
		},
		getTotalFn: func(_ context.Context, _, _, _ string, _ entity.ShowtimeFormat, _ entity.ShowtimeStatus, df, dt *time.Time, _ bool) (int, error) {
			if df != nil && dt != nil { return 1, nil }
			return 0, nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	showtimes, total, err := biz.GetShowtimes(context.Background(), 1, 10, "", "", "", "", "", &dateFrom, &dateTo, false)
	assert.NoError(t, err)
	assert.Len(t, showtimes, 1)
	assert.Equal(t, 1, total)
}

// P3_030: GetShowtimes - ExcludeEnded
func TestShowtimeBusiness_P3_030_GetShowtimes_ExcludeEnded(t *testing.T) {
	repo := &mockShowtimeRepository{
		getManFn: func(_ context.Context, _, _ int, _, _, _ string, _ entity.ShowtimeFormat, _ entity.ShowtimeStatus, _, _ *time.Time, ex bool) ([]*entity.Showtime, error) {
			if ex { return []*entity.Showtime{{Id: "st-1"}}, nil }
			return nil, nil
		},
		getTotalFn: func(_ context.Context, _, _, _ string, _ entity.ShowtimeFormat, _ entity.ShowtimeStatus, _, _ *time.Time, ex bool) (int, error) {
			if ex { return 1, nil }
			return 0, nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	showtimes, total, err := biz.GetShowtimes(context.Background(), 1, 10, "", "", "", "", "", nil, nil, true)
	assert.NoError(t, err)
	assert.Len(t, showtimes, 1)
	assert.Equal(t, 1, total)
}

// P3_031: GetUpcomingShowtimes - Success
func TestShowtimeBusiness_P3_031_GetUpcomingShowtimes_Success(t *testing.T) {
	repo := &mockShowtimeRepository{
		getUpcomingFn: func(_ context.Context, limit int) ([]*entity.Showtime, error) {
			return []*entity.Showtime{makeValidShowtime("st-1")}, nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	showtimes, err := biz.GetUpcomingShowtimes(context.Background(), 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, showtimes)
}

// P3_032: CreateShowtime - Success
func TestShowtimeBusiness_P3_032_CreateShowtime_Success(t *testing.T) {
	repo := &mockShowtimeRepository{
		checkConflictFn: func(_ context.Context, _ string, _, _ time.Time, _ string) (bool, error) {
			return false, nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	err := biz.CreateShowtime(context.Background(), makeValidShowtime("st-1"))
	assert.NoError(t, err)
}

// P3_033: CreateShowtime - TimeConflict
func TestShowtimeBusiness_P3_033_CreateShowtime_TimeConflict(t *testing.T) {
	repo := &mockShowtimeRepository{
		checkConflictFn: func(_ context.Context, _ string, _, _ time.Time, _ string) (bool, error) {
			return true, nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	err := biz.CreateShowtime(context.Background(), makeValidShowtime("st-1"))
	assert.ErrorIs(t, err, ErrTimeConflict)
}

// P3_034: CreateShowtime - PastTime
func TestShowtimeBusiness_P3_034_CreateShowtime_PastTime(t *testing.T) {
	repo := &mockShowtimeRepository{}
	biz := newTestShowtimeBusiness(t, repo)
	st := makeValidShowtime("st-1")
	st.StartTime = time.Now().Add(-1 * time.Hour)
	err := biz.CreateShowtime(context.Background(), st)
	assert.ErrorIs(t, err, ErrShowtimeInPast)
}

// P3_035: CreateShowtime - RepoError
func TestShowtimeBusiness_P3_035_CreateShowtime_RepoError(t *testing.T) {
	repo := &mockShowtimeRepository{
		createFn: func(_ context.Context, _ *entity.Showtime) error {
			return errors.New("failed to create")
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	err := biz.CreateShowtime(context.Background(), makeValidShowtime("st-1"))
	assert.Error(t, err)
}

// P3_036: UpdateShowtime - Success
func TestShowtimeBusiness_P3_036_UpdateShowtime_Success(t *testing.T) {
	repo := &mockShowtimeRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Showtime, error) {
			return makeValidShowtime(id), nil
		},
		checkConflictFn: func(_ context.Context, _ string, _, _ time.Time, _ string) (bool, error) {
			return false, nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	newStart := time.Now().Add(5 * time.Hour)
	newEnd := time.Now().Add(7 * time.Hour)
	err := biz.UpdateShowtime(context.Background(), "st-1", &entity.UpdateShowtimeRequest{StartTime: &newStart, EndTime: &newEnd})
	assert.NoError(t, err)
}

// P3_037: UpdateShowtime - CauseConflict
func TestShowtimeBusiness_P3_037_UpdateShowtime_CauseConflict(t *testing.T) {
	repo := &mockShowtimeRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Showtime, error) {
			return makeValidShowtime(id), nil
		},
		checkConflictFn: func(_ context.Context, _ string, _, _ time.Time, _ string) (bool, error) {
			return true, nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	newStart := time.Now().Add(5 * time.Hour)
	newEnd := time.Now().Add(7 * time.Hour)
	err := biz.UpdateShowtime(context.Background(), "st-1", &entity.UpdateShowtimeRequest{StartTime: &newStart, EndTime: &newEnd})
	assert.ErrorIs(t, err, ErrTimeConflict)
}

// P3_038: UpdateShowtime - RepoError
func TestShowtimeBusiness_P3_038_UpdateShowtime_RepoError(t *testing.T) {
	repo := &mockShowtimeRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Showtime, error) {
			return makeValidShowtime(id), nil
		},
		updateFn: func(_ context.Context, _ *entity.Showtime) error {
			return errors.New("failed to update")
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	p := 100.0
	err := biz.UpdateShowtime(context.Background(), "st-1", &entity.UpdateShowtimeRequest{BasePrice: &p})
	assert.Error(t, err)
}

// P3_039: DeleteShowtime - Success
func TestShowtimeBusiness_P3_039_DeleteShowtime_Success(t *testing.T) {
	repo := &mockShowtimeRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Showtime, error) {
			return makeValidShowtime(id), nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	err := biz.DeleteShowtime(context.Background(), "st-1")
	assert.NoError(t, err)
}

// P3_040: DeleteShowtime - HasBookings
func TestShowtimeBusiness_P3_040_DeleteShowtime_HasBookings(t *testing.T) {
	repo := &mockShowtimeRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Showtime, error) {
			return makeValidShowtime(id), nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	// According to spec: Expected Output = "Conflict error"
	err := biz.DeleteShowtime(context.Background(), "st-with-bookings")
	assert.ErrorIs(t, err, errConflict)
}

// P3_041: CheckTimeConflict - NoConflict
func TestShowtimeBusiness_P3_041_CheckTimeConflict_NoConflict(t *testing.T) {
	repo := &mockShowtimeRepository{
		checkConflictFn: func(_ context.Context, _ string, _, _ time.Time, _ string) (bool, error) {
			return false, nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	has, err := biz.CheckTimeConflict(context.Background(), "r1", time.Now(), time.Now(), "")
	assert.NoError(t, err)
	assert.False(t, has)
}

// P3_042: CheckTimeConflict - Conflict
func TestShowtimeBusiness_P3_042_CheckTimeConflict_Conflict(t *testing.T) {
	repo := &mockShowtimeRepository{
		checkConflictFn: func(_ context.Context, _ string, _, _ time.Time, _ string) (bool, error) {
			return true, nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	has, err := biz.CheckTimeConflict(context.Background(), "r1", time.Now(), time.Now(), "")
	assert.NoError(t, err)
	assert.True(t, has)
}

// P3_043: CheckTimeConflict - ExcludeSelf
func TestShowtimeBusiness_P3_043_CheckTimeConflict_ExcludeSelf(t *testing.T) {
	repo := &mockShowtimeRepository{
		checkConflictFn: func(_ context.Context, _ string, _, _ time.Time, ex string) (bool, error) {
			return ex == "", nil
		},
	}
	biz := newTestShowtimeBusiness(t, repo)
	has, err := biz.CheckTimeConflict(context.Background(), "r1", time.Now(), time.Now(), "st-1")
	assert.NoError(t, err)
	assert.False(t, has)
}
