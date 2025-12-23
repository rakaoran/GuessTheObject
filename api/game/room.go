package game

import "time"

type RoomPhase int

const (
	STATE_PENDING RoomPhase = iota
	STATE_CHOOSING_WORD
	STATE_DRAWING
	STATE_TURN_SUMMARY
	STATE_LEADERBOARD
)

type RoomJoinRequest struct {
	player  *Player
	errChan chan string
}

type ClientPacketEnvelope struct {
	clientPacket ClientPacket
	rawBinary    []byte
	from         *Player
}

type Room struct {
	// Identity / metadata
	id     string
	name   string
	phase  RoomPhase
	hostID string

	// Configuration (static-ish)
	maxPlayers           int
	roundsCount          int
	choosingWordDuration time.Duration
	drawingDuration      time.Duration
	turnSummaryDuration  time.Duration

	// Runtime state
	round           int
	nextTick        time.Time
	drawerIndex     int
	currentWord     string
	bannedIds       map[string]bool
	scoreIncrements map[*Player]int

	// Gameplay data
	wordChoices    []string
	drawingHistory [][]byte
	guessers       map[string]bool
	kickVotes      map[*Player]map[*Player]bool

	// Players
	players []*Player

	// Communication
	inbox                 chan ClientPacketEnvelope
	ticks                 chan struct{}
	playerRemovalRequests chan *Player
	joinRequests          chan RoomJoinRequest
}
