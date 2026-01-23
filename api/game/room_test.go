package game

import (
	"api/domain/protobuf"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupRoom() (*room, *MockPlayer, *MockRandomWordsGenerator) {
	host := &MockPlayer{}
	host.On("Username").Return("host_user")
	host.On("SetRoom", mock.Anything).Return()

	gen := &MockRandomWordsGenerator{}

	r := NewRoom(
		host,
		false,
		10,
		3,
		3,
		time.Minute,
		time.Minute,
		gen,
	)

	return r, host, gen
}

func TestRoom_SetId(t *testing.T) {
	r, _, _ := setupRoom()
	r.SetId("new-id")
	assert.Equal(t, "new-id", r.id)
}

func TestRoom_SetParentLobby(t *testing.T) {
	r, _, _ := setupRoom()
	lobby := &MockLobby{}
	r.SetParentLobby(lobby)
	assert.Equal(t, lobby, r.parentLobby)
}

func TestRoom_Description(t *testing.T) {
	r, _, _ := setupRoom()
	r.SetId("desc-test")

	desc := r.Description()

	assert.Equal(t, "desc-test", desc.id)
	assert.Equal(t, 1, desc.playersCount) // Host is added in NewRoom
	assert.False(t, desc.started)
	assert.False(t, desc.private)
}

func TestRoom_PingPlayers(t *testing.T) {
	r, _, _ := setupRoom()

	// This should be non-blocking
	r.PingPlayers()

	// Verify it actually went into the channel
	select {
	case <-r.pingPlayers:
		// success
	default:
		assert.Fail(t, "Signal was not sent to pingPlayers channel")
	}
}

func TestRoom_Tick(t *testing.T) {
	r, _, _ := setupRoom()
	now := time.Now()

	r.Tick(now)

	select {
	case val := <-r.ticks:
		assert.Equal(t, now, val)
	default:
		assert.Fail(t, "Time signal was not sent to ticks channel")
	}
}

func TestRoom_Send(t *testing.T) {
	r, _, _ := setupRoom()
	ctx := context.Background()
	envelope := ClientPacketEnvelope{from: "user1"}

	r.Send(ctx, envelope)

	select {
	case val := <-r.inbox:
		assert.Equal(t, envelope, val)
	default:
		assert.Fail(t, "Envelope was not sent to inbox")
	}
}

func TestRoom_RequestJoin(t *testing.T) {
	r, _, _ := setupRoom()
	req := roomJoinRequest{roomId: "room1"}

	done := make(chan struct{})
	go func() {
		r.RequestJoin(req)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "RequestJoin blocked too long (channel probably nil or full)")
		return
	}

	assert.Equal(t, req, <-r.joinReqs)
}

func TestRoom_RemoveMe(t *testing.T) {
	r, _, _ := setupRoom()
	p := &MockPlayer{}
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		r.RemoveMe(ctx, p)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "RemoveMe blocked too long (channel probably nil)")
	}
}

func TestRoom_CloseAndRelease(t *testing.T) {
	r, _, _ := setupRoom()

	assert.NotPanics(t, func() {
		r.CloseAndRelease()
	}, "CloseAndRelease panicked (likely closing a nil channel)")

	_, ok := <-r.pingPlayers

	assert.Falsef(t, ok, "Channel is still non closed")
}

func TestRoom_GameLoop_Close_And_Release(t *testing.T) {
	r, _, _ := setupRoom()
	wg := sync.WaitGroup{}

	wg.Go(func() { r.GameLoop() })
	r.CloseAndRelease()
	wg.Wait()
}

func TestRoom_GameLoop_Reads_Ticks_And_Updates_Phase(t *testing.T) {
	r, p, wgen := setupRoom()
	wgen.On("Generate", r.wordsCount).Return([]string{"word1", "word2", "word3"})
	p.On("Send", mock.Anything).Return(nil)
	assert.Equal(t, PHASE_PENDING, r.phase)

	go r.GameLoop()

	r.phase = PHASE_TURN_SUMMARY

	futureTime := time.Now().Add(20 * time.Minute)
	r.ticks <- futureTime

	assert.Eventually(t, func() bool {
		return r.phase == PHASE_CHOOSING_WORD
	}, time.Second, 50*time.Millisecond, "GameLoop should read tick and update phase")
	wgen.AssertExpectations(t)
	p.AssertExpectations(t)
}

func TestRoom_GameLoop_Reads_Ping_And_Queues_Task(t *testing.T) {
	r, p, _ := setupRoom()
	p.On("Ping").Return(nil)
	go r.GameLoop()
	r.PingPlayers()
	time.Sleep(time.Millisecond * 50)
	p.AssertExpectations(t)
}

func TestRoom_GameLoop_Inbox_Sends_Data(t *testing.T) {
	r, host, _ := setupRoom()
	lobby := &MockLobby{}
	lobby.On("RequestUpdateDescription", mock.Anything).Return()
	r.SetParentLobby(lobby)

	host.On("Send", mock.Anything).Return(nil)

	go r.GameLoop()

	envelope := ClientPacketEnvelope{
		from: host.Username(),
		clientPacket: &protobuf.ClientPacket{
			Payload: &protobuf.ClientPacket_StartGame_{
				StartGame: &protobuf.ClientPacket_StartGame{},
			},
		},
	}
	r.inbox <- envelope

	time.Sleep(50 * time.Millisecond)
	host.AssertExpectations(t)
}

func TestRoom_GameLoop_Player_Removal(t *testing.T) {
	// Setup
	r, host, _ := setupRoom()

	lobby := &MockLobby{}
	lobby.On("RequestUpdateDescription", mock.Anything).Return()
	r.SetParentLobby(lobby)

	victim := &MockPlayer{}
	victim.On("Username").Return("victim_user")
	victim.On("SetRoom", r).Return()

	r.addPlayer(victim)

	host.On("Send", mock.Anything).Return(nil)
	victim.On("Send", mock.Anything).Return(nil)
	// Victim should be cancelled
	victim.On("CancelAndRelease").Return()

	// 3. Start the loop
	go r.GameLoop()

	// 4. Action: Trigger removal
	r.playerRemovalRequests <- victim

	// 5. Assert
	time.Sleep(50 * time.Millisecond)
	host.AssertExpectations(t)
	victim.AssertExpectations(t)
}
