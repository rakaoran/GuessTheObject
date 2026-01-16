package game

import "api/domain/protobuf"

const (
	PHASE_PENDING RoomPhase = iota
	PHASE_CHOOSING_WORD
	PHASE_DRAWING
	PHASE_TURN_SUMMARY
	PHASE_LEADERBOARD
)

type RoomPhase int

type RoomJoinRequest struct {
	player  *Player
	errChan chan string
}

type ClientPacketEnvelope struct {
	clientPacket *protobuf.ClientPacket
	rawBinary    []byte
	from         *Player
}

func NewRoomJoinRequest(player *Player) RoomJoinRequest {
	return RoomJoinRequest{player: player, errChan: make(chan string, 1)}
}

func NewRoom(id string, player *Player, configs RoomConfigs, private bool) *Room {
	room := &Room{
		id:                    id,
		hostID:                player.id,
		private:               private,
		configs:               configs,
		phase:                 PHASE_PENDING,
		players:               make([]*Player, 0, configs.maxPlayers),
		bannedPlayerIds:       make(map[string]bool),
		scoreIncrements:       make(map[*Player]int),
		guessers:              make(map[*Player]bool),
		kickVotes:             make(map[*Player]map[*Player]bool),
		inbox:                 make(chan ClientPacketEnvelope, 1024),
		playerRemovalRequests: make(chan *Player, 64),
		ticks:                 make(chan struct{}, 24),
		joinRequests:          make(chan RoomJoinRequest),
	}
	room.players = append(room.players, player)
	player.room = room
	return room
}
