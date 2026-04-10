package services

import (
	"context"
	"database/sql"
	"reflect"
	"regexp"
	"testing"

	"booking-service/internal/datastore"
	grpcclient "booking-service/internal/grpc"
	"booking-service/internal/models"
	"booking-service/proto/pb"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/agiledragon/gomonkey/v2"
	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func newMockBunDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock, func()) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	db := bun.NewDB(sqlDB, pgdialect.New())
	cleanup := func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	}

	return db, mock, cleanup
}

func newRedisClient(t *testing.T) (*miniredis.Miniredis, redis.UniversalClient) {
	t.Helper()

	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		_ = redisClient.Close()
		redisServer.Close()
	})

	return redisServer, redisClient
}

func newBookingServiceForTest() *BookingService {
	return &BookingService{
		movieClient:  &grpcclient.MovieClient{},
		outboxClient: &grpcclient.OutboxClient{},
	}
}

func TestBookingServiceNormalizePagination(t *testing.T) {
	service := &BookingService{}

	t.Run("BOOK-TC-001 sets page zero to one", func(t *testing.T) {
		// Test Case ID: BOOK-TC-001
		page, size, limit, offset := service.normalizePagination(0, 10)
		require.Equal(t, 1, page)
		require.Equal(t, 10, size)
		require.Equal(t, 10, limit)
		require.Equal(t, 0, offset)
	})

	t.Run("BOOK-TC-002 sets negative page to one", func(t *testing.T) {
		// Test Case ID: BOOK-TC-002
		page, size, limit, offset := service.normalizePagination(-5, 10)
		require.Equal(t, 1, page)
		require.Equal(t, 10, size)
		require.Equal(t, 10, limit)
		require.Equal(t, 0, offset)
	})

	t.Run("BOOK-TC-003 sets invalid size to default ten", func(t *testing.T) {
		// Test Case ID: BOOK-TC-003
		page, size, limit, offset := service.normalizePagination(1, 0)
		require.Equal(t, 1, page)
		require.Equal(t, 10, size)
		require.Equal(t, 10, limit)
		require.Equal(t, 0, offset)
	})

	t.Run("BOOK-TC-004 caps size at one hundred", func(t *testing.T) {
		// Test Case ID: BOOK-TC-004
		page, size, limit, offset := service.normalizePagination(1, 200)
		require.Equal(t, 1, page)
		require.Equal(t, 100, size)
		require.Equal(t, 100, limit)
		require.Equal(t, 0, offset)
	})

	t.Run("BOOK-TC-005 calculates offset from page and size", func(t *testing.T) {
		// Test Case ID: BOOK-TC-005
		_, _, _, offset := service.normalizePagination(3, 10)
		require.Equal(t, 20, offset)
	})
}

func TestBookingServiceIsValidStatus(t *testing.T) {
	service := &BookingService{}

	t.Run("BOOK-TC-006 returns true for pending", func(t *testing.T) {
		// Test Case ID: BOOK-TC-006
		require.True(t, service.isValidStatus("PENDING"))
	})
	t.Run("BOOK-TC-007 returns true for confirmed", func(t *testing.T) {
		// Test Case ID: BOOK-TC-007
		require.True(t, service.isValidStatus("CONFIRMED"))
	})
	t.Run("BOOK-TC-008 returns true for cancelled", func(t *testing.T) {
		// Test Case ID: BOOK-TC-008
		require.True(t, service.isValidStatus("CANCELLED"))
	})
	t.Run("BOOK-TC-009 returns false for invalid status", func(t *testing.T) {
		// Test Case ID: BOOK-TC-009
		require.False(t, service.isValidStatus("INVALID"))
	})
	t.Run("BOOK-TC-010 returns false for empty status", func(t *testing.T) {
		// Test Case ID: BOOK-TC-010
		require.False(t, service.isValidStatus(""))
	})
}

func TestBookingServiceCheckSeatAvailability(t *testing.T) {
	t.Run("BOOK-TC-011 returns locked error when another user holds concurrent lock", func(t *testing.T) {
		// Test Case ID: BOOK-TC-011
		redisServer, redisClient := newRedisClient(t)
		service := &BookingService{redisClient: redisClient, roDb: &bun.DB{}}

		redisServer.Set("seat:concurrent_lock:showtime-1:seat-1", "other-user")
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookedSeatsForShowtime, func(ctx context.Context, db bun.IDB, showtimeID string) (map[string]string, error) {
			return map[string]string{}, nil
		})

		err := service.checkSeatAvailability(context.Background(), "showtime-1", []string{"seat-1"}, "user-1")
		require.ErrorIs(t, err, ErrSeatAlreadyLocked)
	})

	t.Run("BOOK-TC-012 allows the current user to reuse their own concurrent lock", func(t *testing.T) {
		// Test Case ID: BOOK-TC-012
		redisServer, redisClient := newRedisClient(t)
		service := &BookingService{redisClient: redisClient, roDb: &bun.DB{}}

		redisServer.Set("seat:concurrent_lock:showtime-1:seat-1", "user-1")
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookedSeatsForShowtime, func(ctx context.Context, db bun.IDB, showtimeID string) (map[string]string, error) {
			return map[string]string{}, nil
		})

		err := service.checkSeatAvailability(context.Background(), "showtime-1", []string{"seat-1"}, "user-1")
		require.NoError(t, err)
	})

	t.Run("BOOK-TC-013 returns booked error when permanent seat lock exists", func(t *testing.T) {
		// Test Case ID: BOOK-TC-013
		redisServer, redisClient := newRedisClient(t)
		service := &BookingService{redisClient: redisClient, roDb: &bun.DB{}}

		redisServer.Set("seat_lock:showtime-1:seat-1", "1")
		err := service.checkSeatAvailability(context.Background(), "showtime-1", []string{"seat-1"}, "user-1")
		require.ErrorIs(t, err, ErrSeatAlreadyBooked)
	})

	t.Run("BOOK-TC-014 returns booked error when seat already exists in database", func(t *testing.T) {
		// Test Case ID: BOOK-TC-014
		_, redisClient := newRedisClient(t)
		service := &BookingService{redisClient: redisClient, roDb: &bun.DB{}}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookedSeatsForShowtime, func(ctx context.Context, db bun.IDB, showtimeID string) (map[string]string, error) {
			return map[string]string{"seat-1": "booking-1"}, nil
		})

		err := service.checkSeatAvailability(context.Background(), "showtime-1", []string{"seat-1"}, "user-1")
		require.ErrorIs(t, err, ErrSeatAlreadyBooked)
	})

	t.Run("BOOK-TC-015 returns nil when every seat is free", func(t *testing.T) {
		// Test Case ID: BOOK-TC-015
		_, redisClient := newRedisClient(t)
		service := &BookingService{redisClient: redisClient, roDb: &bun.DB{}}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookedSeatsForShowtime, func(ctx context.Context, db bun.IDB, showtimeID string) (map[string]string, error) {
			return map[string]string{}, nil
		})

		err := service.checkSeatAvailability(context.Background(), "showtime-1", []string{"seat-1", "seat-2"}, "user-1")
		require.NoError(t, err)
	})
}

func TestBookingServiceCreateBooking(t *testing.T) {
	t.Run("BOOK-TC-016 returns booked error when seat is unavailable", func(t *testing.T) {
		// Test Case ID: BOOK-TC-016
		_, redisClient := newRedisClient(t)
		service := newBookingServiceForTest()
		service.redisClient = redisClient
		service.roDb = &bun.DB{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookedSeatsForShowtime, func(ctx context.Context, db bun.IDB, showtimeID string) (map[string]string, error) {
			return map[string]string{"seat-1": "booking-1"}, nil
		})

		booking, err := service.CreateBooking(context.Background(), "user-1", "showtime-1", []string{"seat-1"}, 100000, models.BookingTypeOnline)
		require.Nil(t, booking)
		require.ErrorIs(t, err, ErrSeatAlreadyBooked)
	})

	t.Run("BOOK-TC-017 acquires a distributed lock for each seat", func(t *testing.T) {
		// Test Case ID: BOOK-TC-017
		// CheckDB: the booking insert is patched out; the assertion focuses on Redis lock creation for each seat.
		redisServer, redisClient := newRedisClient(t)
		db, mock, cleanup := newMockBunDB(t)
		defer cleanup()
		mock.ExpectBegin()
		mock.ExpectCommit()

		service := newBookingServiceForTest()
		service.db = db
		service.roDb = db
		service.redisClient = redisClient

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookedSeatsForShowtime, func(ctx context.Context, db bun.IDB, showtimeID string) (map[string]string, error) {
			return map[string]string{}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetSeatsWithPrice",
			func(_ *grpcclient.MovieClient, ctx context.Context, showtimeID string, seatIDs []string) (*pb.GetSeatsWithPriceResponse, error) {
				return &pb.GetSeatsWithPriceResponse{
					TotalAmount: 100000,
					Data: []*pb.SeatPriceData{
						{SeatId: "seat-1", SeatNumber: "A1", Available: true},
						{SeatId: "seat-2", SeatNumber: "A2", Available: true},
					},
				}, nil
			})
		patches.ApplyFunc(datastore.CreateBooking, func(ctx context.Context, db bun.IDB, booking *models.Booking) error {
			return nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.OutboxClient{}), "CreateOutboxEvent",
			func(_ *grpcclient.OutboxClient, ctx context.Context, eventType string, eventData interface{}) error {
				return nil
			})

		booking, err := service.CreateBooking(context.Background(), "user-1", "showtime-1", []string{"seat-1", "seat-2"}, 100000, models.BookingTypeOnline)
		require.NoError(t, err)
		require.NotNil(t, booking)
		valueOne, errOne := redisServer.Get("seat:concurrent_lock:showtime-1:seat-1")
		valueTwo, errTwo := redisServer.Get("seat:concurrent_lock:showtime-1:seat-2")
		require.NoError(t, errOne)
		require.NoError(t, errTwo)
		require.Equal(t, "user-1", valueOne)
		require.Equal(t, "user-1", valueTwo)
	})

	t.Run("BOOK-TC-018 releases previously acquired locks when a later lock fails", func(t *testing.T) {
		// Test Case ID: BOOK-TC-018
		redisServer, redisClient := newRedisClient(t)
		service := &BookingService{redisClient: redisClient}

		redisServer.Set("seat:concurrent_lock:showtime-1:seat-2", "other-user")
		lockedKeys, err := service.acquireDistributedSeatLocks(context.Background(), "showtime-1", []string{"seat-1", "seat-2"}, "user-1", 0)
		require.Nil(t, lockedKeys)
		require.ErrorIs(t, err, ErrSeatAlreadyLocked)
		require.False(t, redisServer.Exists("seat:concurrent_lock:showtime-1:seat-1"))
	})

	t.Run("BOOK-TC-019 validates price data from movie service", func(t *testing.T) {
		// Test Case ID: BOOK-TC-019
		_, redisClient := newRedisClient(t)
		db, mock, cleanup := newMockBunDB(t)
		defer cleanup()
		mock.ExpectBegin()
		mock.ExpectCommit()

		var priceValidationCalled bool
		service := newBookingServiceForTest()
		service.db = db
		service.roDb = db
		service.redisClient = redisClient

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookedSeatsForShowtime, func(ctx context.Context, db bun.IDB, showtimeID string) (map[string]string, error) {
			return map[string]string{}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetSeatsWithPrice",
			func(_ *grpcclient.MovieClient, ctx context.Context, showtimeID string, seatIDs []string) (*pb.GetSeatsWithPriceResponse, error) {
				priceValidationCalled = true
				return &pb.GetSeatsWithPriceResponse{
					TotalAmount: 150000,
					Data:        []*pb.SeatPriceData{{SeatId: "seat-1", SeatNumber: "A1", Available: true}},
				}, nil
			})
		patches.ApplyFunc(datastore.CreateBooking, func(ctx context.Context, db bun.IDB, booking *models.Booking) error { return nil })
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.OutboxClient{}), "CreateOutboxEvent",
			func(_ *grpcclient.OutboxClient, ctx context.Context, eventType string, eventData interface{}) error { return nil })

		booking, err := service.CreateBooking(context.Background(), "user-1", "showtime-1", []string{"seat-1"}, 150000, models.BookingTypeOnline)
		require.NoError(t, err)
		require.NotNil(t, booking)
		require.True(t, priceValidationCalled)
		require.Equal(t, float64(150000), booking.TotalAmount)
	})

	t.Run("BOOK-TC-020 returns an error when online total amount does not match", func(t *testing.T) {
		// Test Case ID: BOOK-TC-020
		_, redisClient := newRedisClient(t)
		service := newBookingServiceForTest()
		service.redisClient = redisClient
		service.roDb = &bun.DB{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookedSeatsForShowtime, func(ctx context.Context, db bun.IDB, showtimeID string) (map[string]string, error) {
			return map[string]string{}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetSeatsWithPrice",
			func(_ *grpcclient.MovieClient, ctx context.Context, showtimeID string, seatIDs []string) (*pb.GetSeatsWithPriceResponse, error) {
				return &pb.GetSeatsWithPriceResponse{
					TotalAmount: 100000,
					Data:        []*pb.SeatPriceData{{SeatId: "seat-1", SeatNumber: "A1", Available: true}},
				}, nil
			})

		booking, err := service.CreateBooking(context.Background(), "user-1", "showtime-1", []string{"seat-1"}, 120000, models.BookingTypeOnline)
		require.Nil(t, booking)
		require.EqualError(t, err, "invalid total amount: expected 100000.00, got 120000.00")
	})

	t.Run("BOOK-TC-021 accepts a small online amount tolerance", func(t *testing.T) {
		// Test Case ID: BOOK-TC-021
		_, redisClient := newRedisClient(t)
		db, mock, cleanup := newMockBunDB(t)
		defer cleanup()
		mock.ExpectBegin()
		mock.ExpectCommit()

		service := newBookingServiceForTest()
		service.db = db
		service.roDb = db
		service.redisClient = redisClient

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookedSeatsForShowtime, func(ctx context.Context, db bun.IDB, showtimeID string) (map[string]string, error) {
			return map[string]string{}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetSeatsWithPrice",
			func(_ *grpcclient.MovieClient, ctx context.Context, showtimeID string, seatIDs []string) (*pb.GetSeatsWithPriceResponse, error) {
				return &pb.GetSeatsWithPriceResponse{
					TotalAmount: 100000.005,
					Data:        []*pb.SeatPriceData{{SeatId: "seat-1", SeatNumber: "A1", Available: true}},
				}, nil
			})
		patches.ApplyFunc(datastore.CreateBooking, func(ctx context.Context, db bun.IDB, booking *models.Booking) error { return nil })
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.OutboxClient{}), "CreateOutboxEvent",
			func(_ *grpcclient.OutboxClient, ctx context.Context, eventType string, eventData interface{}) error { return nil })

		booking, err := service.CreateBooking(context.Background(), "user-1", "showtime-1", []string{"seat-1"}, 100000, models.BookingTypeOnline)
		require.NoError(t, err)
		require.NotNil(t, booking)
	})
}

func TestBookingServiceQueryFlows(t *testing.T) {
	t.Run("BOOK-TC-022 BOOK-TC-023 BOOK-TC-024 BOOK-TC-025 persists pending booking in transaction, emits outbox event, and returns full booking data", func(t *testing.T) {
		// Test Case ID: BOOK-TC-022
		// Test Case ID: BOOK-TC-023
		// Test Case ID: BOOK-TC-024
		// Test Case ID: BOOK-TC-025
		// CheckDB: verify the insert path is invoked inside a transaction and the pending booking payload is returned.
		_, redisClient := newRedisClient(t)
		db, mock, cleanup := newMockBunDB(t)
		defer cleanup()
		mock.ExpectBegin()
		mock.ExpectCommit()

		var persistedBooking *models.Booking
		var emittedEventType string

		service := newBookingServiceForTest()
		service.db = db
		service.roDb = db
		service.redisClient = redisClient

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookedSeatsForShowtime, func(ctx context.Context, db bun.IDB, showtimeID string) (map[string]string, error) {
			return map[string]string{}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetSeatsWithPrice",
			func(_ *grpcclient.MovieClient, ctx context.Context, showtimeID string, seatIDs []string) (*pb.GetSeatsWithPriceResponse, error) {
				return &pb.GetSeatsWithPriceResponse{
					TotalAmount: 175000,
					Data:        []*pb.SeatPriceData{{SeatId: "seat-1", SeatNumber: "A1", Available: true}},
				}, nil
			})
		patches.ApplyFunc(datastore.CreateBooking, func(ctx context.Context, db bun.IDB, booking *models.Booking) error {
			persistedBooking = booking
			return nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.OutboxClient{}), "CreateOutboxEvent",
			func(_ *grpcclient.OutboxClient, ctx context.Context, eventType string, eventData interface{}) error {
				emittedEventType = eventType
				return nil
			})

		booking, err := service.CreateBooking(context.Background(), "user-1", "showtime-1", []string{"seat-1"}, 175000, models.BookingTypeOnline)
		require.NoError(t, err)
		require.NotNil(t, booking)
		require.NotNil(t, persistedBooking)
		require.NotEmpty(t, booking.Id)
		require.Equal(t, models.BookingStatusPending, booking.Status)
		require.Equal(t, string(models.EventTypeBookingCreated), emittedEventType)
	})

	t.Run("BOOK-TC-026 returns bookings for the requested user", func(t *testing.T) {
		// Test Case ID: BOOK-TC-026
		service := newBookingServiceForTest()
		service.roDb = &bun.DB{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookingsByUserId, func(ctx context.Context, db *bun.DB, userID string, limit, offset int) ([]*models.Booking, error) {
			return []*models.Booking{
				{Id: "booking-1", UserId: userID, ShowtimeId: "showtime-1", Status: models.BookingStatusPending, BookingType: models.BookingTypeOnline},
				{Id: "booking-2", UserId: userID, ShowtimeId: "showtime-2", Status: models.BookingStatusConfirmed, BookingType: models.BookingTypeOnline},
				{Id: "booking-3", UserId: userID, ShowtimeId: "showtime-3", Status: models.BookingStatusCancelled, BookingType: models.BookingTypeOnline},
				{Id: "booking-4", UserId: userID, ShowtimeId: "showtime-4", Status: models.BookingStatusPending, BookingType: models.BookingTypeOnline},
				{Id: "booking-5", UserId: userID, ShowtimeId: "showtime-5", Status: models.BookingStatusConfirmed, BookingType: models.BookingTypeOnline},
			}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetShowtimes",
			func(_ *grpcclient.MovieClient, ctx context.Context, ids []string) ([]*pb.ShowtimeData, error) {
				result := make([]*pb.ShowtimeData, 0, len(ids))
				for _, id := range ids {
					result = append(result, &pb.ShowtimeData{Id: id})
				}
				return result, nil
			})

		bookings, total, err := service.GetUserBookings(context.Background(), "user-1", 1, 10, "")
		require.NoError(t, err)
		require.Len(t, bookings, 5)
		require.Zero(t, total)
	})

	t.Run("BOOK-TC-027 normalizes pagination into limit and offset", func(t *testing.T) {
		// Test Case ID: BOOK-TC-027
		service := newBookingServiceForTest()
		service.roDb = &bun.DB{}

		var capturedLimit, capturedOffset int
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookingsByUserId, func(ctx context.Context, db *bun.DB, userID string, limit, offset int) ([]*models.Booking, error) {
			capturedLimit = limit
			capturedOffset = offset
			return []*models.Booking{
				{Id: "booking-3", UserId: userID, ShowtimeId: "showtime-3", BookingType: models.BookingTypeOnline},
				{Id: "booking-4", UserId: userID, ShowtimeId: "showtime-4", BookingType: models.BookingTypeOnline},
			}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetShowtimes",
			func(_ *grpcclient.MovieClient, ctx context.Context, ids []string) ([]*pb.ShowtimeData, error) {
				result := make([]*pb.ShowtimeData, 0, len(ids))
				for _, id := range ids {
					result = append(result, &pb.ShowtimeData{Id: id})
				}
				return result, nil
			})

		bookings, _, err := service.GetUserBookings(context.Background(), "user-1", 2, 2, "")
		require.NoError(t, err)
		require.Len(t, bookings, 2)
		require.Equal(t, 2, capturedLimit)
		require.Equal(t, 2, capturedOffset)
	})

	t.Run("BOOK-TC-028 filters bookings by status", func(t *testing.T) {
		// Test Case ID: BOOK-TC-028
		service := newBookingServiceForTest()
		service.roDb = &bun.DB{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookingsByUserIdAndStatus, func(ctx context.Context, db *bun.DB, userID string, status models.BookingStatus, limit, offset int) ([]*models.Booking, error) {
			require.Equal(t, models.BookingStatusConfirmed, status)
			return []*models.Booking{
				{Id: "booking-1", UserId: userID, ShowtimeId: "showtime-1", Status: models.BookingStatusConfirmed, BookingType: models.BookingTypeOnline},
			}, nil
		})
		patches.ApplyFunc(datastore.GetTotalBookingsByUserIdAndStatus, func(ctx context.Context, db *bun.DB, userID string, status models.BookingStatus) (int, error) {
			return 1, nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetShowtimes",
			func(_ *grpcclient.MovieClient, ctx context.Context, ids []string) ([]*pb.ShowtimeData, error) {
				return []*pb.ShowtimeData{{Id: "showtime-1"}}, nil
			})

		bookings, total, err := service.GetUserBookings(context.Background(), "user-1", 1, 10, "CONFIRMED")
		require.NoError(t, err)
		require.Len(t, bookings, 1)
		require.Equal(t, 1, total)
		require.Equal(t, "CONFIRMED", bookings[0].Status)
	})

	t.Run("BOOK-TC-029 enriches bookings with showtime data", func(t *testing.T) {
		// Test Case ID: BOOK-TC-029
		service := newBookingServiceForTest()
		service.roDb = &bun.DB{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookingsByUserId, func(ctx context.Context, db *bun.DB, userID string, limit, offset int) ([]*models.Booking, error) {
			return []*models.Booking{
				{Id: "booking-1", UserId: userID, ShowtimeId: "showtime-1", BookingType: models.BookingTypeOnline},
			}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetShowtimes",
			func(_ *grpcclient.MovieClient, ctx context.Context, ids []string) ([]*pb.ShowtimeData, error) {
				return []*pb.ShowtimeData{{
					Id:           "showtime-1",
					MovieTitle:   "Avengers",
					ShowtimeDate: "2026-04-11",
					ShowtimeTime: "19:30",
				}}, nil
			})

		bookings, _, err := service.GetUserBookings(context.Background(), "user-1", 1, 10, "")
		require.NoError(t, err)
		require.Equal(t, "Avengers", bookings[0].MovieTitle)
		require.Equal(t, "2026-04-11", bookings[0].ShowtimeDate)
		require.Equal(t, "19:30", bookings[0].ShowtimeTime)
	})
}

func TestBookingServiceBookingAndTicketStatusFlows(t *testing.T) {
	t.Run("BOOK-TC-030 returns booking when it exists", func(t *testing.T) {
		// Test Case ID: BOOK-TC-030
		service := &BookingService{roDb: &bun.DB{}}
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookingById, func(ctx context.Context, db *bun.DB, id string) (*models.Booking, error) {
			return &models.Booking{Id: id, UserId: "user-1"}, nil
		})

		booking, err := service.GetBookingByID(context.Background(), "booking-1")
		require.NoError(t, err)
		require.Equal(t, "booking-1", booking.Id)
	})

	t.Run("BOOK-TC-031 returns booking not found when datastore misses", func(t *testing.T) {
		// Test Case ID: BOOK-TC-031
		service := &BookingService{roDb: &bun.DB{}}
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookingById, func(ctx context.Context, db *bun.DB, id string) (*models.Booking, error) {
			return nil, sql.ErrNoRows
		})

		booking, err := service.GetBookingByID(context.Background(), "missing")
		require.Nil(t, booking)
		require.ErrorIs(t, err, ErrBookingNotFound)
	})

	t.Run("BOOK-TC-032 rejects invalid status values", func(t *testing.T) {
		// Test Case ID: BOOK-TC-032
		service := &BookingService{}
		userID, err := service.UpdateBookingStatus(context.Background(), "booking-1", "INVALID")
		require.Empty(t, userID)
		require.EqualError(t, err, "invalid booking status: INVALID")
	})

	t.Run("BOOK-TC-033 updates booking status and returns booking user id", func(t *testing.T) {
		// Test Case ID: BOOK-TC-033
		// CheckDB: verify the update path runs before the follow-up read.
		service := &BookingService{db: &bun.DB{}, roDb: &bun.DB{}}
		updateCalled := false

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.UpdateBookingStatus, func(ctx context.Context, db *bun.DB, bookingID string, status models.BookingStatus) error {
			updateCalled = true
			return nil
		})
		patches.ApplyFunc(datastore.GetBookingById, func(ctx context.Context, db *bun.DB, id string) (*models.Booking, error) {
			require.True(t, updateCalled)
			return &models.Booking{Id: id, UserId: "user-1"}, nil
		})

		userID, err := service.UpdateBookingStatus(context.Background(), "booking-1", "CONFIRMED")
		require.NoError(t, err)
		require.Equal(t, "user-1", userID)
	})

	t.Run("BOOK-TC-034 creates a ticket row for every seat", func(t *testing.T) {
		// Test Case ID: BOOK-TC-034
		// CheckDB: verify CreateTickets receives one model per requested seat id.
		service := &BookingService{db: &bun.DB{}}
		var createdTickets []*models.Ticket

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.CreateTickets, func(ctx context.Context, db bun.IDB, tickets []*models.Ticket) error {
			createdTickets = tickets
			return nil
		})

		count, err := service.CreateTicketsForBooking(context.Background(), "booking-1", "showtime-1", []string{"seat-1", "seat-2", "seat-3"})
		require.NoError(t, err)
		require.Equal(t, 3, count)
		require.Len(t, createdTickets, 3)
	})

	t.Run("BOOK-TC-035 generates unique ids for each ticket", func(t *testing.T) {
		// Test Case ID: BOOK-TC-035
		service := &BookingService{db: &bun.DB{}}
		var createdTickets []*models.Ticket

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.CreateTickets, func(ctx context.Context, db bun.IDB, tickets []*models.Ticket) error {
			createdTickets = tickets
			return nil
		})

		_, err := service.CreateTicketsForBooking(context.Background(), "booking-1", "showtime-1", []string{"seat-1", "seat-2", "seat-3"})
		require.NoError(t, err)
		seen := map[string]struct{}{}
		for _, ticket := range createdTickets {
			_, exists := seen[ticket.Id]
			require.False(t, exists)
			seen[ticket.Id] = struct{}{}
		}
	})

	t.Run("BOOK-TC-036 searches tickets by booking id", func(t *testing.T) {
		// Test Case ID: BOOK-TC-036
		service := newBookingServiceForTest()
		service.roDb = &bun.DB{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetBookingByIdWithTickets, func(ctx context.Context, db bun.IDB, id string) (*models.Booking, error) {
			return &models.Booking{
				Id:          id,
				BookingType: models.BookingTypeOnline,
				TotalAmount: 150000,
				Ticket: []*models.Ticket{
					{Id: "ticket-1", BookingId: id, ShowtimeId: "showtime-1", SeatId: "seat-1", Status: models.TicketStatusUnused},
				},
			}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetShowtimes",
			func(_ *grpcclient.MovieClient, ctx context.Context, ids []string) ([]*pb.ShowtimeData, error) {
				return []*pb.ShowtimeData{{Id: "showtime-1"}}, nil
			})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetSeatDetails",
			func(_ *grpcclient.MovieClient, ctx context.Context, ids []string) ([]*pb.SeatDetailData, error) {
				return []*pb.SeatDetailData{{SeatId: "seat-1"}}, nil
			})

		tickets, err := service.SearchTickets(context.Background(), "booking-1", "")
		require.NoError(t, err)
		require.Len(t, tickets, 1)
		require.Equal(t, "booking-1", tickets[0].BookingId)
	})

	t.Run("BOOK-TC-037 searches tickets by showtime id", func(t *testing.T) {
		// Test Case ID: BOOK-TC-037
		service := newBookingServiceForTest()
		service.roDb = &bun.DB{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetTicketsByShowtimeId, func(ctx context.Context, db bun.IDB, showtimeID string) ([]*models.Ticket, error) {
			return []*models.Ticket{
				{Id: "ticket-1", BookingId: "booking-1", ShowtimeId: showtimeID, SeatId: "seat-1", Status: models.TicketStatusUnused},
				{Id: "ticket-2", BookingId: "booking-2", ShowtimeId: showtimeID, SeatId: "seat-2", Status: models.TicketStatusUnused},
			}, nil
		})
		patches.ApplyFunc(datastore.GetBookingsByIds, func(ctx context.Context, db bun.IDB, ids []string) ([]*models.Booking, error) {
			return []*models.Booking{
				{Id: "booking-1", BookingType: models.BookingTypeOnline, TotalAmount: 100000},
				{Id: "booking-2", BookingType: models.BookingTypeOffline, TotalAmount: 120000},
			}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetShowtimes",
			func(_ *grpcclient.MovieClient, ctx context.Context, ids []string) ([]*pb.ShowtimeData, error) {
				return []*pb.ShowtimeData{{Id: "showtime-1"}}, nil
			})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetSeatDetails",
			func(_ *grpcclient.MovieClient, ctx context.Context, ids []string) ([]*pb.SeatDetailData, error) {
				return []*pb.SeatDetailData{{SeatId: "seat-1"}, {SeatId: "seat-2"}}, nil
			})

		tickets, err := service.SearchTickets(context.Background(), "", "showtime-1")
		require.NoError(t, err)
		require.Len(t, tickets, 2)
	})

	t.Run("BOOK-TC-038 enriches tickets with seat details", func(t *testing.T) {
		// Test Case ID: BOOK-TC-038
		service := newBookingServiceForTest()
		service.roDb = &bun.DB{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetTicketsByShowtimeId, func(ctx context.Context, db bun.IDB, showtimeID string) ([]*models.Ticket, error) {
			return []*models.Ticket{
				{Id: "ticket-1", BookingId: "booking-1", ShowtimeId: showtimeID, SeatId: "seat-1", Status: models.TicketStatusUnused},
			}, nil
		})
		patches.ApplyFunc(datastore.GetBookingsByIds, func(ctx context.Context, db bun.IDB, ids []string) ([]*models.Booking, error) {
			return []*models.Booking{{Id: "booking-1", BookingType: models.BookingTypeOnline, TotalAmount: 100000}}, nil
		})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetShowtimes",
			func(_ *grpcclient.MovieClient, ctx context.Context, ids []string) ([]*pb.ShowtimeData, error) {
				return []*pb.ShowtimeData{{Id: "showtime-1", MovieTitle: "Interstellar", ShowtimeDate: "2026-04-12", ShowtimeTime: "20:00"}}, nil
			})
		patches.ApplyMethod(reflect.TypeOf(&grpcclient.MovieClient{}), "GetSeatDetails",
			func(_ *grpcclient.MovieClient, ctx context.Context, ids []string) ([]*pb.SeatDetailData, error) {
				return []*pb.SeatDetailData{{SeatId: "seat-1", SeatRow: "B", SeatNumber: 5, SeatType: "VIP"}}, nil
			})

		tickets, err := service.SearchTickets(context.Background(), "", "showtime-1")
		require.NoError(t, err)
		require.Equal(t, "B", tickets[0].SeatRow)
		require.Equal(t, "5", tickets[0].SeatNumber)
		require.Equal(t, "VIP", tickets[0].SeatType)
	})

	t.Run("BOOK-TC-039 returns invalid data when ticket id is empty", func(t *testing.T) {
		// Test Case ID: BOOK-TC-039
		service := &BookingService{}
		err := service.MarkTicketAsUsed(context.Background(), "")
		require.ErrorIs(t, err, ErrInvalidBookingData)
	})

	t.Run("BOOK-TC-040 returns ticket not found when no ticket exists", func(t *testing.T) {
		// Test Case ID: BOOK-TC-040
		// CheckDB: patch the ticket lookup to simulate a not-found read from the datastore.
		service := &BookingService{db: &bun.DB{}}
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetTicketById, func(ctx context.Context, db bun.IDB, ticketID string) (*models.Ticket, error) {
			return nil, sql.ErrNoRows
		})

		err := service.MarkTicketAsUsed(context.Background(), "missing-ticket")
		require.ErrorIs(t, err, ErrTicketNotFound)
	})

	t.Run("BOOK-TC-041 updates status to used", func(t *testing.T) {
		// Test Case ID: BOOK-TC-041
		// CheckDB: patch the read and update datastore calls and assert the USED status is sent to the update path.
		service := &BookingService{db: &bun.DB{}}
		var updatedStatus models.TicketStatus

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(datastore.GetTicketById, func(ctx context.Context, db bun.IDB, ticketID string) (*models.Ticket, error) {
			return &models.Ticket{Id: ticketID, Status: models.TicketStatusUnused}, nil
		})
		patches.ApplyFunc(datastore.UpdateTicketStatus, func(ctx context.Context, db bun.IDB, ticketID string, status models.TicketStatus) error {
			updatedStatus = status
			return nil
		})

		err := service.MarkTicketAsUsed(context.Background(), "ticket-1")
		require.NoError(t, err)
		require.Equal(t, models.TicketStatusUsed, updatedStatus)
	})
}

func TestDatastoreSqlCompatibilityHelpers(t *testing.T) {
	t.Run("sqlmock helper stays compatible with bun inserts", func(t *testing.T) {
		db, mock, cleanup := newMockBunDB(t)
		defer cleanup()

		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tickets"`)).
			WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(nil, nil))

		err := datastore.CreateTickets(context.Background(), db, []*models.Ticket{
			{Id: "ticket-1", BookingId: "booking-1", ShowtimeId: "showtime-1", SeatId: "seat-1", Status: models.TicketStatusUnused},
		})
		require.NoError(t, err)
	})
}
