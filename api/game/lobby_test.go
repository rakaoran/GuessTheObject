package game

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupLobby(_ *testing.T) (*lobby, *MockUniqueIdGenerator, *MockPeriodicTickerChannelCreator, chan time.Time, chan time.Time) {
	mockIdGen := &MockUniqueIdGenerator{}
	mockTickerCreator := &MockPeriodicTickerChannelCreator{}

	tickChan := make(chan time.Time)
	pingChan := make(chan time.Time)

	mockTickerCreator.On("Create", time.Second).Return(tickChan)
	mockTickerCreator.On("Create", time.Second*30).Return(pingChan)

	l := NewLobby(mockIdGen, mockTickerCreator)

	return l, mockIdGen, mockTickerCreator, tickChan, pingChan
}

func TestLobby_RequestAddAndRunRoom(t *testing.T) {
	t.Parallel()
	l, _, _, _, _ := setupLobby(t)
	mockRoom := &MockRoom{}

	// We don't need to run the actor here, just testing the method pushes to the channel
	go func() {
		l.RequestAddAndRunRoom(context.Background(), mockRoom)
	}()

	select {
	case r := <-l.addAndRunRoomChan:
		assert.Equal(t, mockRoom, r)
	case <-time.After(time.Second * 5):
		assert.Fail(t, "timed out waiting for room addition")
	}
}

func TestLobby_ForwardPlayerJoinRequestToRoom(t *testing.T) {
	t.Parallel()
	l, _, _, _, _ := setupLobby(t)
	req := roomJoinRequest{roomId: "test-room"}

	go func() {
		l.ForwardPlayerJoinRequestToRoom(context.Background(), req)
	}()

	select {
	case r := <-l.roomJoinReqs:
		assert.Equal(t, req, r)
	case <-time.After(time.Second):
		assert.Fail(t, "timed out waiting for join request forwarding")
	}
}

func TestLobbyActor(t *testing.T) {
	t.Parallel()

	t.Run("Ticker Ticks Rooms", func(t *testing.T) {
		t.Parallel()
		l, _, _, tickChan, _ := setupLobby(t)

		mockRoom := &MockRoom{}
		mockRoom.On("Tick", mock.Anything).Return()

		// Manually inject a room into the map to avoid going through the AddRoom flow for this simple test
		l.rooms["room1"] = mockRoom

		started := make(chan struct{})
		go l.LobbyActor(started)
		<-started

		now := time.Now()
		tickChan <- now
		tickChan <- now

		mockRoom.AssertCalled(t, "Tick", now)
	})

	t.Run("Ping Ticker Pings Players", func(t *testing.T) {
		t.Parallel()
		l, _, _, _, pingChan := setupLobby(t)

		mockRoom := &MockRoom{}
		mockRoom.On("PingPlayers").Return()

		l.rooms["room1"] = mockRoom

		started := make(chan struct{})
		go l.LobbyActor(started)
		<-started

		pingChan <- time.Now()
		pingChan <- time.Now()

		mockRoom.AssertCalled(t, "PingPlayers")
	})

	t.Run("Add Public Room", func(t *testing.T) {
		t.Parallel()
		l, mockIdGen, _, _, _ := setupLobby(t)
		mockRoom := &MockRoom{}
		// Expectations
		mockIdGen.On("Generate").Return("room-123")
		mockRoom.On("SetParentLobby", l).Return()
		mockRoom.On("SetId", "room-123").Return()
		mockRoom.On("Description").Return(roomDescription{id: "room-123", private: false})
		mockRoom.On("GameLoop").Return() // Gets called in a goroutine

		started := make(chan struct{})
		go l.LobbyActor(started)
		<-started

		l.addAndRunRoomChan <- mockRoom

		time.Sleep(50 * time.Millisecond) // Wait for async operations

		mockIdGen.AssertExpectations(t)
		mockRoom.AssertExpectations(t)
		mockRoom.AssertCalled(t, "GameLoop")

		// Verify it's in the public list
		reqChan := make(chan []roomDescription, 1)
		l.pubGamesReq <- reqChan
		descs := <-reqChan
		assert.Len(t, descs, 1)
		assert.Equal(t, "room-123", descs[0].id)
	})

	t.Run("Add Private Room", func(t *testing.T) {
		t.Parallel()
		l, mockIdGen, _, _, _ := setupLobby(t)
		mockRoom := &MockRoom{}
		roomID := "room-private"

		mockIdGen.On("Generate").Return(roomID)
		mockRoom.On("SetParentLobby", l).Return()
		mockRoom.On("SetId", roomID).Return()
		mockRoom.On("GameLoop").Return()
		mockRoom.On("Description").Return(roomDescription{id: roomID, private: true})

		started := make(chan struct{})
		go l.LobbyActor(started)
		<-started

		l.addAndRunRoomChan <- mockRoom

		time.Sleep(20 * time.Millisecond)

		mockIdGen.AssertExpectations(t)
		mockRoom.AssertExpectations(t)
		mockRoom.AssertCalled(t, "GameLoop")

		// Verify it is NOT in the public list
		reqChan := make(chan []roomDescription, 1)
		l.pubGamesReq <- reqChan
		descs := <-reqChan
		assert.Len(t, descs, 0)
	})

	t.Run("Remove Room", func(t *testing.T) {
		t.Parallel()
		l, mockIdGen, _, _, _ := setupLobby(t)
		mockRoom := &MockRoom{}
		roomID := "room-remove"

		// Setup: Add the room first manually
		l.rooms[roomID] = mockRoom
		l.pubRoomsDescriptions[roomID] = roomDescription{id: roomID}

		mockRoom.On("CloseAndRelease").Return()
		mockIdGen.On("Dispose", roomID).Return()

		started := make(chan struct{})
		go l.LobbyActor(started)
		<-started

		l.removeRoomChan <- roomID
		time.Sleep(50 * time.Millisecond)

		mockRoom.AssertExpectations(t)
		mockIdGen.AssertExpectations(t)

		// Verify removed from public list
		reqChan := make(chan []roomDescription, 1)
		l.pubGamesReq <- reqChan
		descs := <-reqChan
		assert.Len(t, descs, 0)

		// Verify removed completely
		req := roomJoinRequest{
			roomId:  "non-existent",
			errChan: make(chan error, 1),
		}

		l.roomJoinReqs <- req

		// Should receive error as the room must be completely removed from the underlying map too
		err := <-req.errChan
		assert.Equal(t, ErrRoomNotFound, err)

	})

	t.Run("Update Room Description", func(t *testing.T) {
		t.Parallel()
		l, _, _, _, _ := setupLobby(t)
		roomID := "room-update"

		started := make(chan struct{})
		go l.LobbyActor(started)
		<-started

		newDesc := roomDescription{id: roomID, playersCount: 5}
		l.roomDescUpdate <- newDesc
		time.Sleep(50 * time.Millisecond)

		reqChan := make(chan []roomDescription, 1)
		l.pubGamesReq <- reqChan
		descs := <-reqChan

		assert.Len(t, descs, 1)
		assert.Equal(t, 5, descs[0].playersCount)
	})

	t.Run("Join Room Success", func(t *testing.T) {
		t.Parallel()
		l, _, _, _, _ := setupLobby(t)
		mockRoom := &MockRoom{}
		roomID := "room-join"

		l.rooms[roomID] = mockRoom

		// Create request
		req := roomJoinRequest{
			roomId:  roomID,
			errChan: make(chan error, 1),
		}

		mockRoom.On("RequestJoin", req).Return()

		started := make(chan struct{})
		go l.LobbyActor(started)
		<-started

		l.roomJoinReqs <- req
		time.Sleep(50 * time.Millisecond)

		mockRoom.AssertExpectations(t)
		assert.Len(t, req.errChan, 0) // Should be no error sent here, room handles it
	})

	t.Run("Join Room Not Found", func(t *testing.T) {
		t.Parallel()
		l, _, _, _, _ := setupLobby(t)

		req := roomJoinRequest{
			roomId:  "non-existent",
			errChan: make(chan error, 1),
		}

		started := make(chan struct{})
		go l.LobbyActor(started)
		<-started

		l.roomJoinReqs <- req

		// Should receive error
		err := <-req.errChan
		assert.Equal(t, ErrRoomNotFound, err)
	})
}
