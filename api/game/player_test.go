package game

import (
	"api/domain/protobuf"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		mockRoom := &MockRoom{}
		p := NewPlayer("id", "username")
		mockRoom.On("RemoveMe", p.ctx, p).Return()
		p.SetRoom(mockRoom)
		mockSocket.On("Read").Return([]byte{}, assert.AnError)
		mockSocket.On("Close").Return()
		wg := sync.WaitGroup{}
		wg.Go(func() {
			p.ReadPump(mockSocket)
		})
		// on read error, the goroutine must release
		wg.Wait()

		mockSocket.AssertExpectations(t)
	})

	t.Run("Read Error With Context Cancelation", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		mockRoom := &MockRoom{}
		p := NewPlayer("id", "username")
		p.SetRoom(mockRoom)
		mockRoom.On("RemoveMe", p.ctx, p).Return()
		mockSocket.On("Read").Return([]byte{}, assert.AnError)
		mockSocket.On("Close").Return()
		wg := sync.WaitGroup{}
		wg.Go(func() {
			p.ReadPump(mockSocket)
		})
		// on cancel, the goroutine must release
		p.cancelCtx()
		wg.Wait()
		mockSocket.AssertExpectations(t)
	})

	t.Run("Blocked Room Write With Context Cancelation", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		p := NewPlayer("id", "username")
		clientPacket := &protobuf.ClientPacket{
			Payload: &protobuf.ClientPacket_DrawingData{
				DrawingData: &protobuf.DrawingData{
					Data: []byte{1, 2, 3},
				},
			},
		}
		mockRoom := &MockRoom{}
		p.SetRoom(mockRoom)
		mockRoom.On("RemoveMe", p.ctx, p).Return()
		marshaledClientPacket, _ := proto.Marshal(clientPacket)
		mockRoom.On("Send", p.ctx, mock.Anything).Run(func(args mock.Arguments) {
			envlp, ok := args.Get(1).(ClientPacketEnvelope)
			assert.True(t, ok)
			AssertProtoEq(t, envlp.clientPacket, clientPacket)
			assert.Equal(t, p.username, envlp.from)
			select {
			case <-time.After(time.Second):
			case <-args.Get(0).(context.Context).Done():
				return
			}
		}).Return()

		mockSocket.On("Read").Return(marshaledClientPacket, nil)
		mockSocket.On("Close").Return()
		wg := sync.WaitGroup{}
		wg.Go(func() {
			p.ReadPump(mockSocket)
		})
		p.CancelAndRelease()
		// on cancel, the goroutine must release
		wg.Wait()
		mockSocket.AssertExpectations(t)
	})

	t.Run("Read garbage data", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		p := NewPlayer("id", "username")
		mockRoom := &MockRoom{}
		p.SetRoom(mockRoom)
		mockRoom.On("RemoveMe", p.ctx, p).Return()
		marshaledClientPacket := []byte{1, 5}
		mockSocket.On("Read").Return(marshaledClientPacket, nil).Once()
		mockSocket.On("Read").Return(marshaledClientPacket, assert.AnError).Once()
		mockSocket.On("Close").Return()
		wg := sync.WaitGroup{}
		wg.Go(func() {
			p.ReadPump(mockSocket)
		})
		wg.Wait()
		mockSocket.AssertExpectations(t)
	})

	t.Run("Read good data", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		p := NewPlayer("id", "username")
		mockRoom := &MockRoom{}
		p.SetRoom(mockRoom)
		mockRoom.On("RemoveMe", p.ctx, p).Return()
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

		mockRoom.On("Send", p.ctx, mock.AnythingOfType("ClientPacketEnvelope")).Run(func(args mock.Arguments) {
			envlp, ok := args.Get(1).(ClientPacketEnvelope)
			assert.True(t, ok)
			AssertProtoEq(t, envlp.clientPacket, clientPacket)
			assert.Equal(t, p.username, envlp.from)
		})

		wg := sync.WaitGroup{}
		wg.Go(func() {
			p.ReadPump(mockSocket)
		})
		wg.Wait()

		mockSocket.AssertExpectations(t)
	})

	t.Run("Spam Messages Rate Limiting", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		p := NewPlayer("id", "username")

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

		mockRoom := &MockRoom{}
		p.SetRoom(mockRoom)
		mockRoom.On("Send", p.ctx, mock.Anything).Times(5).Return()
		mockRoom.On("RemoveMe", p.ctx, p)
		wg := sync.WaitGroup{}
		wg.Go(func() {
			p.ReadPump(mockSocket)
		})
		wg.Wait()

		mockSocket.AssertExpectations(t)
	})

	t.Run("Stuff like drawing data doesn't get rate limited", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		p := NewPlayer("id", "username")

		clientPacket := &protobuf.ClientPacket{
			Payload: &protobuf.ClientPacket_DrawingData{
				DrawingData: &protobuf.DrawingData{
					Data: []byte{1, 2, 3},
				},
			},
		}
		mockRoom := &MockRoom{}
		mockRoom.On("Send", p.ctx, mock.Anything).Times(50).Return()
		mockRoom.On("RemoveMe", p.ctx, p)
		p.SetRoom(mockRoom)
		marshaledClientPacket, _ := proto.Marshal(clientPacket)
		mockSocket.On("Read").Return(marshaledClientPacket, nil).Times(50)
		mockSocket.On("Read").Return(marshaledClientPacket, assert.AnError).Once()
		mockSocket.On("Close").Return()

		wg := sync.WaitGroup{}
		wg.Go(func() {
			p.ReadPump(mockSocket)
		})
		wg.Wait()

		mockSocket.AssertExpectations(t)
	})

}

func TestWritePump(t *testing.T) {
	t.Parallel()

	t.Run("Canceling and Releasing Must Release The Goroutine", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		mockSocket.On("Close").Return().Once()
		p := NewPlayer("id", "username")
		wg := sync.WaitGroup{}
		wg.Go(func() {
			p.WritePump(mockSocket)
		})
		p.CancelAndRelease()
		wg.Wait()
		mockSocket.AssertExpectations(t)
	})

	t.Run("Write Error Must Notify Room Then Release The Goroutine", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		mockRoom := &MockRoom{}

		data := []byte{1, 2, 3}

		// Expect Write to fail
		mockSocket.On("Write", data).Return(assert.AnError).Once()
		mockSocket.On("Close").Return().Once()

		p := NewPlayer("id", "username")
		p.SetRoom(mockRoom)

		// Expect Room.RemoveMe to be called on write failure
		mockRoom.On("RemoveMe", p.ctx, p).Return()

		wg := sync.WaitGroup{}
		wg.Go(func() {
			p.WritePump(mockSocket)
		})

		p.Send(data)
		wg.Wait()

		mockSocket.AssertExpectations(t)
		mockRoom.AssertExpectations(t)
	})

	t.Run("Correct Data Writing", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}
		data := []byte{1, 2, 3}

		// First write succeeds, second fails to trigger exit
		mockSocket.On("Write", data).Return(nil).Once()
		mockSocket.On("Write", data).Return(assert.AnError).Once()
		mockSocket.On("Close").Return().Once()

		p := NewPlayer("id", "username")
		mockRoom := &MockRoom{}
		p.SetRoom(mockRoom)

		// Room removal on the second (failed) write
		mockRoom.On("RemoveMe", p.ctx, p).Return()

		wg := sync.WaitGroup{}
		wg.Go(func() {
			p.WritePump(mockSocket)
		})
		p.Send(data)
		p.Send(data)
		wg.Wait()

		mockSocket.AssertExpectations(t)
	})

	t.Run("Correct Ping Writing", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}

		mockSocket.On("Ping").Return(nil).Once()
		mockSocket.On("Close").Return().Once()

		p := NewPlayer("id", "username")
		mockRoom := &MockRoom{}
		p.SetRoom(mockRoom)

		wg := sync.WaitGroup{}
		wg.Go(func() {
			p.WritePump(mockSocket)
		})

		p.Ping()
		close(p.pingChan)
		wg.Wait()

		mockSocket.AssertExpectations(t)
	})

	t.Run("Ping Writing Must Release", func(t *testing.T) {
		t.Parallel()
		mockSocket := &MockWebsocketConnection{}

		mockSocket.On("Ping").Return(assert.AnError).Once()
		mockSocket.On("Close").Return().Once()

		p := NewPlayer("id", "username")
		mockRoom := &MockRoom{}
		mockRoom.On("RemoveMe", p.ctx, p).Return()
		p.SetRoom(mockRoom)

		wg := sync.WaitGroup{}
		wg.Go(func() {
			p.WritePump(mockSocket)
		})

		p.Ping()
		wg.Wait()

		mockSocket.AssertExpectations(t)
	})
}
