package datastore

import (
	"context"
	"regexp"
	"testing"
	"time"

	"booking-service/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func newMockDatabase(t *testing.T) (*bun.DB, sqlmock.Sqlmock, func()) {
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

func TestCreateBooking(t *testing.T) {
	// Test Case ID: BOOK-TC-051
	// CheckDB: verify one insert query is issued for the booking record.
	db, mock, cleanup := newMockDatabase(t)
	defer cleanup()

	booking := &models.Booking{
		Id:          "booking-1",
		UserId:      "user-1",
		ShowtimeId:  "showtime-1",
		TotalAmount: 125000,
		Status:      models.BookingStatusPending,
		BookingType: models.BookingTypeOnline,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "bookings"`)).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(nil, nil))

	err := CreateBooking(context.Background(), db, booking)

	require.NoError(t, err)
}

func TestGetBookingById(t *testing.T) {
	// Test Case ID: BOOK-TC-052
	// CheckDB: verify the booking read query returns the expected row and fields.
	db, mock, cleanup := newMockDatabase(t)
	defer cleanup()

	createdAt := time.Date(2026, 4, 11, 9, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT .* FROM "bookings" AS "b"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "showtime_id", "total_amount", "status", "staff_id", "booking_type", "created_at", "updated_at",
		}).AddRow("booking-1", "user-1", "showtime-1", float64(125000), "PENDING", "", "ONLINE", createdAt, nil))

	booking, err := GetBookingById(context.Background(), db, "booking-1")

	require.NoError(t, err)
	require.Equal(t, "booking-1", booking.Id)
	require.Equal(t, "user-1", booking.UserId)
	require.Equal(t, "showtime-1", booking.ShowtimeId)
	require.Equal(t, float64(125000), booking.TotalAmount)
	require.Equal(t, models.BookingStatusPending, booking.Status)
	require.Equal(t, models.BookingTypeOnline, booking.BookingType)
	require.Equal(t, createdAt, booking.CreatedAt)
}

func TestGetBookingsByUserId(t *testing.T) {
	// Test Case ID: BOOK-TC-053
	// CheckDB: verify the query orders bookings by created_at descending.
	db, mock, cleanup := newMockDatabase(t)
	defer cleanup()

	newerTime := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	olderTime := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`SELECT .* FROM "bookings" AS "b".*ORDER BY "created_at" DESC`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "showtime_id", "total_amount", "status", "staff_id", "booking_type", "created_at", "updated_at",
		}).
			AddRow("booking-new", "user-1", "showtime-1", float64(150000), "CONFIRMED", "", "ONLINE", newerTime, nil).
			AddRow("booking-old", "user-1", "showtime-2", float64(90000), "PENDING", "", "ONLINE", olderTime, nil))

	bookings, err := GetBookingsByUserId(context.Background(), db, "user-1", 10, 0)

	require.NoError(t, err)
	require.Len(t, bookings, 2)
	require.Equal(t, "booking-new", bookings[0].Id)
	require.Equal(t, "booking-old", bookings[1].Id)
	require.True(t, bookings[0].CreatedAt.After(bookings[1].CreatedAt))
}

func TestGetBookedSeatsForShowtime(t *testing.T) {
	t.Run("BOOK-TC-054 joins tickets and bookings correctly", func(t *testing.T) {
		// Test Case ID: BOOK-TC-054
		// CheckDB: verify the JOIN query returns seat-to-booking mappings for the showtime.
		db, mock, cleanup := newMockDatabase(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT .*"t"\."seat_id".*"t"\."booking_id".*INNER JOIN bookings b ON b\.id = t\.booking_id.*WHERE \(b\.showtime_id = 'showtime-1'\)`).
			WillReturnRows(sqlmock.NewRows([]string{"seat_id", "booking_id"}).
				AddRow("seat-1", "booking-1").
				AddRow("seat-2", "booking-2"))

		bookedSeats, err := GetBookedSeatsForShowtime(context.Background(), db, "showtime-1")

		require.NoError(t, err)
		require.Equal(t, map[string]string{
			"seat-1": "booking-1",
			"seat-2": "booking-2",
		}, bookedSeats)
	})

	t.Run("BOOK-TC-055 excludes cancelled bookings", func(t *testing.T) {
		// Test Case ID: BOOK-TC-055
		// CheckDB: verify only PENDING and CONFIRMED seats are returned by the query.
		db, mock, cleanup := newMockDatabase(t)
		defer cleanup()

		mock.ExpectQuery(`SELECT .*WHERE .*b\.status IN \('PENDING', 'CONFIRMED'\)`).
			WillReturnRows(sqlmock.NewRows([]string{"seat_id", "booking_id"}).
				AddRow("seat-1", "booking-pending"))

		bookedSeats, err := GetBookedSeatsForShowtime(context.Background(), db, "showtime-1")

		require.NoError(t, err)
		require.Equal(t, map[string]string{"seat-1": "booking-pending"}, bookedSeats)
		require.NotContains(t, bookedSeats, "seat-cancelled")
	})
}
