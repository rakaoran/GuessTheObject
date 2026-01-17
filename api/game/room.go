package game

import (
	"api/domain/protobuf"
	"time"

	"google.golang.org/protobuf/proto"
)

const (
	PHASE_PENDING RoomPhase = iota
	PHASE_CHOOSING_WORD
	PHASE_DRAWING
	PHASE_TURN_SUMMARY
	PHASE_GAMEEND
)

func (r *Room) addPlayer(p *Player) error {
	if len(r.players) >= r.maxPlayers {
		return ErrRoomFull
	}
	r.players = append(r.players, p)
	return nil
}

func (r *Room) removePlayer(toRemove *Player) {
	// TODO: if drawer left and stuff
	for i, p := range r.players {
		if p == toRemove {
			r.players = append(r.players[0:i], r.players[i+1:]...)
			return
		}
	}
}

func (r *Room) RoomActor() {
	for {
		if r.phase == PHASE_GAMEEND {
			return
		}
		select {
		case envelope := <-r.inbox:
			r.handleEnvelope(envelope)

		case now := <-r.ticks:
			r.handleTick(now)

		case p := <-r.playerRemovalRequests:
			r.removePlayer(p)
			r.broadcastPlayerLeft(p.username)

		case joinReq := <-r.joinRequests:
			p := joinReq.player
			err := r.addPlayer(p)
			if err != nil {
				joinReq.errChan <- err
			} else {
				r.broadcastPlayerJoined(p.username)
			}
		}

	}
}

func (r *Room) handleEnvelope(env ClientPacketEnvelope) {
	switch payload := env.clientPacket.Payload.(type) {
	case *protobuf.ClientPacket_DrawingData:
		r.handleDrawingData(payload.DrawingData, env.from)
	case *protobuf.ClientPacket_StartGame_:
		r.handleStartGame(env.from)
	case *protobuf.ClientPacket_WordChoice_:
		r.handleWordChoice(payload.WordChoice, env.from)
	case *protobuf.ClientPacket_PlayerMessage_:
		r.handlePlayerMessage(payload.PlayerMessage, env.from)
	}
}

func (r *Room) handleDrawingData(drawingData *protobuf.DrawingData, from *Player) {
	for i, p := range r.players {
		if i == r.drawerIndex && p == from {
			pkt := &protobuf.ServerPacket{
				Payload: &protobuf.ServerPacket_DrawingData{
					DrawingData: drawingData,
				},
			}

			r.broadcastToAll(pkt)
		} else {
			// TODO
		}
	}
}

func (r *Room) handleStartGame(from *Player) {
	if r.host != from {
		// TODO
		return
	}

	pkt := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_GameStarted_{
			GameStarted: &protobuf.ServerPacket_GameStarted{},
		},
	}
	r.broadcastToAll(pkt)
}

func (r *Room) handleWordChoice(wordChoice *protobuf.ClientPacket_WordChoice, from *Player) {
	if r.phase != PHASE_CHOOSING_WORD || from != r.currentDrawer {
		return
	}

	var n int64 = int64(len(r.wordChoices))
	choiceIndex := wordChoice.Choice

	if choiceIndex < 0 || choiceIndex >= n {
		// TODO
		return
	}

	r.currentWord = r.wordChoices[choiceIndex]

	playerStartedDrawing := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_PlayerIsDrawing_{
			PlayerIsDrawing: &protobuf.ServerPacket_PlayerIsDrawing{
				Username: from.username,
			},
		},
	}

	yourTurn := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_YourTurnToDraw_{
			YourTurnToDraw: &protobuf.ServerPacket_YourTurnToDraw{
				Word: r.currentWord,
			},
		},
	}

	r.broadcastToAllExcept(playerStartedDrawing, from)
	r.broadcastTo(yourTurn, from)
}

func (r *Room) handlePlayerMessage(clientMessage *protobuf.ClientPacket_PlayerMessage, from *Player) {
	serverPacket := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_PlayerMessage_{
			PlayerMessage: &protobuf.ServerPacket_PlayerMessage{
				From:    from.username,
				Message: clientMessage.Message,
			},
		},
		ServerTimestamp: time.Now().UnixMilli(),
	}

	bytesPacket, err := proto.Marshal(serverPacket)

	if err != nil {
		// TODO
		return
	}

	if from.hasGuessed {
		for i, p := range r.players {
			if p.hasGuessed || i == r.drawerIndex {
				p.inbox <- bytesPacket
			}
		}
	} else {
		for _, p := range r.players {
			p.inbox <- bytesPacket
		}
	}
}

func (r *Room) handleTick(now time.Time) {
	if now.Before(r.nextTick) {
		return
	}

	switch r.phase {
	case PHASE_PENDING:

	case PHASE_CHOOSING_WORD:

	case PHASE_DRAWING:

	case PHASE_TURN_SUMMARY:

	case PHASE_GAMEEND:
	}
}

func (r *Room) transitionToChoosingWord() {
	r.phase = PHASE_CHOOSING_WORD
	r.currentWord = ""
	if r.currentDrawer == nil || r.drawerIndex == 0 {
		r.drawerIndex = len(r.players) - 1
	} else {
		r.drawerIndex--
	}
	r.currentDrawer = r.players[r.drawerIndex]

	words := r.randomWordsGenerator.Generate(r.wordsCount)
	r.wordChoices = words

	plzChoose := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_PleaseChooseAWord_{
			PleaseChooseAWord: &protobuf.ServerPacket_PleaseChooseAWord{
				Words: words,
			},
		},
	}

	playerIsChoosing := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_PlayerIsChoosingWord_{
			PlayerIsChoosingWord: &protobuf.ServerPacket_PlayerIsChoosingWord{
				Username: r.currentDrawer.username,
			},
		},
	}

	r.broadcastTo(plzChoose, r.currentDrawer)
	r.broadcastToAllExcept(playerIsChoosing, r.currentDrawer)
	r.nextTick = time.Now().Add(r.choosingWordDuration)
}

func (r *Room) transitionToDrawing() {

}

func (r *Room) transitionToTurnSummary() {

}

func (r *Room) transitionToGameEnd() {

}

func (r *Room) broadcastPlayerLeft(username string) {
	playerLeft := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_PlayerLeft_{
			PlayerLeft: &protobuf.ServerPacket_PlayerLeft{
				Username: username,
			},
		},
	}
	r.broadcastToAll(playerLeft)
}

func (r *Room) broadcastPlayerJoined(username string) {
	playerJoined := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_PlayerJoined_{
			PlayerJoined: &protobuf.ServerPacket_PlayerJoined{
				Username: username,
			},
		},
	}
	r.broadcastToAll(playerJoined)
}
func (r *Room) broadcastToAll(serverPacket *protobuf.ServerPacket) {
	bytesPacket, err := proto.Marshal(serverPacket)

	if err != nil {
		// TODO
		return
	}

	for _, p := range r.players {
		p.inbox <- bytesPacket
	}
}

func (r *Room) broadcastTo(serverPacket *protobuf.ServerPacket, player *Player) {
	bytesPacket, err := proto.Marshal(serverPacket)

	if err != nil {
		// TODO
		return
	}

	for _, p := range r.players {
		if p == player {
			p.inbox <- bytesPacket
			return
		}
	}
}

func (r *Room) broadcastToAllExcept(serverPacket *protobuf.ServerPacket, player *Player) {
	bytesPacket, err := proto.Marshal(serverPacket)

	if err != nil {
		// TODO
		return
	}

	for _, p := range r.players {
		if p != player {
			p.inbox <- bytesPacket
			return
		}
	}
}
