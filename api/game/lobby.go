package game

import (
	"context"
	"time"
)

func NewLobby(idgen UniqueIdGenerator, tickerCreator PeriodicTickerChannelCreator) *Lobby {
	ctx, cancel := context.WithCancel(context.Background())
	return &Lobby{
		rooms:                map[string]*Room{},
		pubRoomsDescriptions: map[string]RoomDescription{},
		addRoomChan:          make(chan *Room, 256),
		removeRoomChan:       make(chan *Room, 256),
		pubGamesReq:          make(chan chan []RoomDescription, 256),
		roomDescUpdate:       make(chan RoomDescription, 256),
		joinRoomReq:          make(chan RoomJoinRequest, 1024),
		ctx:                  ctx,
		cancelCtx:            cancel,
		idGenerator:          idgen,
		tickerCreator:        tickerCreator,
	}
}

func (l *Lobby) LobbyActor(started chan struct{}) {
	ticker := l.tickerCreator.Create(time.Second)
	pingTicker := l.tickerCreator.Create(time.Second * 30)
	s := struct{}{}

	close(started)

	for {
		select {
		case now := <-ticker:
			for _, r := range l.rooms {
				select {
				case r.ticks <- now:
				default:
				}
			}
		case <-pingTicker:
			for _, r := range l.rooms {
				select {
				case r.pingPlayers <- s:
				default:
				}
			}

		case room := <-l.addRoomChan:
			l.handlAddRoom(room)

		case room := <-l.removeRoomChan:
			l.handleRemoveRoom(room)

		case desc := <-l.roomDescUpdate:
			l.pubRoomsDescriptions[desc.id] = desc

		case pubGamesReq := <-l.pubGamesReq:
			l.handleGetPublicRoomsDescription(pubGamesReq)

		case joinReq := <-l.joinRoomReq:
			l.handleJoinReq(joinReq)
		}
	}
}

func (l *Lobby) handlAddRoom(r *Room) {
	id := l.idGenerator.Generate()
	r.removeMe = l.removeRoomChan
	r.updateDescriptionChan = l.roomDescUpdate
	r.joinRequests = l.joinRoomReq
	r.id = id

	l.rooms[id] = r

	if r.private {
		return
	}
	l.pubRoomsDescriptions[id] = RoomDescription{
		id:           id,
		playersCount: len(r.players),
		maxPlayers:   r.maxPlayers,
		started:      r.phase != PHASE_PENDING,
	}
}

func (l *Lobby) handleRemoveRoom(toRemove *Room) {
	println("NIGGA ", toRemove.id)
	delete(l.rooms, toRemove.id)
	delete(l.pubRoomsDescriptions, toRemove.id)
	close(toRemove.ticks)
	close(toRemove.pingPlayers)
	close(toRemove.joinRequests)
	l.idGenerator.Dispose(toRemove.id)
}

func (l *Lobby) handleGetPublicRoomsDescription(req chan []RoomDescription) {
	x := make([]RoomDescription, 0, len(l.pubRoomsDescriptions))
	for _, description := range l.pubRoomsDescriptions {
		x = append(x, description)
	}
	select {
	case req <- x:
	default:
	}
}

func (l *Lobby) handleJoinReq(joinReq RoomJoinRequest) {
	room, ok := l.rooms[joinReq.roomId]
	if !ok {
		select {
		case joinReq.errChan <- ErrRoomNotFound:
			close(joinReq.errChan)
		default:
		}
	}
	select {
	case room.joinRequests <- joinReq:
	default:
		select {
		case joinReq.errChan <- ErrRoomFull:
			close(joinReq.errChan)
		default:
		}

	}
}
