package game

import (
	"api/domain"
	"api/domain/protobuf"
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RoomPhase int

type WebsocketConnection interface {
	Close(errCode string)
	Write(data []byte) error
	Read() ([]byte, error)
	Ping() error
}
type RandomWordsGenerator interface {
	Generate(count int) []string
}
type UserGetter interface {
	GetUserById(ctx context.Context, id string) (domain.User, error)
}

type Player struct {
	id             string
	username       string
	score          int
	scoreIncrement int
	hasGuessed     bool
	rateLimiter    rate.Limiter
	inbox          chan []byte
	pingChan       chan struct{}
	roomChan       chan<- ClientPacketEnvelope
}

type ClientPacketEnvelope struct {
	clientPacket *protobuf.ClientPacket
	from         *Player
}

type RoomJoinRequest struct {
	player  *Player
	errChan chan error
}

type Room struct {
	private               bool
	id                    string
	host                  *Player
	players               []*Player
	drawerIndex           int
	currentDrawer         *Player
	maxPlayers            int
	roundsCount           int
	wordsCount            int
	phase                 RoomPhase
	round                 int
	nextTick              time.Time
	choosingWordDuration  time.Duration
	drawingDuration       time.Duration
	currentWord           string
	wordChoices           []string
	drawingHistory        [][]byte
	inbox                 chan ClientPacketEnvelope
	ticks                 chan time.Time
	playerRemovalRequests chan *Player
	joinRequests          chan RoomJoinRequest

	randomWordsGenerator RandomWordsGenerator
}

type Idgen struct {
	ids    map[string]struct{}
	locker sync.Mutex
}
