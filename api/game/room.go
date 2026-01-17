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

	x := &protobuf.ServerPacket_InitialRoomSnapshot{
		PlayersStates: make([]*protobuf.ServerPacket_InitialRoomSnapshot_PlayerState, len(r.players)),
	}
	initialRoomSnapshot := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_InitialRoomSnapshot_{
			InitialRoomSnapshot: x,
		},
	}
	for _, p := range r.players {
		x.PlayersStates = append(x.PlayersStates, &protobuf.ServerPacket_InitialRoomSnapshot_PlayerState{
			Username:  p.username,
			Score:     int64(p.score),
			IsGuesser: p.hasGuessed,
		})
	}
	x.CurrentDrawer = r.currentDrawer.username
	x.CurrentRound = int32(r.round)
	x.DrawingHistory = r.drawingHistory

	r.broadcastTo(initialRoomSnapshot, p)
	return nil
}

func (r *Room) removePlayer(toRemove *Player) {
	for i, p := range r.players {
		if p == toRemove {
			r.players = append(r.players[0:i], r.players[i+1:]...)

			if i < r.drawerIndex {
				r.drawerIndex--
			} else if i == r.drawerIndex {
				r.transitionToChoosingWord()
			}
			if len(r.players) <= 1 && r.phase != PHASE_PENDING {
				r.transitionToGameEnd()
			}
			return
		}
	}
}

func (r *Room) RoomActor() {
	for {
		if r.phase == PHASE_GAMEEND {
			break
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
	if r.currentDrawer == from {
		pkt := &protobuf.ServerPacket{
			Payload: &protobuf.ServerPacket_DrawingData{
				DrawingData: drawingData,
			},
		}

		r.broadcastToAll(pkt)
		r.drawingHistory = append(r.drawingHistory, drawingData.Data)
		return
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
		r.transitionToChoosingWord()
	case PHASE_CHOOSING_WORD:
		r.transitionToDrawing()
	case PHASE_DRAWING:
		r.transitionToTurnSummary()
	case PHASE_TURN_SUMMARY:
		r.transitionToChoosingWord()
	case PHASE_GAMEEND:
	}
}

func (r *Room) transitionToChoosingWord() {
	r.phase = PHASE_CHOOSING_WORD
	r.currentWord = ""
	if r.currentDrawer == nil {
		r.drawerIndex = len(r.players) - 1
	} else if r.drawerIndex == 0 {
		r.transitionToNextRound()
		return
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
	r.phase = PHASE_DRAWING
	if r.currentWord == "" {
		r.currentWord = r.wordChoices[0]
	}

	drawer := r.currentDrawer

	playerStartedDrawing := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_PlayerIsDrawing_{
			PlayerIsDrawing: &protobuf.ServerPacket_PlayerIsDrawing{
				Username: drawer.username,
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

	r.broadcastToAllExcept(playerStartedDrawing, drawer)
	r.broadcastTo(yourTurn, drawer)
	r.nextTick = time.Now().Add(r.drawingDuration)
}

func (r *Room) transitionToTurnSummary() {
	r.phase = PHASE_TURN_SUMMARY
	clear(r.drawingHistory)
	r.drawingHistory = r.drawingHistory[:0]

	x := &protobuf.ServerPacket_TurnSummary{
		WordReveal: r.currentWord,
		Deltas:     []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{},
	}
	turnSummary := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_TurnSummary_{
			TurnSummary: x,
		},
	}

	for _, p := range r.players {
		x.Deltas = append(x.Deltas, &protobuf.ServerPacket_TurnSummary_ScoreDeltas{
			ScoreDelta: int64(p.scoreIncrement),
			Username:   p.username,
		})
	}

	r.broadcastToAll(turnSummary)
	r.nextTick = time.Now().Add(5 * time.Second)
}

func (r *Room) transitionToNextRound() {
	r.round++
	if r.round > r.roundsCount {
		r.transitionToGameEnd()
		return
	}
	nextRound := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_RoundUpdate_{
			RoundUpdate: &protobuf.ServerPacket_RoundUpdate{
				RoundNumber: int64(r.round),
			},
		},
	}

	r.broadcastToAll(nextRound)
	r.transitionToChoosingWord()
}

func (r *Room) transitionToGameEnd() {
	leaderboard := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_Leaderboard{},
	}

	r.broadcastToAll(leaderboard)
	r.clearResources()
	r.phase = PHASE_GAMEEND
}

func (r *Room) clearResources() {
	// ! I need to make the lobby remove this game first before cleaning so there can't be a panic of writing to a closed channel
	r.players = nil
	r.wordChoices = nil
	r.drawingHistory = nil
	close(r.inbox)
	close(r.ticks)
	close(r.playerRemovalRequests)
	close(r.joinRequests)
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
