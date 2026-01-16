package game

import (
	"api/domain/protobuf"

	"golang.org/x/time/rate"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func NewPlayer(id, username string, socket WebsocketConnection) *Player {
	return &Player{
		id:          id,
		username:    username,
		rateLimiter: *rate.NewLimiter(1, 5),
		socket:      socket,
		inbox:       make(chan []byte, 256),
		pingChan:    make(chan struct{}),
	}
}

func (p *Player) ReadPump() {
	room := p.room

	for {
		data, err := p.socket.Read()

		if err != nil {
			break
		}

		fieldNum, _, n := protowire.ConsumeTag(data)

		if n < 0 {
			continue
		}

		envelope := ClientPacketEnvelope{from: p}

		if fieldNum == 1 { // = 1 tag in .proto
			envelope.rawBinary = data
		} else {
			envelope.clientPacket = &protobuf.ClientPacket{}
			if err := proto.Unmarshal(data, envelope.clientPacket); err != nil {
				continue
			}

		}

		room.inbox <- envelope
	}
}

func (p *Player) WritePump() {
loop:
	for {
		select {
		case data, ok := <-p.inbox:
			if !ok {
				break loop
			}
			err := p.socket.Write(data)
			if err != nil {
				break loop
			}
		case _, ok := <-p.pingChan:
			if !ok {
				break loop
			}
			err := p.socket.Ping()
			if err != nil {
				break loop
			}
		}
	}
}
