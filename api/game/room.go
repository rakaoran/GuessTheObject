package game

import "time"

type RoomPhase int

const (
	PHASE_PENDING RoomPhase = iota
	PHASE_CHOOSING_WORD
	PHASE_DRAWING
	PHASE_TURN_SUMMARY
	PHASE_LEADERBOARD
)

type RoomJoinRequest struct {
	player  *Player
	errChan chan string
}

func NewRoomJoinRequest(player *Player) RoomJoinRequest {
	return RoomJoinRequest{player: player, errChan: make(chan string, 1)}
}

type ClientPacketEnvelope struct {
	clientPacket *ClientPacket
	rawBinary    []byte
	from         *Player
}

type RoomConfigs struct {
	maxPlayers           int
	roundsCount          int
	choosingWordDuration time.Duration
	drawingDuration      time.Duration
}

type Room struct {
	// Identity / metadata
	id      string
	hostID  string
	private bool

	// configs
	configs RoomConfigs

	// Runtime state
	phase           RoomPhase
	round           int
	nextTick        time.Time
	drawerIndex     int
	currentWord     string
	bannedPlayerIds map[string]bool
	scoreIncrements map[*Player]int

	// Gameplay data
	wordChoices    []string
	drawingHistory [][]byte
	guessers       map[*Player]bool
	kickVotes      map[*Player]map[*Player]bool

	// Players
	players []*Player

	// Communication
	inbox                 chan ClientPacketEnvelope
	ticks                 chan struct{}
	playerRemovalRequests chan *Player
	joinRequests          chan RoomJoinRequest
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
