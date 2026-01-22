package game

import (
	"api/domain/protobuf"
	"context"

	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
)

func NewPlayer(id string, username string) *player {
	ctx, cancel := context.WithCancel(context.Background())
	return &player{
		id:          id,
		username:    username,
		rateLimiter: *rate.NewLimiter(rate.Limit(2), 5),
		pingChan:    make(chan struct{}, 1),
		inbox:       make(chan []byte, 1024),
		ctx:         ctx,
		cancelCtx:   cancel,
	}
}

func (p *player) ReadPump(socket WebsocketConnection) {
	defer socket.Close()
	defer p.cancelCtx()
	for {

		data, err := socket.Read()
		if err != nil {
			p.room.RemoveMe(p.ctx, p)
			return
		}
		packet := &protobuf.ClientPacket{}
		err = proto.Unmarshal(data, packet)
		if err != nil {
			continue
		}
		if _, ok := packet.Payload.(*protobuf.ClientPacket_PlayerMessage_); ok {
			if !p.rateLimiter.Allow() {
				continue
			}
		}
		envelope := ClientPacketEnvelope{clientPacket: packet, from: p.username}
		p.room.Send(p.ctx, envelope)
		select {
		case <-p.ctx.Done():
			return
		default:
		}
	}
}

func (p *player) WritePump(socket WebsocketConnection) {
	defer socket.Close()
	defer p.cancelCtx()
	for {
		select {
		case marshalledServerPacket, ok := <-p.inbox:
			if !ok {
				return
			}
			err := socket.Write(marshalledServerPacket)
			if err != nil {
				p.room.RemoveMe(p.ctx, p)
				return
			}
		case _, ok := <-p.pingChan:
			if !ok {
				return
			}
			err := socket.Ping()
			if err != nil {
				p.room.RemoveMe(p.ctx, p)
				return
			}
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *player) Send(data []byte) error {
	select {
	case p.inbox <- data:
		return nil
	default:
		return ErrSendBufferFull
	}

}
func (p *player) Ping() error {
	select {
	case p.pingChan <- struct{}{}:
		return nil
	default:
		return ErrSendBufferFull
	}
}
func (p *player) SetRoom(r Room) {
	p.room = r
}
func (p *player) CancelAndRelease() {
	close(p.inbox)
	close(p.pingChan)
	p.cancelCtx()
}

func (p *player) Username() string {
	return p.username
}
