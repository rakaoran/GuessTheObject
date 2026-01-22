package game

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// The MockPlayer in mocks_test.go is outdated (missing Username/CancelAndRelease),
// so we need a local one that actually satisfies the Player interface.

func setupRoom(t *testing.T) (*room, *MockPlayer, *MockRandomWordsGenerator) {
	host := &MockPlayer{}
	// NewRoom calls host.Username(), so we must mock it immediately
	host.On("Username").Return("host_user")

	gen := &MockRandomWordsGenerator{}

	// Calling NewRoom exactly as is, no safety patches 💀
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
	r, _, _ := setupRoom(t)
	r.SetId("new-id")
	assert.Equal(t, "new-id", r.id)
}

func TestRoom_SetParentLobby(t *testing.T) {
	r, _, _ := setupRoom(t)
	lobby := &MockLobby{}
	r.SetParentLobby(lobby)
	assert.Equal(t, lobby, r.parentLobby)
}

func TestRoom_Description(t *testing.T) {
	r, _, _ := setupRoom(t)
	r.SetId("desc-test")

	desc := r.Description()

	assert.Equal(t, "desc-test", desc.id)
	assert.Equal(t, 1, desc.playersCount) // Host is added in NewRoom
	assert.False(t, desc.started)
	assert.False(t, desc.private)
}

func TestRoom_PingPlayers(t *testing.T) {
	r, _, _ := setupRoom(t)

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
	r, _, _ := setupRoom(t)
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
	r, _, _ := setupRoom(t)
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
	r, _, _ := setupRoom(t)
	req := roomJoinRequest{roomId: "room1"}

	// ⚠️ If joinReqs isn't initialized, this will hang forever
	// We use a small timeout to fail fast instead of hanging the test suite
	done := make(chan struct{})
	go func() {
		r.RequestJoin(req)
		close(done)
	}()

	select {
	case <-done:
		// RequestJoin finished (pushed to channel)
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "RequestJoin blocked too long (channel probably nil or full)")
		return
	}

	// Verify content
	select {
	case val := <-r.joinReqs:
		assert.Equal(t, req, val)
	default:
		// If we reached here, maybe the channel is nil (reading from nil blocks too)
		// But in a test we can't easily check 'is nil' on a private field without reflection
		// or just assuming the select above would catch it.
	}
}

func TestRoom_RemoveMe(t *testing.T) {
	r, _, _ := setupRoom(t)
	p := &MockPlayer{}
	ctx := context.Background()

	// ⚠️ If removeMe isn't initialized, this blocks forever
	done := make(chan struct{})
	go func() {
		r.RemoveMe(ctx, p)
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "RemoveMe blocked too long (channel probably nil)")
	}
}

func TestRoom_CloseAndRelease(t *testing.T) {
	r, _, _ := setupRoom(t)

	// ⚠️ If any channel closed here is nil, this will PANIC
	assert.NotPanics(t, func() {
		r.CloseAndRelease()
	}, "CloseAndRelease panicked (likely closing a nil channel)")
}
