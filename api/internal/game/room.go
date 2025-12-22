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
	round       int
	nextTick    time.Time
	drawerIndex int
	currentWord string
	bannedIds   map[string]bool

	// Gameplay data
	wordChoices    []string
	drawingHistory [][]byte
	guessers       map[string]bool
	kickVotes      map[*Player]map[*Player]bool

	// Players
	players []*Player

	// Communication
	inbox chan ClientPacket
	ticks chan struct{}
}
