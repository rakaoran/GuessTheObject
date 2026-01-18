package game

import "time"

func (l *Lobby) LobbyActor(tickerCreator PeriodicTickerChannelCreator) {
	ticker := tickerCreator.Create(time.Second)
	pingTicker := tickerCreator.Create(time.Second * 30)
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
				case r.pingPlayers <- struct{}{}:
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
	l.rooms[id] = r
}

func (l *Lobby) handleRemoveRoom(toRemove *Room) {
	delete(l.rooms, toRemove.id)
	close(toRemove.ticks)
	close(toRemove.pingPlayers)
	close(toRemove.joinRequests)
}

func (l *Lobby) handleGetPublicRoomsDescription(req chan []RoomDescription) {
	x := make([]RoomDescription, 0, len(l.rooms))
	for _, description := range l.pubRoomsDescriptions {
		x = append(x, description)
	}
	req <- x
}

func (l *Lobby) handleJoinReq(joinReq RoomJoinRequest) {
	room, ok := l.rooms[joinReq.roomId]
	if !ok {
		joinReq.errChan <- ErrRoomNotFound
		close(joinReq.errChan)
	}
	select {
	case room.joinRequests <- joinReq:
	default:
		joinReq.errChan <- ErrRoomFull
		close(joinReq.errChan)

	}
}
