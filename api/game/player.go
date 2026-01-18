package game

import (
	"api/domain/protobuf"
	"context"

	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
)

func NewPlayer(id string, username string) *Player {
	ctx, cancel := context.WithCancel(context.Background())
	return &Player{
		id:          id,
		username:    username,
		rateLimiter: *rate.NewLimiter(rate.Limit(2), 5),
		pingChan:    make(chan struct{}, 1),
		inbox:       make(chan []byte, 1024),
		ctx:         ctx,
		cancelCtx:   cancel,
	}
}

func (p *Player) ReadPump(socket WebsocketConnection) {
	defer socket.Close()
	for {
		data, err := socket.Read()
		if err != nil {
			select {
			case p.removeMe <- p:
				return
			case <-p.ctx.Done():
				return
			}
		}
		packet := &protobuf.ClientPacket{}
		err = proto.Unmarshal(data, packet)
		if err != nil {
			// TODO
			continue
		}
		if _, ok := packet.Payload.(*protobuf.ClientPacket_PlayerMessage_); ok {
			if !p.rateLimiter.Allow() {
				continue
			}
		}
		envelope := ClientPacketEnvelope{clientPacket: packet, from: p}
		select {
		case p.roomChan <- envelope:
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Player) WritePump(socket WebsocketConnection) {
	defer socket.Close()
	for {
		select {
		case marshalledServerPacket, ok := <-p.inbox:
			if !ok {
				return
			}
			err := socket.Write(marshalledServerPacket)
			if err != nil {
				select {
				case p.removeMe <- p:
					return
				case <-p.ctx.Done():
					return
				}
			}
		case _, ok := <-p.pingChan:
			if !ok {
				return
			}
			err := socket.Ping()
			if err != nil {
				select {
				case p.removeMe <- p:
					return
				case <-p.ctx.Done():
					return
				}
			}
		case <-p.ctx.Done():
			return
		}
	}
}
