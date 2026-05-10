package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	seatEntity "movie-service/internal/module/seat/entity"
	"movie-service/internal/module/showtime/entity"
	"movie-service/proto/pb"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockShowtimeBusiness struct {
	getShowtimeByIdFn             func(context.Context, string) (*entity.Showtime, error)
	getShowtimesByIdsFn           func(context.Context, []string) ([]*entity.Showtime, error)
	validateShowtimeForBookingFn  func(context.Context, string) error
}

func (m *mockShowtimeBusiness) GetShowtimeById(ctx context.Context, id string) (*entity.Showtime, error) {
	if m.getShowtimeByIdFn != nil {
		return m.getShowtimeByIdFn(ctx, id)
	}
	return nil, nil
}
func (m *mockShowtimeBusiness) GetShowtimesByIds(ctx context.Context, ids []string) ([]*entity.Showtime, error) {
	if m.getShowtimesByIdsFn != nil {
		return m.getShowtimesByIdsFn(ctx, ids)
	}
	return nil, nil
}
func (m *mockShowtimeBusiness) ValidateShowtimeForBooking(ctx context.Context, id string) error {
	if m.validateShowtimeForBookingFn != nil {
		return m.validateShowtimeForBookingFn(ctx, id)
	}
	return nil
}

type mockSeatBusiness struct {
	getSeatByIdFn   func(context.Context, string) (*seatEntity.Seat, error)
	getSeatsByIdsFn func(context.Context, []string) ([]*seatEntity.Seat, error)
}

func (m *mockSeatBusiness) GetSeatById(ctx context.Context, id string) (*seatEntity.Seat, error) {
	if m.getSeatByIdFn != nil {
		return m.getSeatByIdFn(ctx, id)
	}
	return nil, nil
}
func (m *mockSeatBusiness) GetSeatsByIds(ctx context.Context, ids []string) ([]*seatEntity.Seat, error) {
	if m.getSeatsByIdsFn != nil {
		return m.getSeatsByIdsFn(ctx, ids)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Tests P3_170 - P3_175 (gRPC Server)
// ---------------------------------------------------------------------------

// P3_170: GetShowtimeByID - Success
func TestMovieGRPCServer_P3_170_GetShowtimeByID_Success(t *testing.T) {
	mockShowtime := &entity.Showtime{
		Id: "st1", MovieId: "m1", RoomId: "r1",
		StartTime: time.Now(), EndTime: time.Now().Add(2 * time.Hour),
		Movie: &entity.Movie{Title: "Movie 1"}, Room: &entity.Room{RoomNumber: 101},
	}
	showtimeBiz := &mockShowtimeBusiness{
		getShowtimeByIdFn: func(_ context.Context, id string) (*entity.Showtime, error) {
			return mockShowtime, nil
		},
	}
	server := NewMovieGRPCServer(showtimeBiz, &mockSeatBusiness{})
	resp, err := server.GetShowtime(context.Background(), &pb.GetShowtimeRequest{Id: "st1"})
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "st1", resp.Data.Id)
}

// P3_171: GetShowtimeByID - NotFound
func TestMovieGRPCServer_P3_171_GetShowtimeByID_NotFound(t *testing.T) {
	showtimeBiz := &mockShowtimeBusiness{
		getShowtimeByIdFn: func(_ context.Context, _ string) (*entity.Showtime, error) {
			return nil, errors.New("not found")
		},
	}
	server := NewMovieGRPCServer(showtimeBiz, &mockSeatBusiness{})
	resp, err := server.GetShowtime(context.Background(), &pb.GetShowtimeRequest{Id: "bad"})
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Message, "not found")
}

// P3_172: GetSeatsByIDs - Batch
func TestMovieGRPCServer_P3_172_GetSeatsByIDs_Batch(t *testing.T) {
	seatBiz := &mockSeatBusiness{
		getSeatsByIdsFn: func(_ context.Context, ids []string) ([]*seatEntity.Seat, error) {
			return []*seatEntity.Seat{
				{Id: "s1", SeatNumber: "1", RowNumber: "A"},
				{Id: "s2", SeatNumber: "2", RowNumber: "A"},
			}, nil
		},
	}
	server := NewMovieGRPCServer(&mockShowtimeBusiness{}, seatBiz)
	resp, err := server.GetSeatDetails(context.Background(), &pb.GetSeatDetailsRequest{SeatIds: []string{"s1", "s2"}})
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Len(t, resp.Data, 2)
}

// P3_173: GetShowtime() - Success with MovieTitle
func TestMovieGRPCServer_P3_173_GetShowtime_MovieTitle(t *testing.T) {
	mockShowtime := &entity.Showtime{
		Id: "st1", Movie: &entity.Movie{Title: "Inception"}, Room: &entity.Room{RoomNumber: 1},
	}
	showtimeBiz := &mockShowtimeBusiness{
		getShowtimeByIdFn: func(_ context.Context, _ string) (*entity.Showtime, error) {
			return mockShowtime, nil
		},
	}
	server := NewMovieGRPCServer(showtimeBiz, &mockSeatBusiness{})
	resp, err := server.GetShowtime(context.Background(), &pb.GetShowtimeRequest{Id: "st1"})
	assert.NoError(t, err)
	assert.Equal(t, "Inception", resp.Data.MovieTitle)
}

// P3_174: GetSeatsWithPrice() - Active Showtime
func TestMovieGRPCServer_P3_174_GetSeatsWithPrice_Active(t *testing.T) {
	showtimeBiz := &mockShowtimeBusiness{
		getShowtimeByIdFn: func(_ context.Context, _ string) (*entity.Showtime, error) {
			return &entity.Showtime{Id: "st1", BasePrice: 100}, nil
		},
		validateShowtimeForBookingFn: func(_ context.Context, _ string) error {
			return nil
		},
	}
	seatBiz := &mockSeatBusiness{
		getSeatsByIdsFn: func(_ context.Context, _ []string) ([]*seatEntity.Seat, error) {
			return []*seatEntity.Seat{{Id: "s1", SeatType: seatEntity.SeatTypeRegular}}, nil
		},
	}
	server := NewMovieGRPCServer(showtimeBiz, seatBiz)
	resp, err := server.GetSeatsWithPrice(context.Background(), &pb.GetSeatsWithPriceRequest{ShowtimeId: "st1", SeatIds: []string{"s1"}})
	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

// P3_175: GetSeatsWithPrice() - Ended Showtime
func TestMovieGRPCServer_P3_175_GetSeatsWithPrice_Ended(t *testing.T) {
	showtimeBiz := &mockShowtimeBusiness{
		validateShowtimeForBookingFn: func(_ context.Context, _ string) error {
			return errors.New("showtime has ended")
		},
	}
	server := NewMovieGRPCServer(showtimeBiz, &mockSeatBusiness{})
	resp, err := server.GetSeatsWithPrice(context.Background(), &pb.GetSeatsWithPriceRequest{ShowtimeId: "ended", SeatIds: []string{"s1"}})
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Message, "ended")
}
