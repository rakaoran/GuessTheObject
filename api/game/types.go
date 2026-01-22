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
	Create(duration time.Duration) chan time.Time
}
type UserGetter interface {
	GetUserById(ctx context.Context, id string) (domain.User, error)
}

type Player interface {
	Send(data []byte) error
	Ping() error
	SetRoom(r Room)
	CancelAndRelease()
	Username() string
}

type Room interface {
	PingPlayers()
	Send(ctx context.Context, e ClientPacketEnvelope)
	RemoveMe(ctx context.Context, p Player)
	RequestJoin(jreq roomJoinRequest)
	Tick(now time.Time) // time injected for testing
	GameLoop()
	CloseAndRelease()
	Description() roomDescription
	SetParentLobby(l Lobby)
	SetId(string)
}

type Lobby interface {
	RequestAddAndRunRoom(ctx context.Context, r Room)
	ForwardPlayerJoinRequestToRoom(ctx context.Context, jreq roomJoinRequest)
}

type roomJoinRequest struct {
	ctx     context.Context
	roomId  string
	player  Player
	errChan chan error
}

type player struct {
	id          string
	username    string
	room        Room
	rateLimiter rate.Limiter
	inbox       chan []byte
	pingChan    chan struct{}
	ctx         context.Context
	cancelCtx   context.CancelFunc
}

type ClientPacketEnvelope struct {
	clientPacket *protobuf.ClientPacket
	from         string
}

type room struct {
	private               bool
	id                    string
	host                  string
	players               []playerGameState
	drawerIndex           int
	currentDrawer         string
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
	playerRemovalRequests chan Player
	updateDescriptionChan chan roomDescription
	removeMe              chan Player
	joinReqs              chan roomJoinRequest
	randomWordsGenerator  RandomWordsGenerator
	parentLobby           Lobby
}

type playerGameState struct {
	player      Player
	username    string
	score       int
	hasGuessesd bool
}

type roomDescription struct {
	id           string
	private      bool
	playersCount int
	maxPlayers   int
	started      bool
}

type lobby struct {
	rooms                map[string]Room
	pubRoomsDescriptions map[string]roomDescription
	addAndRunRoomChan    chan Room
	removeRoomChan       chan string
	pubGamesReq          chan chan []roomDescription
	roomDescUpdate       chan roomDescription
	roomJoinReqs         chan roomJoinRequest
	idGenerator          UniqueIdGenerator
	tickerCreator        PeriodicTickerChannelCreator
}

type idgen struct {
	ids    map[string]struct{}
	locker sync.Mutex
}
