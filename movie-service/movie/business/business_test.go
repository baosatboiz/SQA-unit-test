package business

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	gocache "github.com/go-redis/cache/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"movie-service/internal/module/movie/entity"
)

var (
	errConflict    = errors.New("conflict")
	errInvalidData = errors.New("invalid data")
)

// ---------------------------------------------------------------------------
// mockRepository
// ---------------------------------------------------------------------------

type mockRepository struct {
	getByIDFn       func(context.Context, string) (*entity.Movie, error)
	getManyFn       func(context.Context, int, int, string, string) ([]*entity.Movie, error)
	getTotalCountFn func(context.Context, string, string) (int, error)
	getMovieStatsFn func(context.Context) ([]*entity.MovieStat, error)
	getGenresFn     func(context.Context) ([]*entity.Genre, error)
	createFn        func(context.Context, *entity.Movie, []string) error
	updateFn        func(context.Context, *entity.Movie, []string) error
	deleteFn        func(context.Context, string) error

	getByIDCalls       int
	getManyCalls       int
	getTotalCountCalls int
	getMovieStatsCalls int
	getGenresCalls     int
	createCalls        int
	updateCalls        int
	deleteCalls        int

	lastLimit  int
	lastOffset int
	lastSearch string
	lastStatus string

	lastCreateMovie    *entity.Movie
	lastCreateGenreIDs []string
	lastUpdateMovie    *entity.Movie
	lastUpdateGenreIDs []string
	lastDeleteID       string
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*entity.Movie, error) {
	m.getByIDCalls++
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepository) GetMany(ctx context.Context, limit, offset int, search string, status string) ([]*entity.Movie, error) {
	m.getManyCalls++
	m.lastLimit = limit
	m.lastOffset = offset
	m.lastSearch = search
	m.lastStatus = status
	if m.getManyFn != nil {
		return m.getManyFn(ctx, limit, offset, search, status)
	}
	return nil, nil
}

func (m *mockRepository) GetTotalCount(ctx context.Context, search string, status string) (int, error) {
	m.getTotalCountCalls++
	m.lastSearch = search
	m.lastStatus = status
	if m.getTotalCountFn != nil {
		return m.getTotalCountFn(ctx, search, status)
	}
	return 0, nil
}

func (m *mockRepository) GetMovieStats(ctx context.Context) ([]*entity.MovieStat, error) {
	m.getMovieStatsCalls++
	if m.getMovieStatsFn != nil {
		return m.getMovieStatsFn(ctx)
	}
	return nil, nil
}

func (m *mockRepository) GetGenres(ctx context.Context) ([]*entity.Genre, error) {
	m.getGenresCalls++
	if m.getGenresFn != nil {
		return m.getGenresFn(ctx)
	}
	return nil, nil
}

func (m *mockRepository) Create(ctx context.Context, movie *entity.Movie, genreIds []string) error {
	m.createCalls++
	m.lastCreateMovie = movie
	m.lastCreateGenreIDs = genreIds
	if m.createFn != nil {
		return m.createFn(ctx, movie, genreIds)
	}
	return nil
}

func (m *mockRepository) Update(ctx context.Context, movie *entity.Movie, genreIds []string) error {
	m.updateCalls++
	m.lastUpdateMovie = movie
	m.lastUpdateGenreIDs = genreIds
	if m.updateFn != nil {
		return m.updateFn(ctx, movie, genreIds)
	}
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	m.deleteCalls++
	m.lastDeleteID = id
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// mockROCache
// ---------------------------------------------------------------------------

type mockROCache struct {
	errByKey map[string]error
	data     map[string]any
}

func (m *mockROCache) Get(_ context.Context, key string, target any) error {
	if err, ok := m.errByKey[key]; ok {
		return err
	}
	value, ok := m.data[key]
	if !ok {
		return gocache.ErrCacheMiss
	}
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Ptr {
		return errors.New("target must be pointer")
	}
	targetVal.Elem().Set(reflect.ValueOf(value))
	return nil
}

// ---------------------------------------------------------------------------
// mockCache
// ---------------------------------------------------------------------------

type mockCache struct {
	*mockROCache
	setCalls    int
	deleteCalls int
	deletedKeys []string
}

func (m *mockCache) Set(_ context.Context, key string, value any, _ time.Duration) error {
	m.setCalls++
	if m.data == nil {
		m.data = map[string]any{}
	}
	m.data[key] = value
	return nil
}

func (m *mockCache) Delete(_ context.Context, key string) error {
	m.deleteCalls++
	m.deletedKeys = append(m.deletedKeys, key)
	if m.data != nil {
		delete(m.data, key)
	}
	return nil
}

// ---------------------------------------------------------------------------
// mockRedisClient — embed redis.UniversalClient, chỉ implement Scan + Del
// (2 method duy nhất mà caching.deleteKeys thực sự gọi)
// Các method khác (~50 cái) được kế thừa từ embedded interface nhưng nil:
// nếu test nào gọi nhầm method khác → panic ngay, dễ phát hiện.
// ---------------------------------------------------------------------------

type mockRedisClient struct {
	redis.UniversalClient        // nil — panic nếu gọi method không override
	scannedKeys  []string        // keys giả lập có trong Redis, trả về lúc Scan
	deletedKeys  []string        // ghi lại keys đã bị Del
}

// Scan trả về scannedKeys một lần rồi dừng (cursor=0)
func (m *mockRedisClient) Scan(_ context.Context, _ uint64, _ string, _ int64) *redis.ScanCmd {
	cmd := redis.NewScanCmd(context.Background(), nil)
	cmd.SetVal(m.scannedKeys, 0) // cursor=0 → loop trong deleteKeys dừng
	m.scannedKeys = nil
	return cmd
}

// Del ghi lại keys bị xóa
func (m *mockRedisClient) Del(_ context.Context, keys ...string) *redis.IntCmd {
	m.deletedKeys = append(m.deletedKeys, keys...)
	cmd := redis.NewIntCmd(context.Background())
	cmd.SetVal(int64(len(keys)))
	return cmd
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newTestBusiness wires up a *business hoàn toàn in-memory, zero network.
func newTestBusiness(t *testing.T, repo *mockRepository, ro *mockROCache, cache *mockCache) *business {
	t.Helper()
	return newTestBusinessWithRedis(t, repo, ro, cache, &mockRedisClient{})
}

// newTestBusinessWithRedis cho phép inject mockRedisClient có pre-seeded keys.
func newTestBusinessWithRedis(t *testing.T, repo *mockRepository, ro *mockROCache, cache *mockCache, rc *mockRedisClient) *business {
	t.Helper()
	if ro == nil {
		ro = &mockROCache{errByKey: map[string]error{}, data: map[string]any{}}
	}
	if cache == nil {
		cache = &mockCache{mockROCache: &mockROCache{errByKey: map[string]error{}, data: map[string]any{}}}
	}
	return &business{
		repository:  repo,
		roCache:     ro,
		cache:       cache,
		redisClient: rc,
	}
}

func makeValidMovie(id string, status entity.MovieStatus) *entity.Movie {
	if status == "" {
		status = entity.MovieStatusUpcoming
	}
	return &entity.Movie{
		Id:       id,
		Title:    "Avengers",
		Duration: 120,
		Status:   status,
	}
}

// ---------------------------------------------------------------------------
// Tests P3_001 - P3_023 (Movie Business)
// ---------------------------------------------------------------------------

// P3_001: GetMovieById - Success
func TestMovieBusiness_P3_001_GetMovieById_Success(t *testing.T) {
	repo := &mockRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Movie, error) {
			return makeValidMovie(id, entity.MovieStatusShowing), nil
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	movie, err := biz.GetMovieById(context.Background(), "movie-uuid")
	assert.NoError(t, err)
	assert.NotNil(t, movie)
	assert.Equal(t, "movie-uuid", movie.Id)
}

// P3_002: GetMovieById - NotFound
func TestMovieBusiness_P3_002_GetMovieById_NotFound(t *testing.T) {
	repo := &mockRepository{
		getByIDFn: func(_ context.Context, _ string) (*entity.Movie, error) {
			return nil, sql.ErrNoRows
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	_, err := biz.GetMovieById(context.Background(), "bad")
	assert.ErrorIs(t, err, ErrMovieNotFound)
}

// P3_003: GetMovieById - CacheHit
func TestMovieBusiness_P3_003_GetMovieById_CacheHit(t *testing.T) {
	ro := &mockROCache{
		data: map[string]any{
			redisMovieDetail("movie-uuid"): makeValidMovie("movie-uuid", entity.MovieStatusShowing),
		},
	}
	repo := &mockRepository{}
	biz := newTestBusiness(t, repo, ro, nil)
	movie, err := biz.GetMovieById(context.Background(), "movie-uuid")
	assert.NoError(t, err)
	assert.Equal(t, "movie-uuid", movie.Id)
	assert.Equal(t, 0, repo.getByIDCalls)
}

// P3_004: GetMovies - Success
func TestMovieBusiness_P3_004_GetMovies_Success(t *testing.T) {
	repo := &mockRepository{
		getManyFn: func(_ context.Context, _, _ int, _, _ string) ([]*entity.Movie, error) {
			return []*entity.Movie{makeValidMovie("1", "")}, nil
		},
		getTotalCountFn: func(_ context.Context, _, _ string) (int, error) {
			return 100, nil
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	movies, total, err := biz.GetMovies(context.Background(), 1, 10, "", "")
	assert.NoError(t, err)
	assert.Len(t, movies, 1)
	assert.Equal(t, 100, total)
}

// P3_005: GetMovies - WithSearch
func TestMovieBusiness_P3_005_GetMovies_WithSearch(t *testing.T) {
	repo := &mockRepository{
		getManyFn: func(_ context.Context, _, _ int, search, _ string) ([]*entity.Movie, error) {
			if search == "Aveng" {
				return []*entity.Movie{makeValidMovie("1", "")}, nil
			}
			return nil, nil
		},
		getTotalCountFn: func(_ context.Context, search, _ string) (int, error) {
			if search == "Aveng" { return 1, nil }
			return 0, nil
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	movies, _, _ := biz.GetMovies(context.Background(), 1, 10, "Aveng", "")
	assert.NotEmpty(t, movies)
}

// P3_006: GetMovies - WithStatusFilter
func TestMovieBusiness_P3_006_GetMovies_WithStatusFilter(t *testing.T) {
	repo := &mockRepository{
		getManyFn: func(_ context.Context, _, _ int, _, status string) ([]*entity.Movie, error) {
			if status == string(entity.MovieStatusShowing) {
				return []*entity.Movie{makeValidMovie("1", entity.MovieStatusShowing)}, nil
			}
			return nil, nil
		},
		getTotalCountFn: func(_ context.Context, _, status string) (int, error) {
			if status == string(entity.MovieStatusShowing) { return 1, nil }
			return 0, nil
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	movies, _, _ := biz.GetMovies(context.Background(), 1, 10, "", string(entity.MovieStatusShowing))
	assert.NotEmpty(t, movies)
}

// P3_007: GetMovies - EmptyResult
func TestMovieBusiness_P3_007_GetMovies_EmptyResult(t *testing.T) {
	repo := &mockRepository{
		getManyFn: func(_ context.Context, _, _ int, _, _ string) ([]*entity.Movie, error) {
			return []*entity.Movie{}, nil
		},
		getTotalCountFn: func(_ context.Context, _, _ string) (int, error) {
			return 0, nil
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	movies, total, _ := biz.GetMovies(context.Background(), 1, 10, "xyzzy", "")
	assert.Empty(t, movies)
	assert.Equal(t, 0, total)
}

// P3_008: GetMovies - RepoError
func TestMovieBusiness_P3_008_GetMovies_RepoError(t *testing.T) {
	repo := &mockRepository{
		getTotalCountFn: func(_ context.Context, _, _ string) (int, error) {
			return 0, errors.New("failed to count")
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	_, _, err := biz.GetMovies(context.Background(), 1, 10, "", "")
	assert.Error(t, err)
}

// P3_009: CreateMovie - Success
func TestMovieBusiness_P3_009_CreateMovie_Success(t *testing.T) {
	repo := &mockRepository{}
	biz := newTestBusiness(t, repo, nil, nil)
	movie := makeValidMovie("", "")
	err := biz.CreateMovie(context.Background(), movie, []string{"g1", "g2"})
	assert.NoError(t, err)
	assert.Equal(t, 1, repo.createCalls)
}

// P3_010: CreateMovie - DuplicateTitle
func TestMovieBusiness_P3_010_CreateMovie_DuplicateTitle(t *testing.T) {
	repo := &mockRepository{
		createFn: func(_ context.Context, _ *entity.Movie, _ []string) error {
			return errConflict
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	err := biz.CreateMovie(context.Background(), makeValidMovie("", ""), []string{"g1"})
	assert.ErrorIs(t, err, errConflict)
}

// P3_011: CreateMovie - InvalidGenre
func TestMovieBusiness_P3_011_CreateMovie_InvalidGenre(t *testing.T) {
	repo := &mockRepository{
		createFn: func(_ context.Context, _ *entity.Movie, _ []string) error {
			return errInvalidData
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	err := biz.CreateMovie(context.Background(), makeValidMovie("", ""), []string{"bad"})
	assert.ErrorIs(t, err, errInvalidData)
}

// P3_012: UpdateMovie - Success
func TestMovieBusiness_P3_012_UpdateMovie_Success(t *testing.T) {
	repo := &mockRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Movie, error) {
			return makeValidMovie(id, ""), nil
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	movie := makeValidMovie("movie-uuid", "")
	err := biz.UpdateMovie(context.Background(), movie, []string{"g1"})
	assert.NoError(t, err)
}

// P3_013: UpdateMovie - NotFound
func TestMovieBusiness_P3_013_UpdateMovie_NotFound(t *testing.T) {
	repo := &mockRepository{
		getByIDFn: func(_ context.Context, _ string) (*entity.Movie, error) {
			return nil, sql.ErrNoRows
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	err := biz.UpdateMovie(context.Background(), makeValidMovie("bad", ""), nil)
	assert.ErrorIs(t, err, ErrMovieNotFound)
}

// P3_014: DeleteMovie - Success
func TestMovieBusiness_P3_014_DeleteMovie_Success(t *testing.T) {
	repo := &mockRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Movie, error) {
			return makeValidMovie(id, ""), nil
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	err := biz.DeleteMovie(context.Background(), "movie-uuid")
	assert.NoError(t, err)
}

// P3_015: DeleteMovie - HasShowtimes
func TestMovieBusiness_P3_015_DeleteMovie_HasShowtimes(t *testing.T) {
	repo := &mockRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Movie, error) {
			return makeValidMovie(id, ""), nil
		},
		deleteFn: func(_ context.Context, _ string) error { return errConflict },
	}
	biz := newTestBusiness(t, repo, nil, nil)
	err := biz.DeleteMovie(context.Background(), "movie-uuid")
	assert.ErrorIs(t, err, errConflict)
}

// P3_016: UpdateMovieStatus - Success
func TestMovieBusiness_P3_016_UpdateMovieStatus_Success(t *testing.T) {
	repo := &mockRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Movie, error) {
			return makeValidMovie(id, entity.MovieStatusUpcoming), nil
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	err := biz.UpdateMovieStatus(context.Background(), "movie-uuid", entity.MovieStatusShowing)
	assert.NoError(t, err)
}

// P3_017: UpdateMovieStatus - InvalidStatus
func TestMovieBusiness_P3_017_UpdateMovieStatus_InvalidStatus(t *testing.T) {
	repo := &mockRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Movie, error) {
			return makeValidMovie(id, entity.MovieStatusUpcoming), nil
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	err := biz.UpdateMovieStatus(context.Background(), "movie-uuid", entity.MovieStatus("INVALID"))
	
	// According to spec: Expected Output = "Validation error"
	// In this module, validation errors are typically ErrInvalidMovieData.
	assert.ErrorIs(t, err, ErrInvalidMovieData)
}

// P3_018: GetGenres - Success
func TestMovieBusiness_P3_018_GetGenres_Success(t *testing.T) {
	repo := &mockRepository{
		getGenresFn: func(_ context.Context) ([]*entity.Genre, error) {
			return []*entity.Genre{{Id: "1", Name: "Action"}}, nil
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	genres, err := biz.GetGenres(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, genres)
}

// P3_019: GetGenres - RepoError
func TestMovieBusiness_P3_019_GetGenres_RepoError(t *testing.T) {
	repo := &mockRepository{
		getGenresFn: func(_ context.Context) ([]*entity.Genre, error) {
			return nil, errors.New("failed to get genres")
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	_, err := biz.GetGenres(context.Background())
	assert.Error(t, err)
}

// P3_020: GetMovieStats - Success
func TestMovieBusiness_P3_020_GetMovieStats_Success(t *testing.T) {
	repo := &mockRepository{
		getMovieStatsFn: func(_ context.Context) ([]*entity.MovieStat, error) {
			return []*entity.MovieStat{{Status: "SHOWING", Count: 10}}, nil
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	stats, err := biz.GetMovieStats(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, stats)
}

// P3_021: GetMovieStats - RepoError
func TestMovieBusiness_P3_021_GetMovieStats_RepoError(t *testing.T) {
	repo := &mockRepository{
		getMovieStatsFn: func(_ context.Context) ([]*entity.MovieStat, error) {
			return nil, errors.New("failed to get stats")
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	_, err := biz.GetMovieStats(context.Background())
	assert.Error(t, err)
}

// P3_022: ValidateMovieForShowtime - Active
func TestMovieBusiness_P3_022_ValidateMovieForShowtime_Active(t *testing.T) {
	repo := &mockRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Movie, error) {
			return makeValidMovie(id, entity.MovieStatusShowing), nil
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	err := biz.ValidateMovieForShowtime(context.Background(), "movie-uuid")
	assert.NoError(t, err)
}

// P3_023: ValidateMovieForShowtime - NotActive
func TestMovieBusiness_P3_023_ValidateMovieForShowtime_NotActive(t *testing.T) {
	repo := &mockRepository{
		getByIDFn: func(_ context.Context, id string) (*entity.Movie, error) {
			return makeValidMovie(id, entity.MovieStatusEnded), nil
		},
	}
	biz := newTestBusiness(t, repo, nil, nil)
	err := biz.ValidateMovieForShowtime(context.Background(), "movie-uuid")
	assert.ErrorIs(t, err, ErrMovieNotShowing)
}
