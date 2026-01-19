package game

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func br(i int) {
	println("mmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmmm", i)
}

func TestLobby(t *testing.T) {

	mockTickerCreator := &MockPeriodicTickerChannelCreator{}
	mockIdgenerator := &MockUniqueIdGenerator{}

	ticker := make(chan time.Time)
	pingTicker := make(chan time.Time)
	mockTickerCreator.On("Create", time.Second).Return(ticker)
	mockTickerCreator.On("Create", time.Second*30).Return(pingTicker)

	lobby := NewLobby(mockIdgenerator, mockTickerCreator)
	startedSignal := make(chan struct{})
	go lobby.LobbyActor(startedSignal)

	<-startedSignal

	// when no room is there
	pingTicker <- time.Now()
	ticker <- time.Now()

	mockIdgenerator.On("Generate").Return("id1").Once()
	mockIdgenerator.On("Generate").Return("id2").Once()
	mockIdgenerator.On("Generate").Return("id3").Once()
	mockIdgenerator.On("Generate").Return("id4").Once()
	mockIdgenerator.On("Dispose", "id1").Return()
	mockIdgenerator.On("Dispose", "id2").Return()
	mockIdgenerator.On("Dispose", "id3").Return()
	mockIdgenerator.On("Dispose", "id44").Return()
	mockIdgenerator.On("Dispose", "RESERVED").Return()

	room1 := NewRoom(nil, true, 0, 0, 0, 0, 0, nil)
	room2 := NewRoom(nil, true, 0, 0, 0, 0, 0, nil)
	room3 := NewRoom(nil, false, 15, 0, 0, 0, 0, nil)
	room4 := NewRoom(nil, false, 7, 0, 0, 0, 0, nil)

	t.Run("Add Room 1 (Private)", func(t *testing.T) {

		lobby.addRoomChan <- room1

		tick := time.Now()
		ping := time.Now().Add(time.Hour)

		ticker <- tick
		pingTicker <- ping

		tick1 := <-room1.ticks
		_, ok := <-room1.pingPlayers

		assert.Equal(t, tick, tick1)
		assert.True(t, ok)
		assert.Equal(t, "id1", room1.id)
		pubRoomsReq := make(chan []RoomDescription, 1)
		lobby.pubGamesReq <- pubRoomsReq
		assert.Empty(t, <-pubRoomsReq)
	})

	t.Run("Add Room 2 (Private)", func(t *testing.T) {

		lobby.addRoomChan <- room2

		tick := time.Now()
		ping := time.Now().Add(time.Hour)

		ticker <- tick
		pingTicker <- ping

		tick1 := <-room1.ticks
		_, ok1 := <-room1.pingPlayers

		tick2 := <-room2.ticks
		_, ok2 := <-room2.pingPlayers

		assert.Equal(t, tick, tick1)
		assert.Equal(t, tick, tick2)
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.Equal(t, "id2", room2.id)

		pubRoomsReq := make(chan []RoomDescription, 1)
		lobby.pubGamesReq <- pubRoomsReq
		assert.Empty(t, <-pubRoomsReq)
	})

	t.Run("Add Room 3 (Public)", func(t *testing.T) {

		tick := time.Now()
		ping := time.Now().Add(time.Hour)

		lobby.addRoomChan <- room3

		ticker <- tick
		pingTicker <- ping

		tick1 := <-room1.ticks
		_, ok1 := <-room1.pingPlayers

		tick2 := <-room2.ticks
		_, ok2 := <-room2.pingPlayers

		tick3 := <-room3.ticks
		_, ok3 := <-room3.pingPlayers

		assert.Equal(t, tick, tick1)
		assert.Equal(t, tick, tick2)
		assert.Equal(t, tick, tick3)
		assert.True(t, ok1)
		assert.True(t, ok2)
		assert.True(t, ok3)
		assert.Equal(t, "id3", room3.id)

		pubRoomsReq := make(chan []RoomDescription, 1)
		lobby.pubGamesReq <- pubRoomsReq
		expectedRoomDescs := []RoomDescription{
			{id: "id3", playersCount: 1, maxPlayers: 15, started: false},
		}
		assert.ElementsMatch(t, expectedRoomDescs, <-pubRoomsReq)
	})

	t.Run("Remove Rooms 1 and 2", func(t *testing.T) {
		tick := time.Now()
		ping := time.Now().Add(time.Hour)

		lobby.removeRoomChan <- room1

		_, okT1 := <-room1.ticks
		_, okP1 := <-room1.pingPlayers

		lobby.removeRoomChan <- room2
		_, okT2 := <-room2.ticks
		br(99)
		_, okP2 := <-room2.pingPlayers
		br(0)
		// to verify that all waiting players to be removed are done
		helperRoom := NewRoom(nil, true, 0, 0, 0, 0, 0, nil)
		helperRoom.joinRequests = lobby.joinRoomReq
		helperRoom.id = "RESERVED"
		lobby.removeRoomChan <- helperRoom
		<-helperRoom.ticks
		br(1)
		ticker <- tick
		pingTicker <- ping

		tick3 := <-room3.ticks
		_, okP3 := <-room3.pingPlayers
		br(2)
		assert.Equal(t, tick, tick3)
		assert.True(t, okP3)
		assert.False(t, okP1)
		assert.False(t, okT1)
		assert.False(t, okP2)
		assert.False(t, okT2)

		pubRoomsReq := make(chan []RoomDescription, 1)
		lobby.pubGamesReq <- pubRoomsReq
		expectedRoomDescs := []RoomDescription{
			{id: "id3", playersCount: 1, maxPlayers: 15, started: false},
		}
		br(3)
		assert.ElementsMatch(t, expectedRoomDescs, <-pubRoomsReq)
	})

	t.Run("Add Room 4 (Public)", func(t *testing.T) {

		tick := time.Now()
		ping := time.Now().Add(time.Hour)

		lobby.addRoomChan <- room4

		ticker <- tick
		pingTicker <- ping

		tick3 := <-room3.ticks
		_, ok3 := <-room3.pingPlayers

		tick4 := <-room4.ticks
		_, ok4 := <-room4.pingPlayers

		assert.Equal(t, tick, tick3)
		assert.True(t, ok3)
		assert.Equal(t, tick, tick4)
		assert.True(t, ok4)
		assert.Equal(t, "id4", room4.id)

		pubRoomsReq := make(chan []RoomDescription, 1)
		lobby.pubGamesReq <- pubRoomsReq
		expectedRoomDescs := []RoomDescription{
			{id: "id3", playersCount: 1, maxPlayers: 15, started: false},
			{id: "id4", playersCount: 1, maxPlayers: 7, started: false},
		}
		assert.ElementsMatch(t, expectedRoomDescs, <-pubRoomsReq)
		//riemtrimtriesmtesm
	})

	t.Run("Update Description For Room 3", func(t *testing.T) {
		room3.updateDescriptionChan <- RoomDescription{id: room3.id, playersCount: 55, started: true, maxPlayers: 21}

		pubRoomsReq := make(chan []RoomDescription, 1)
		lobby.pubGamesReq <- pubRoomsReq
		expectedRoomDescs := []RoomDescription{
			{id: "id3", playersCount: 55, maxPlayers: 21, started: true},
			{id: "id4", playersCount: 1, maxPlayers: 7, started: false},
		}
		assert.ElementsMatch(t, expectedRoomDescs, <-pubRoomsReq)
	})

	t.Run("Room Join Request Forwarding Correct ID", func(t *testing.T) {
		roomJoinreq := RoomJoinRequest{roomId: "id4", player: &Player{}, errChan: nil} // expect no error bru

		lobby.joinRoomReq <- roomJoinreq

		assert.Equal(t, roomJoinreq, <-room4.joinRequests)
	})

	t.Run("Room Join Request Forwarding Wrong ID", func(t *testing.T) {
		errChan := make(chan error, 1)
		roomJoinreq := RoomJoinRequest{roomId: "WRONG ID HAHA", player: &Player{}, errChan: errChan}
		lobby.joinRoomReq <- roomJoinreq
		assert.Equal(t, ErrRoomNotFound, <-errChan)
	})

	t.Run("Remove Room 3", func(t *testing.T) {
		lobby.removeRoomChan <- room3
		_, ok := <-room3.pingPlayers

		assert.False(t, ok)

		pubRoomsReq := make(chan []RoomDescription, 1)
		lobby.pubGamesReq <- pubRoomsReq
		expectedRoomDescs := []RoomDescription{
			{id: "id4", playersCount: 1, maxPlayers: 7, started: false},
		}
		assert.ElementsMatch(t, expectedRoomDescs, <-pubRoomsReq)
	})

	t.Run("Remove Room 4", func(t *testing.T) {
		lobby.removeRoomChan <- room4
		_, ok := <-room4.ticks
		assert.False(t, ok)
		pubRoomsReq := make(chan []RoomDescription, 1)
		lobby.pubGamesReq <- pubRoomsReq
		assert.Empty(t, <-pubRoomsReq)
	})

	mockIdgenerator.AssertExpectations(t)
	mockTickerCreator.AssertExpectations(t)
}
