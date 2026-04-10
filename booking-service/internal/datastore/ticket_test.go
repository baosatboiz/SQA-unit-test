package datastore

import (
	"context"
	"regexp"
	"testing"

	"booking-service/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestCreateTickets(t *testing.T) {
	// Test Case ID: BOOK-TC-056
	// CheckDB: verify the bulk insert query runs once for all ticket rows.
	db, mock, cleanup := newMockDatabase(t)
	defer cleanup()

	tickets := []*models.Ticket{
		{Id: "ticket-1", BookingId: "booking-1", ShowtimeId: "showtime-1", SeatId: "seat-1", Status: models.TicketStatusUnused},
		{Id: "ticket-2", BookingId: "booking-1", ShowtimeId: "showtime-1", SeatId: "seat-2", Status: models.TicketStatusUnused},
		{Id: "ticket-3", BookingId: "booking-1", ShowtimeId: "showtime-1", SeatId: "seat-3", Status: models.TicketStatusUnused},
		{Id: "ticket-4", BookingId: "booking-1", ShowtimeId: "showtime-1", SeatId: "seat-4", Status: models.TicketStatusUnused},
		{Id: "ticket-5", BookingId: "booking-1", ShowtimeId: "showtime-1", SeatId: "seat-5", Status: models.TicketStatusUnused},
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "tickets"`)).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(nil, nil).
			AddRow(nil, nil).
			AddRow(nil, nil).
			AddRow(nil, nil).
			AddRow(nil, nil))

	err := CreateTickets(context.Background(), db, tickets)

	require.NoError(t, err)
}

func TestUpdateTicketStatus(t *testing.T) {
	// Test Case ID: BOOK-TC-057
	// CheckDB: verify the update statement sets both status and updated_at.
	db, mock, cleanup := newMockDatabase(t)
	defer cleanup()

	mock.ExpectExec(`UPDATE "tickets" AS "t" SET status = 'USED', updated_at = CURRENT_TIMESTAMP`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := UpdateTicketStatus(context.Background(), db, "ticket-1", models.TicketStatusUsed)

	require.NoError(t, err)
}
