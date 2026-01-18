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
	Close()
	Write(data []byte) error
	Read() ([]byte, error)
	Ping() error
}
type RandomWordsGenerator interface {
	Generate(count int) []string
}
type UniqueIdGenerator interface {
	Generate() string
	Dispose(word string)
}

type PeriodicTickerChannelCreator interface {
	Create(duration time.Duration) <-chan time.Time
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
	roomChan       chan ClientPacketEnvelope
	removeMe       chan *Player
	ctx            context.Context
	cancelCtx      context.CancelFunc
}

type ClientPacketEnvelope struct {
	clientPacket *protobuf.ClientPacket
	from         *Player
}

type RoomJoinRequest struct {
	roomId  string
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
	guessersCount         int
	round                 int
	nextTick              time.Time
	choosingWordDuration  time.Duration
	drawingDuration       time.Duration
	currentWord           string
	wordChoices           []string
	drawingHistory        [][]byte
	inbox                 chan ClientPacketEnvelope
	ticks                 chan time.Time
	pingPlayers           chan struct{}
	playerRemovalRequests chan *Player
	joinRequests          chan RoomJoinRequest
	updateDescriptionChan chan RoomDescription
	removeMe              chan *Room
	randomWordsGenerator  RandomWordsGenerator
}

type RoomDescription struct {
	id           string
	playersCount int
	maxPlayers   int
	started      bool
}

type Lobby struct {
	rooms                map[string]*Room
	pubRoomsDescriptions map[string]RoomDescription
	addRoomChan          chan *Room
	removeRoomChan       chan *Room
	pingPlayers          chan struct{}
	pubGamesReq          chan chan []RoomDescription
	roomDescUpdate       chan RoomDescription
	joinRoomReq          chan RoomJoinRequest
	idGenerator          UniqueIdGenerator
}

type Idgen struct {
	ids    map[string]struct{}
	locker sync.Mutex
}
