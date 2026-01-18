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
		case pubGamesReq := <-l.pubGamesReq:
			x := make([]RoomDescription, 0, len(l.rooms))
			for _, room := range l.rooms {
				if room.private {
					continue
				}
				// here it's safe to read without locks, even tho it's a race condition,
				// a difference in the number by 1 or 2 is usually harmless
				// as the user knows that it can have already changed and needs to refresh
				x = append(x, RoomDescription{id: room.id, playersCount: len(l.rooms), maxPlayers: room.maxPlayers, started: room.phase != PHASE_PENDING})
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
