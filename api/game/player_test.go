package game

import (
	"api/domain/protobuf"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func AssertProtoEq(t *testing.T, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	diff := cmp.Diff(expected, actual, protocmp.Transform())
	if diff != "" {
		assert.Fail(t, "Protobuf mismatch (-want +got):\n"+diff, msgAndArgs...)
	}
}

func TestReadPump(t *testing.T) {
	t.Parallel()
	t.Run("Read Error", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		player := NewPlayer("id", "username")
		mockSocket.On("Read").Return([]byte{}, assert.AnError)
		mockSocket.On("Close").Return()
		removeMe := make(chan *Player, 1)
		player.removeMe = removeMe
		wg := sync.WaitGroup{}
		wg.Go(func() {
			player.ReadPump(mockSocket)
		})
		// on read error, the goroutine must release
		wg.Wait()

		assert.Equal(t, player, <-removeMe)
		mockSocket.AssertExpectations(t)
	})

	t.Run("Read Error With Context Cancelation", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		player := NewPlayer("id", "username")
		mockSocket.On("Read").Return([]byte{}, assert.AnError)
		mockSocket.On("Close").Return()
		player.removeMe = nil
		wg := sync.WaitGroup{}
		wg.Go(func() {
			player.ReadPump(mockSocket)
		})
		// on cancel, the goroutine must release
		player.cancelCtx()
		wg.Wait()
		mockSocket.AssertExpectations(t)
	})

	t.Run("Blocked Room Write With Context Cancelation", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		player := NewPlayer("id", "username")
		player.roomChan = make(chan ClientPacketEnvelope)
		clientPacket := &protobuf.ClientPacket{
			Payload: &protobuf.ClientPacket_DrawingData{
				DrawingData: &protobuf.DrawingData{
					Data: []byte{1, 2, 3},
				},
			},
		}
		marshaledClientPacket, _ := proto.Marshal(clientPacket)
		mockSocket.On("Read").Return(marshaledClientPacket, nil)
		mockSocket.On("Close").Return()
		wg := sync.WaitGroup{}
		wg.Go(func() {
			player.ReadPump(mockSocket)
		})
		player.cancelCtx()
		// on cancel, the goroutine must release
		wg.Wait()
		mockSocket.AssertExpectations(t)
	})

	t.Run("Read garbage data", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		player := NewPlayer("id", "username")
		marshaledClientPacket := []byte{1, 5}
		mockSocket.On("Read").Return(marshaledClientPacket, nil).Once()
		mockSocket.On("Read").Return(marshaledClientPacket, assert.AnError).Once()
		mockSocket.On("Close").Return()
		roomchan := make(chan ClientPacketEnvelope, 1)
		player.roomChan = roomchan
		removeMe := make(chan *Player, 1)
		player.removeMe = removeMe
		wg := sync.WaitGroup{}
		wg.Go(func() {
			player.ReadPump(mockSocket)
		})
		wg.Wait()

		assert.Empty(t, roomchan)
		assert.Equal(t, player, <-removeMe)
		mockSocket.AssertExpectations(t)
	})

	t.Run("Read good data", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		player := NewPlayer("id", "username")

		clientPacket := &protobuf.ClientPacket{
			Payload: &protobuf.ClientPacket_DrawingData{
				DrawingData: &protobuf.DrawingData{
					Data: []byte{1, 2, 3},
				},
			},
		}
		marshaledClientPacket, _ := proto.Marshal(clientPacket)
		mockSocket.On("Read").Return(marshaledClientPacket, nil).Once()
		mockSocket.On("Read").Return(marshaledClientPacket, assert.AnError).Once()
		mockSocket.On("Close").Return()
		roomchan := make(chan ClientPacketEnvelope, 1)
		player.roomChan = roomchan
		player.removeMe = make(chan *Player, 1)
		wg := sync.WaitGroup{}
		wg.Go(func() {
			player.ReadPump(mockSocket)
		})
		wg.Wait()

		require.Len(t, roomchan, 1)
		envelope := <-roomchan
		require.Equal(t, player, envelope.from)
		AssertProtoEq(t, clientPacket, envelope.clientPacket)

		mockSocket.AssertExpectations(t)
	})

	t.Run("Spam Messages Rate Limiting", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		player := NewPlayer("id", "username")

		clientPacket := &protobuf.ClientPacket{
			Payload: &protobuf.ClientPacket_PlayerMessage_{
				PlayerMessage: &protobuf.ClientPacket_PlayerMessage{
					Message: "spam spamm",
				},
			},
		}
		marshaledClientPacket, _ := proto.Marshal(clientPacket)
		mockSocket.On("Read").Return(marshaledClientPacket, nil).Times(50)
		mockSocket.On("Read").Return(marshaledClientPacket, assert.AnError).Once()
		mockSocket.On("Close").Return()
		roomchan := make(chan ClientPacketEnvelope, 50)
		player.roomChan = roomchan
		player.removeMe = make(chan *Player, 1)
		wg := sync.WaitGroup{}
		wg.Go(func() {
			player.ReadPump(mockSocket)
		})
		wg.Wait()

		require.Len(t, roomchan, 5)
		envelope := <-roomchan
		require.Equal(t, player, envelope.from)
		AssertProtoEq(t, clientPacket, envelope.clientPacket)

		mockSocket.AssertExpectations(t)
	})

	t.Run("Stuff like drawing data doesn't get rate limited", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		player := NewPlayer("id", "username")

		clientPacket := &protobuf.ClientPacket{
			Payload: &protobuf.ClientPacket_DrawingData{
				DrawingData: &protobuf.DrawingData{
					Data: []byte{1, 2, 3},
				},
			},
		}
		marshaledClientPacket, _ := proto.Marshal(clientPacket)
		mockSocket.On("Read").Return(marshaledClientPacket, nil).Times(50)
		mockSocket.On("Read").Return(marshaledClientPacket, assert.AnError).Once()
		mockSocket.On("Close").Return()
		roomchan := make(chan ClientPacketEnvelope, 60)
		player.roomChan = roomchan
		player.removeMe = make(chan *Player, 1)
		wg := sync.WaitGroup{}
		wg.Go(func() {
			player.ReadPump(mockSocket)
		})
		wg.Wait()

		require.Len(t, roomchan, 50)
		envelope := <-roomchan
		require.Equal(t, player, envelope.from)
		AssertProtoEq(t, clientPacket, envelope.clientPacket)

		mockSocket.AssertExpectations(t)
	})

}

func TestWritePump(t *testing.T) {
	t.Parallel()

	t.Run("Inbox Closing Must Release The Goroutine", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		mockSocket.On("Close").Return().Once()
		player := NewPlayer("id", "username")
		wg := sync.WaitGroup{}
		wg.Go(func() {
			player.WritePump(mockSocket)
		})
		close(player.inbox)
		wg.Wait()
		mockSocket.AssertExpectations(t)
	})

	t.Run("Ping Channel Closing Must Release The Goroutine", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		mockSocket.On("Close").Return().Once()
		player := NewPlayer("id", "username")
		wg := sync.WaitGroup{}
		wg.Go(func() {
			player.WritePump(mockSocket)
		})
		close(player.pingChan)
		wg.Wait()
		mockSocket.AssertExpectations(t)
	})

	t.Run("Context Cancelation Must Release The Goroutine", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		mockSocket.On("Close").Return().Once()
		player := NewPlayer("id", "username")
		wg := sync.WaitGroup{}
		wg.Go(func() {
			player.WritePump(mockSocket)
		})
		player.cancelCtx()
		wg.Wait()
		mockSocket.AssertExpectations(t)
	})

	t.Run("Write Error Must Notify Room Then Release The Goroutine", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		data := []byte{1, 2, 3}
		mockSocket.On("Close").Return().Once()
		mockSocket.On("Write", data).Return(assert.AnError).Once()
		player := NewPlayer("id", "username")
		removeMe := make(chan *Player, 1)
		player.removeMe = removeMe
		wg := sync.WaitGroup{}
		wg.Go(func() {
			player.WritePump(mockSocket)
		})
		player.inbox <- data
		wg.Wait()
		assert.Equal(t, player, <-removeMe)
		mockSocket.AssertExpectations(t)
	})

	t.Run("Correct Data Writing", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		data := []byte{1, 2, 3}
		mockSocket.On("Write", data).Return(nil).Once()
		mockSocket.On("Write", data).Return(assert.AnError).Once()
		mockSocket.On("Close").Return().Once()
		player := NewPlayer("id", "username")
		player.removeMe = make(chan *Player, 1)
		wg := sync.WaitGroup{}
		wg.Go(func() {
			player.WritePump(mockSocket)
		})
		player.inbox <- data
		player.inbox <- data
		wg.Wait()
		mockSocket.AssertExpectations(t)
	})

	t.Run("Correct Ping Handling", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		mockSocket.On("Ping").Return(nil).Once()
		mockSocket.On("Ping").Return(assert.AnError).Once()
		mockSocket.On("Close").Return().Once()
		player := NewPlayer("id", "username")
		player.removeMe = make(chan *Player, 1)
		wg := sync.WaitGroup{}
		wg.Go(func() {
			player.WritePump(mockSocket)
		})
		player.pingChan <- struct{}{}
		player.pingChan <- struct{}{}
		wg.Wait()
		mockSocket.AssertExpectations(t)
	})

}
