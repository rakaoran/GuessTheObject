package game

import (
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

type Player struct {
	id          string
	username    string
	score       int
	rateLimiter RateLimiter
	socket      NetworkSession
	inbox       chan []byte
	pingChan    chan struct{}
	room        *Room
}

type RateLimiter struct {
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
			envelope.clientPacket = &ClientPacket{}
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
