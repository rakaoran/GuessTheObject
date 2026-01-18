package game

func (l *Lobby) LobbyActor(tickerCreator PeriodicTickerChannelCreator) {
	c := tickerCreator.Create()
	for {
		select {
		case now := <-c:
			for _, r := range l.rooms {
				select {
				case r.ticks <- now:
				default:
				}
			}
		case s := <-l.pingPlayers:
			for _, r := range l.rooms {
				select {
				case r.pingPlayers <- s:
				default:
				}
			}
		case room := <-l.addRoomChan:
			l.addRoom(room)
		case room := <-l.removeRoomChan:
			l.removeRoom(room)
		case desc := <-l.roomDescUpdate:
			l.pubRoomsDescriptions[desc.id] = desc
		case pubGamesReq := <-l.pubGamesReq:
			x := make([]RoomDescription, 0, len(l.rooms))
			for _, description := range l.pubRoomsDescriptions {
				x = append(x, description)
			}
			pubGamesReq <- x
		}
	}
}

func (l *Lobby) addRoom(r *Room) {
	id := l.idGenerator.Generate()
	l.rooms[id] = r
}

func (l *Lobby) removeRoom(toRemove *Room) {
	delete(l.rooms, toRemove.id)
	close(toRemove.ticks)
	close(toRemove.pingPlayers)
}
