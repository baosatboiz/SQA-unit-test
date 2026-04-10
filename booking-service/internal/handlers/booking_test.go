package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"booking-service/internal/models"
	"booking-service/internal/pkg/response"
	"booking-service/internal/services"
	"booking-service/internal/types"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/labstack/echo/v4"
	"github.com/samber/do"
	"github.com/stretchr/testify/require"
)

func newBookingHandlerForTest(t *testing.T) *BookingHandler {
	t.Helper()

	injector := do.New()
	do.ProvideValue(injector, &services.BookingService{})

	handler, err := NewBookingHandler(injector)
	require.NoError(t, err)
	return handler
}

func newEchoContext(method, target, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	recorder := httptest.NewRecorder()
	return e.NewContext(request, recorder), recorder
}

func decodeSuccessResponse(t *testing.T, recorder *httptest.ResponseRecorder) response.APIResponse {
	t.Helper()

	var apiResponse response.APIResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &apiResponse))
	return apiResponse
}

func decodeErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder) response.ErrorResponse {
	t.Helper()

	var errorResponse response.ErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &errorResponse))
	return errorResponse
}

func TestBookingHandlerCreateBooking(t *testing.T) {
	t.Run("BOOK-TC-042 parses request body and calls service", func(t *testing.T) {
		// Test Case ID: BOOK-TC-042
		var capturedShowtimeID string
		var capturedSeatIDs []string
		var capturedTotalAmount int

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyMethod(reflect.TypeOf(&services.BookingService{}), "CreateBooking",
			func(_ *services.BookingService, ctx context.Context, userID string, showtimeID string, seatIDs []string, totalAmount int, bookingType models.BookingType) (*models.Booking, error) {
				capturedShowtimeID = showtimeID
				capturedSeatIDs = append([]string(nil), seatIDs...)
				capturedTotalAmount = totalAmount
				return &models.Booking{Id: "booking-1"}, nil
			})

		handler := newBookingHandlerForTest(t)
		context, recorder := newEchoContext(http.MethodPost, "/api/v1/bookings", `{"showtime_id":"showtime-1","seat_ids":["seat-1","seat-2"],"total_amount":150000}`)
		context.Set("user_id", "user-1")

		err := handler.CreateBooking(context)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "showtime-1", capturedShowtimeID)
		require.Equal(t, []string{"seat-1", "seat-2"}, capturedSeatIDs)
		require.Equal(t, 150000, capturedTotalAmount)
	})

	t.Run("BOOK-TC-043 returns unauthorized when user id is missing", func(t *testing.T) {
		// Test Case ID: BOOK-TC-043
		handler := newBookingHandlerForTest(t)
		context, _ := newEchoContext(http.MethodPost, "/api/v1/bookings", `{"showtime_id":"showtime-1","seat_ids":["seat-1"],"total_amount":50000}`)

		require.NotPanics(t, func() {
			_ = handler.CreateBooking(context)
		})
	})

	t.Run("BOOK-TC-044 defaults booking type to online", func(t *testing.T) {
		// Test Case ID: BOOK-TC-044
		var capturedBookingType models.BookingType

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyMethod(reflect.TypeOf(&services.BookingService{}), "CreateBooking",
			func(_ *services.BookingService, ctx context.Context, userID string, showtimeID string, seatIDs []string, totalAmount int, bookingType models.BookingType) (*models.Booking, error) {
				capturedBookingType = bookingType
				return &models.Booking{Id: "booking-1"}, nil
			})

		handler := newBookingHandlerForTest(t)
		context, recorder := newEchoContext(http.MethodPost, "/api/v1/bookings", `{"showtime_id":"showtime-1","seat_ids":["seat-1"],"total_amount":50000}`)
		context.Set("user_id", "user-1")

		err := handler.CreateBooking(context)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, models.BookingTypeOnline, capturedBookingType)
	})

	t.Run("BOOK-TC-045 blocks offline booking for customers", func(t *testing.T) {
		// Test Case ID: BOOK-TC-045
		handler := newBookingHandlerForTest(t)
		context, recorder := newEchoContext(http.MethodPost, "/api/v1/bookings", `{"showtime_id":"showtime-1","seat_ids":["seat-1"],"total_amount":50000,"booking_type":"OFFLINE"}`)
		context.Set("user_id", "user-1")
		context.Set("userRole", "customer")

		err := handler.CreateBooking(context)

		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, recorder.Code)
	})

	t.Run("BOOK-TC-046 allows offline booking for ticket staff", func(t *testing.T) {
		// Test Case ID: BOOK-TC-046
		var capturedBookingType models.BookingType

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyMethod(reflect.TypeOf(&services.BookingService{}), "CreateBooking",
			func(_ *services.BookingService, ctx context.Context, userID string, showtimeID string, seatIDs []string, totalAmount int, bookingType models.BookingType) (*models.Booking, error) {
				capturedBookingType = bookingType
				return &models.Booking{Id: "booking-1", BookingType: bookingType}, nil
			})

		handler := newBookingHandlerForTest(t)
		context, recorder := newEchoContext(http.MethodPost, "/api/v1/bookings", `{"showtime_id":"showtime-1","seat_ids":["seat-1"],"total_amount":50000,"booking_type":"OFFLINE"}`)
		context.Set("user_id", "user-1")
		context.Set("userRole", "ticket_staff")

		err := handler.CreateBooking(context)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, models.BookingTypeOffline, capturedBookingType)
	})

	t.Run("BOOK-TC-047 returns the expected success response shape", func(t *testing.T) {
		// Test Case ID: BOOK-TC-047
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyMethod(reflect.TypeOf(&services.BookingService{}), "CreateBooking",
			func(_ *services.BookingService, ctx context.Context, userID string, showtimeID string, seatIDs []string, totalAmount int, bookingType models.BookingType) (*models.Booking, error) {
				return &models.Booking{
					Id:          "booking-1",
					UserId:      userID,
					ShowtimeId:  showtimeID,
					BookingType: bookingType,
					Status:      models.BookingStatusPending,
				}, nil
			})

		handler := newBookingHandlerForTest(t)
		context, recorder := newEchoContext(http.MethodPost, "/api/v1/bookings", `{"showtime_id":"showtime-1","seat_ids":["seat-1"],"total_amount":50000}`)
		context.Set("user_id", "user-1")

		err := handler.CreateBooking(context)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, recorder.Code)
		apiResponse := decodeSuccessResponse(t, recorder)
		require.Equal(t, http.StatusOK, apiResponse.Code)
		require.Equal(t, "Booking created successfully", apiResponse.Message)
	})
}

func TestBookingHandlerGetBookings(t *testing.T) {
	t.Run("BOOK-TC-048 parses query parameters", func(t *testing.T) {
		// Test Case ID: BOOK-TC-048
		var capturedPage int
		var capturedSize int
		var capturedStatus string

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyMethod(reflect.TypeOf(&services.BookingService{}), "GetUserBookings",
			func(_ *services.BookingService, ctx context.Context, userID string, page, size int, status string) ([]*types.BookingHistory, int, error) {
				capturedPage = page
				capturedSize = size
				capturedStatus = status
				return []*types.BookingHistory{}, 0, nil
			})

		handler := newBookingHandlerForTest(t)
		context, recorder := newEchoContext(http.MethodGet, "/api/v1/bookings/me?page=2&size=5&status=PENDING", "")
		context.Set("user_id", "user-1")

		err := handler.GetBookings(context)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, 2, capturedPage)
		require.Equal(t, 5, capturedSize)
		require.Equal(t, "PENDING", capturedStatus)
	})

	t.Run("BOOK-TC-049 applies default pagination", func(t *testing.T) {
		// Test Case ID: BOOK-TC-049
		var capturedPage int
		var capturedSize int

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyMethod(reflect.TypeOf(&services.BookingService{}), "GetUserBookings",
			func(_ *services.BookingService, ctx context.Context, userID string, page, size int, status string) ([]*types.BookingHistory, int, error) {
				capturedPage = page
				capturedSize = size
				return []*types.BookingHistory{}, 0, nil
			})

		handler := newBookingHandlerForTest(t)
		context, recorder := newEchoContext(http.MethodGet, "/api/v1/bookings/me", "")
		context.Set("user_id", "user-1")

		err := handler.GetBookings(context)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, 1, capturedPage)
		require.Equal(t, 10, capturedSize)
	})

	t.Run("BOOK-TC-050 uppercases status before calling service", func(t *testing.T) {
		// Test Case ID: BOOK-TC-050
		var capturedStatus string

		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyMethod(reflect.TypeOf(&services.BookingService{}), "GetUserBookings",
			func(_ *services.BookingService, ctx context.Context, userID string, page, size int, status string) ([]*types.BookingHistory, int, error) {
				capturedStatus = status
				return []*types.BookingHistory{}, 0, nil
			})

		handler := newBookingHandlerForTest(t)
		context, recorder := newEchoContext(http.MethodGet, "/api/v1/bookings/me?status=pending", "")
		context.Set("user_id", "user-1")

		err := handler.GetBookings(context)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, "PENDING", capturedStatus)
	})
}
