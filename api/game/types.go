package game

import (
	"api/domain"
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type WebsocketConnection interface {
	Close(errCode string)
	Write(data []byte) error
	Read() ([]byte, error)
	Ping() error
}

type UserGetter interface {
	GetUserById(ctx context.Context, id string) (domain.User, error)
}

type Player struct {
	id          string
	username    string
	score       int
	rateLimiter rate.Limiter
	socket      WebsocketConnection
	inbox       chan []byte
	pingChan    chan struct{}
	room        *Room
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

type Idgen struct {
	ids    map[string]struct{}
	locker sync.Mutex
}

type service struct {
	locker     sync.RWMutex
	rooms      map[string]*Room
	idGen      Idgen
	userGetter UserGetter
}
