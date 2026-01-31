package game

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRoom_MaxPlayers_Join_Scenario(t *testing.T) {
	// Setup a room with MaxPlayers = 5
	host := &MockPlayer{}
	host.On("Username").Return("host_user")
	host.On("SetRoom", mock.Anything).Return()
	host.On("Send", mock.Anything).Return(nil) // Host receives updates

	gen := &MockRandomWordsGenerator{}

	// MaxPlayers = 5
	r := NewRoom(
		host,
		false,
		5, // MaxPlayers
		3,
		3,
		time.Minute,
		time.Minute,
		gen,
	)

	// We need to simulate the game loop running because addPlayer might interact with channels
	// However, addPlayer is synchronous in current implementation (mostly),
	// but it does broadcast which puts things in channels.
	// For this unit test, we might not strictly need the loop if we just check state,
	// BUT the issue might be related to channel blocking if the loop isn't draining them.
	// So let's start the loop.
	go r.GameLoop()
	defer r.CloseAndRelease()

	// Add 4 more players (Total 5)
	players := []*MockPlayer{}
	for i := 0; i < 4; i++ {
		p := &MockPlayer{}
		username := "player_" + string(rune('A'+i)) // player_A, player_B ...
		p.On("Username").Return(username)
		p.On("SetRoom", mock.Anything).Return()
		p.On("Send", mock.Anything).Return(nil) // Should receive updates
		// Crucially, we want to ensure CancelAndRelease is NOT called for successful joins
		p.On("CancelAndRelease").Run(func(args mock.Arguments) {
			t.Logf("CancelAndRelease called for %s", username)
		}).Return().Maybe()

		players = append(players, p)

		// Join
		err := r.addPlayer(p)
		assert.NoError(t, err, "Player %s should join successfully", username)

		// Give a tiny bit of time for async operations if any (though addPlayer broadcast is sync-ish to channel)
		time.Sleep(10 * time.Millisecond)
	}

	// Verify state
	assert.Equal(t, 5, len(r.playerStates), "Room should have 5 players")

	// Check if the last player ("player_D", the 5th one) is still in the list
	// and hasn't been removed (which would trigger CancelAndRelease in a real scenario if triggered by handleRemovePlayer)

	// In the real code, if addPlayer logic is flawed regarding the snapshot,
	// the player MIGHT be added to state but receive a snapshot that excludes them,
	// leading to CLIENT side disconnect.
	// OR, if the channel is full, they might be dropped.

	// Let's verify that the 5th player (players[3]) received the InitialRoomSnapshot.
	// faster way is to check the calls.
	lastPlayer := players[3]
	lastPlayer.AssertCalled(t, "Send", mock.Anything)
	// In a real failure where backend boots them, we'd expect RemovePlayer to be called or an error.

	// Wait a bit to ensure no async removal happens
	time.Sleep(100 * time.Millisecond)

	// Ensure no CancelAndRelease was called on the last player (using AssertNotCalled if we removed the Maybe/Run above, but let's check manually if we can)
	// mostly we just want to ensure they represent in the state.
	found := false
	for _, ps := range r.playerStates {
		if ps.username == "player_D" {
			found = true
			break
		}
	}
	assert.True(t, found, "5th player should still be in the room state")
}
