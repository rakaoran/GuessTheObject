package game

func (r *Room) RoomActor() {
	for {
		envelope, ok := <-r.inbox
		if !ok {
			break
		}

		if envelope.rawBinary != nil {

		} else {

		}
	}

	if r := recover(); r != nil {
		// TODO
	}
}
