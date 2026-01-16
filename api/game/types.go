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

type UserGetter interface {
	GetUserById(ctx context.Context, id string) (domain.User, error)
}

type Player struct {
	id          string
	username    string
	rateLimiter rate.Limiter
	inbox       chan []byte
	pingChan    chan struct{}
	roomChan    chan<- ClientPacketEnvelope
}

type ClientPacketEnvelope struct {
	clientPacket *protobuf.ClientPacket
	rawBinary    []byte
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
	scores                map[*Player]int
	drawerIndex           int
	maxPlayers            int
	roundsCount           int
	phase                 RoomPhase
	round                 int
	nextTick              time.Time
	choosingWordDuration  time.Duration
	drawingDuration       time.Duration
	currentWord           string
	bannedPlayerIds       map[string]struct{}
	scoreIncrements       map[*Player]int
	wordChoices           []string
	drawingHistory        [][]byte
	guessers              map[string]bool
	inbox                 chan ClientPacketEnvelope
	ticks                 chan time.Time
	playerRemovalRequests chan *Player
	joinRequests          chan RoomJoinRequest
}

type Idgen struct {
	ids    map[string]struct{}
	locker sync.Mutex
}
