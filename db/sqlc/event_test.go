package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kholodihor/charity/util"
	"github.com/stretchr/testify/require"
)

func createRandomEvent(t *testing.T, store Store) Event {
	name, place, date := util.RandomEventParams()
	arg := CreateEventParams{
		Name:  name,
		Place: place,
		Date:  date,
	}

	event, err := store.CreateEvent(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, event)

	require.Equal(t, arg.Name, event.Name)
	require.Equal(t, arg.Place, event.Place)
	require.WithinDuration(t, arg.Date, event.Date, time.Second)
	require.NotZero(t, event.ID)
	require.NotZero(t, event.CreatedAt)

	return event
}

func TestCreateEvent(t *testing.T) {
	event := createRandomEvent(t, testStore)
	require.NotEmpty(t, event)
}

func TestGetEvent(t *testing.T) {
	event1 := createRandomEvent(t, testStore)
	event2, err := testStore.GetEvent(context.Background(), event1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, event2)

	require.Equal(t, event1.ID, event2.ID)
	require.Equal(t, event1.Name, event2.Name)
	require.Equal(t, event1.Place, event2.Place)
	require.WithinDuration(t, event1.Date, event2.Date, time.Second)
	require.WithinDuration(t, event1.CreatedAt, event2.CreatedAt, time.Second)
}

func TestUpdateEvent(t *testing.T) {
	event1 := createRandomEvent(t, testStore)

	newName := "Updated " + util.RandomString(6)
	newPlace := "New Venue " + util.RandomString(4)
	newDate := time.Now().Add(48 * time.Hour)

	arg := UpdateEventParams{
		ID:    event1.ID,
		Name:  pgtype.Text{String: newName, Valid: true},
		Place: pgtype.Text{String: newPlace, Valid: true},
		Date:  pgtype.Timestamptz{Time: newDate, Valid: true},
	}

	event2, err := testStore.UpdateEvent(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, event2)

	require.Equal(t, event1.ID, event2.ID)
	require.Equal(t, newName, event2.Name)
	require.Equal(t, newPlace, event2.Place)
	require.WithinDuration(t, newDate, event2.Date, time.Second)
	require.WithinDuration(t, event1.CreatedAt, event2.CreatedAt, time.Second)
}

func TestDeleteEvent(t *testing.T) {
	event1 := createRandomEvent(t, testStore)
	err := testStore.DeleteEvent(context.Background(), event1.ID)
	require.NoError(t, err)

	event2, err := testStore.GetEvent(context.Background(), event1.ID)
	require.Error(t, err)
	require.EqualError(t, err, "no rows in result set")
	require.Empty(t, event2)
}

func TestListEvents(t *testing.T) {
	for i := 0; i < 10; i++ {
		createRandomEvent(t, testStore)
	}

	arg := ListEventsParams{
		Limit:  5,
		Offset: 5,
	}

	events, err := testStore.ListEvents(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, events, 5)

	for _, event := range events {
		require.NotEmpty(t, event)
	}
}

func TestListUpcomingEvents(t *testing.T) {
	// Create some events in the future
	for i := 0; i < 5; i++ {
		createRandomEvent(t, testStore)
	}

	arg := ListUpcomingEventsParams{
		Limit:  10,
		Offset: 0,
	}

	events, err := testStore.ListUpcomingEvents(context.Background(), arg)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(events), 5)

	// Verify all events are in the future
	now := time.Now()
	for _, event := range events {
		require.True(t, event.Date.After(now), "Event should be in the future")
	}
}

func TestBookEvent(t *testing.T) {
	user := createRandomUser(t, testStore)
	event := createRandomEvent(t, testStore)

	arg := BookEventParams{
		UserID:  user.ID,
		EventID: event.ID,
	}

	booking, err := testStore.BookEvent(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, booking)

	require.Equal(t, user.ID, booking.UserID)
	require.Equal(t, event.ID, booking.EventID)
	require.NotZero(t, booking.ID)
	require.NotZero(t, booking.BookedAt)
}

func TestBookEventDuplicate(t *testing.T) {
	user := createRandomUser(t, testStore)
	event := createRandomEvent(t, testStore)

	arg := BookEventParams{
		UserID:  user.ID,
		EventID: event.ID,
	}

	// First booking should succeed
	booking1, err := testStore.BookEvent(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, booking1)

	// Second booking should fail due to unique constraint
	booking2, err := testStore.BookEvent(context.Background(), arg)
	require.Error(t, err)
	require.Empty(t, booking2)
	require.Contains(t, err.Error(), "duplicate key value")
}

func TestCancelEventBooking(t *testing.T) {
	user := createRandomUser(t, testStore)
	event := createRandomEvent(t, testStore)

	// First book the event
	bookArg := BookEventParams{
		UserID:  user.ID,
		EventID: event.ID,
	}

	booking, err := testStore.BookEvent(context.Background(), bookArg)
	require.NoError(t, err)
	require.NotEmpty(t, booking)

	// Then cancel the booking
	cancelArg := CancelEventBookingParams{
		UserID:  user.ID,
		EventID: event.ID,
	}

	err = testStore.CancelEventBooking(context.Background(), cancelArg)
	require.NoError(t, err)

	// Verify booking is cancelled
	getArg := GetEventBookingParams{
		UserID:  user.ID,
		EventID: event.ID,
	}

	booking2, err := testStore.GetEventBooking(context.Background(), getArg)
	require.Error(t, err)
	require.EqualError(t, err, "no rows in result set")
	require.Empty(t, booking2)
}

func TestIsEventBooked(t *testing.T) {
	user := createRandomUser(t, testStore)
	event := createRandomEvent(t, testStore)

	// Initially should not be booked
	checkArg := IsEventBookedParams{
		UserID:  user.ID,
		EventID: event.ID,
	}

	isBooked, err := testStore.IsEventBooked(context.Background(), checkArg)
	require.NoError(t, err)
	require.False(t, isBooked)

	// Book the event
	bookArg := BookEventParams{
		UserID:  user.ID,
		EventID: event.ID,
	}

	_, err = testStore.BookEvent(context.Background(), bookArg)
	require.NoError(t, err)

	// Now should be booked
	isBooked, err = testStore.IsEventBooked(context.Background(), checkArg)
	require.NoError(t, err)
	require.True(t, isBooked)
}

func TestListUserBookings(t *testing.T) {
	user := createRandomUser(t, testStore)

	// Create and book multiple events
	for i := 0; i < 3; i++ {
		event := createRandomEvent(t, testStore)

		bookArg := BookEventParams{
			UserID:  user.ID,
			EventID: event.ID,
		}

		_, err := testStore.BookEvent(context.Background(), bookArg)
		require.NoError(t, err)
	}

	// List user bookings
	arg := ListUserBookingsParams{
		UserID: user.ID,
		Limit:  10,
		Offset: 0,
	}
	bookings, err := testStore.ListUserBookings(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, bookings, 3)

	// Verify each booking has event details
	for _, booking := range bookings {
		require.Equal(t, user.ID, booking.UserID)
		require.NotEmpty(t, booking.EventName)
		require.NotEmpty(t, booking.EventPlace)
		require.NotZero(t, booking.EventDate)
	}
}

func TestListEventBookings(t *testing.T) {
	event := createRandomEvent(t, testStore)

	// Create multiple users and book the same event
	for i := 0; i < 3; i++ {
		user := createRandomUser(t, testStore)

		bookArg := BookEventParams{
			UserID:  user.ID,
			EventID: event.ID,
		}

		_, err := testStore.BookEvent(context.Background(), bookArg)
		require.NoError(t, err)
	}

	// List event bookings
	arg := ListEventBookingsParams{
		EventID: event.ID,
		Limit:   10,
		Offset:  0,
	}
	bookings, err := testStore.ListEventBookings(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, bookings, 3)

	// Verify each booking has user details
	for _, booking := range bookings {
		require.Equal(t, event.ID, booking.EventID)
		require.NotEmpty(t, booking.UserEmail)
		require.True(t, booking.UserName.Valid)
		require.NotEmpty(t, booking.UserName.String)
	}
}
