package game

import (
	"context"
	"time"
)

func NewLobby(idgen UniqueIdGenerator, tickerCreator PeriodicTickerChannelCreator) *lobby {
	return &lobby{
		rooms:                map[string]Room{},
		pubRoomsDescriptions: map[string]roomDescription{},
		addAndRunRoomChan:    make(chan Room, 32),
		removeRoomChan:       make(chan string, 32),
		pubGamesReq:          make(chan chan []roomDescription, 256),
		roomDescUpdate:       make(chan roomDescription, 256),
		roomJoinReqs:         make(chan roomJoinRequest, 256),
		idGenerator:          idgen,
		tickerCreator:        tickerCreator,
	}
}

func (l *lobby) RequestUpdateDescription(desc roomDescription) {
	select {
	case l.roomDescUpdate <- desc:
	default:
	}
}

func (l *lobby) RequestAddAndRunRoom(ctx context.Context, r Room) {
	select {
	case l.addAndRunRoomChan <- r:
	case <-ctx.Done():
	}

}
func (l *lobby) ForwardPlayerJoinRequestToRoom(ctx context.Context, jreq roomJoinRequest) {
	select {
	case <-ctx.Done():
	case l.roomJoinReqs <- jreq:
	}
}

func (l *lobby) RemoveRoom(roomId string) {
	l.removeRoomChan <- roomId
}

func (l *lobby) GetPublicGames(ctx context.Context) []roomDescription {
	respChan := make(chan []roomDescription, 1)
	select {
	case l.pubGamesReq <- respChan:
		select {
		case resp := <-respChan:
			return resp
		case <-ctx.Done():
			return nil
		}
	case <-ctx.Done():
		return nil
	}
}

func (l *lobby) LobbyActor(started chan struct{}) {
	ticker := l.tickerCreator.Create(time.Second)
	pingTicker := l.tickerCreator.Create(time.Second * 30)

	close(started)

	for {
		select {
		case now := <-ticker:
			for _, r := range l.rooms {
				r.Tick(now)
			}
		case <-pingTicker:
			for _, r := range l.rooms {
				r.PingPlayers()
			}

		case room := <-l.addAndRunRoomChan:
			l.handleAddAndRunRoom(room)

		case room := <-l.removeRoomChan:
			l.handleRemoveRoom(room)

		case desc := <-l.roomDescUpdate:
			l.pubRoomsDescriptions[desc.id] = desc

		case pubGamesReq := <-l.pubGamesReq:
			l.handleGetPublicRoomsDescription(pubGamesReq)

		case joinReq := <-l.roomJoinReqs:
			l.handleJoinReq(joinReq)
		}
	}
}

func (l *lobby) handleAddAndRunRoom(r Room) {
	id := l.idGenerator.Generate()
	r.SetParentLobby(l)

	l.rooms[id] = r
	r.SetId(id)
	rDesc := r.Description()
	go r.GameLoop()
	if rDesc.private {
		return
	}
	l.pubRoomsDescriptions[id] = rDesc
}

func (l *lobby) handleRemoveRoom(toRemoveId string) {
	room, _ := l.rooms[toRemoveId]
	delete(l.rooms, toRemoveId)
	delete(l.pubRoomsDescriptions, toRemoveId)
	room.CloseAndRelease()
	l.idGenerator.Dispose(toRemoveId)
}

func (l *lobby) handleGetPublicRoomsDescription(req chan []roomDescription) {
	x := make([]roomDescription, 0, len(l.pubRoomsDescriptions))
	for _, description := range l.pubRoomsDescriptions {
		x = append(x, description)
	}

	req <- x

}

func (l *lobby) handleJoinReq(joinReq roomJoinRequest) {
	room, ok := l.rooms[joinReq.roomId]
	if !ok {
		joinReq.errChan <- ErrRoomNotFound
		close(joinReq.errChan)
		return
	}
	room.RequestJoin(joinReq)
}
