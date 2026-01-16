package game

import (
	"api/domain/protobuf"

	"google.golang.org/protobuf/proto"
)

func (p *Player) ReadeadPump(socket WebsocketConnection) {
	for {
		data, err := socket.Read()
		if err != nil {
			return
		}
		packet := &protobuf.ClientPacket{}
		err = proto.Unmarshal(data, packet)
		envelope := ClientPacketEnvelope{clientPacket: packet, from: p}
		p.roomChan <- envelope
	}
}

func (p *Player) WritePump(socket WebsocketConnection) {
	for {
		marshalledServerPacket, ok := <-p.inbox
		if !ok {
			return
		}

		socket.Write(marshalledServerPacket)
	}
}
