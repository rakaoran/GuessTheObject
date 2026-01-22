package game

import (
	"context"
	"time"
)

const (
	PHASE_PENDING RoomPhase = iota
	PHASE_CHOOSING_WORD
	PHASE_DRAWING
	PHASE_TURN_SUMMARY
	PHASE_GAMEEND
)

func NewRoom(
	host Player,
	private bool,
	maxPlayers int,
	roundsCount int,
	wordsCount int,
	choosingWordDuration time.Duration,
	drawingDuration time.Duration,
	randomWordsGenerator RandomWordsGenerator,
) *room {
	r := &room{
		private: private,
		host:    host.Username(),
		players: []playerGameState{
			{player: host, username: host.Username()},
		},
		drawerIndex:           0,
		maxPlayers:            maxPlayers,
		roundsCount:           roundsCount,
		wordsCount:            wordsCount,
		phase:                 PHASE_PENDING,
		guessersCount:         0,
		round:                 1,
		nextTick:              time.Now().Add(time.Minute * 15),
		choosingWordDuration:  choosingWordDuration,
		drawingDuration:       drawingDuration,
		wordChoices:           nil,
		drawingHistory:        make([][]byte, 0, 1024),
		inbox:                 make(chan ClientPacketEnvelope, 2048),
		ticks:                 make(chan time.Time, 1),
		pingPlayers:           make(chan struct{}, 1),
		playerRemovalRequests: make(chan Player, 20),
		randomWordsGenerator:  randomWordsGenerator,
	}

	return r
}

func (r *room) PingPlayers() {
	select {
	case r.pingPlayers <- struct{}{}:
	default:
	}
}

func (r *room) Send(ctx context.Context, e ClientPacketEnvelope) {
	select {
	case <-ctx.Done():
	case r.inbox <- e:
	}
}

func (r *room) RemoveMe(ctx context.Context, p Player) {
	select {
	case <-ctx.Done():
	case r.removeMe <- p:
	}
}

func (r *room) RequestJoin(jreq roomJoinRequest) {
	select {
	case r.joinReqs <- jreq:
	default:
		jreq.errChan <- ErrRoomFull
	}
}

func (r *room) Tick(now time.Time) {
	select {
	case r.ticks <- now:
	default:
	}
}

func (r *room) GameLoop() {
	// TODO: implement
}

func (r *room) CloseAndRelease() {
	// TODO: implement
}

func (r *room) Description() roomDescription {
	return roomDescription{
		id:           r.id,
		private:      r.private,
		playersCount: len(r.players),
		maxPlayers:   r.maxPlayers,
		started:      r.phase != PHASE_PENDING,
	}
}

func (r *room) SetParentLobby(l Lobby) {
	r.parentLobby = l
}

func (r *room) SetId(id string) {
	r.id = id
}

// func (r *room) addPlayer(p *player) error {
// 	if len(r.players) >= r.maxPlayers {
// 		return ErrRoomFull
// 	}
// 	r.players = append(r.players, p)

// 	x := &protobuf.ServerPacket_InitialRoomSnapshot{
// 		PlayersStates: make([]*protobuf.ServerPacket_InitialRoomSnapshot_PlayerState, len(r.players)),
// 	}
// 	initialRoomSnapshot := &protobuf.ServerPacket{
// 		Payload: &protobuf.ServerPacket_InitialRoomSnapshot_{
// 			InitialRoomSnapshot: x,
// 		},
// 	}
// 	for _, p := range r.players {
// 		x.PlayersStates = append(x.PlayersStates, &protobuf.ServerPacket_InitialRoomSnapshot_PlayerState{
// 			Username:  p.username,
// 			Score:     int64(p.score),
// 			IsGuesser: p.hasGuessed,
// 		})
// 	}
// 	x.CurrentDrawer = r.currentDrawer.username
// 	x.CurrentRound = int32(r.round)
// 	x.DrawingHistory = r.drawingHistory

// 	r.broadcastTo(initialRoomSnapshot, p)
// 	p.roomChan = r.inbox
// 	p.removeMe = r.playerRemovalRequests
// 	playerJoined := &protobuf.ServerPacket{
// 		Payload: &protobuf.ServerPacket_PlayerJoined_{
// 			PlayerJoined: &protobuf.ServerPacket_PlayerJoined{
// 				Username: p.username,
// 			},
// 		},
// 	}
// 	r.broadcastToAll(playerJoined)
// 	r.updateDescription()
// 	return nil
// }

// func (r *room) removePlayer(toRemove *player) {
// 	for i, p := range r.players {
// 		if p == toRemove {
// 			r.players = append(r.players[0:i], r.players[i+1:]...)

// 			if i < r.drawerIndex {
// 				r.drawerIndex--
// 			} else if i == r.drawerIndex {
// 				r.transitionToChoosingWord()
// 			}
// 			if len(r.players) <= 1 && r.phase != PHASE_PENDING {
// 				r.transitionToGameEnd()
// 			}
// 			p.cancelCtx()
// 			close(p.inbox)
// 			close(p.pingChan)
// 			p.roomChan = nil
// 			p.removeMe = nil
// 			playerLeft := &protobuf.ServerPacket{
// 				Payload: &protobuf.ServerPacket_PlayerLeft_{
// 					PlayerLeft: &protobuf.ServerPacket_PlayerLeft{
// 						Username: toRemove.username,
// 					},
// 				},
// 			}
// 			r.broadcastToAll(playerLeft)
// 			r.updateDescription()
// 			return
// 		}
// 	}
// }

// func (r *room) RoomActor() {
// 	for {
// 		if r.phase == PHASE_GAMEEND {
// 			return
// 		}
// 		select {
// 		case s := <-r.pingPlayers:
// 			for _, p := range r.players {
// 				p.pingChan <- s
// 			}

// 		case envelope := <-r.inbox:
// 			r.handleEnvelope(envelope)

// 		case now := <-r.ticks:
// 			r.handleTick(now)

// 		case p := <-r.playerRemovalRequests:
// 			r.removePlayer(p)

// 		}

// 	}
// }

// func (r *room) handleEnvelope(env ClientPacketEnvelope) {
// 	switch payload := env.clientPacket.Payload.(type) {
// 	case *protobuf.ClientPacket_DrawingData:
// 		r.handleDrawingData(payload.DrawingData, env.from)
// 	case *protobuf.ClientPacket_StartGame_:
// 		r.handleStartGame(env.from)
// 	case *protobuf.ClientPacket_WordChoice_:
// 		r.handleWordChoice(payload.WordChoice, env.from)
// 	case *protobuf.ClientPacket_PlayerMessage_:
// 		r.handlePlayerMessage(payload.PlayerMessage, env.from)
// 	}
// }

// func (r *room) handleDrawingData(drawingData *protobuf.DrawingData, from *player) {
// 	if r.currentDrawer == from {
// 		pkt := &protobuf.ServerPacket{
// 			Payload: &protobuf.ServerPacket_DrawingData{
// 				DrawingData: drawingData,
// 			},
// 		}

// 		r.broadcastToAll(pkt)
// 		r.drawingHistory = append(r.drawingHistory, drawingData.Data)
// 		return
// 	}

// }

// func (r *room) handleStartGame(from *player) {
// 	r.updateDescription()
// 	if r.host != from {
// 		// TODO
// 		return
// 	}

// 	pkt := &protobuf.ServerPacket{
// 		Payload: &protobuf.ServerPacket_GameStarted_{
// 			GameStarted: &protobuf.ServerPacket_GameStarted{},
// 		},
// 	}
// 	r.broadcastToAll(pkt)
// }

// func (r *room) handleWordChoice(wordChoice *protobuf.ClientPacket_WordChoice, from *player) {
// 	if r.phase != PHASE_CHOOSING_WORD || from != r.currentDrawer {
// 		return
// 	}

// 	var n int64 = int64(len(r.wordChoices))
// 	choiceIndex := wordChoice.Choice

// 	if choiceIndex < 0 || choiceIndex >= n {
// 		// TODO
// 		return
// 	}
// 	r.currentWord = r.wordChoices[choiceIndex]
// }

// func (r *room) handlePlayerMessage(clientMessage *protobuf.ClientPacket_PlayerMessage, from *player) {
// 	if clientMessage.Message == r.currentWord && !from.hasGuessed && r.phase == PHASE_DRAWING {
// 		serverPacket := &protobuf.ServerPacket{
// 			Payload: &protobuf.ServerPacket_PlayerGuessedTheWord_{
// 				PlayerGuessedTheWord: &protobuf.ServerPacket_PlayerGuessedTheWord{
// 					Username: from.username,
// 				},
// 			},
// 		}
// 		from.scoreIncrement = (len(r.players) - r.guessersCount) * 100
// 		from.hasGuessed = true
// 		r.guessersCount++
// 		r.broadcastToAll(serverPacket)
// 		if len(r.players) == r.guessersCount {
// 			r.transitionToTurnSummary()
// 		}
// 		return
// 	}
// 	serverPacket := &protobuf.ServerPacket{
// 		Payload: &protobuf.ServerPacket_PlayerMessage_{
// 			PlayerMessage: &protobuf.ServerPacket_PlayerMessage{
// 				From:    from.username,
// 				Message: clientMessage.Message,
// 			},
// 		},
// 		ServerTimestamp: time.Now().UnixMilli(),
// 	}

// 	bytesPacket, err := proto.Marshal(serverPacket)

// 	if err != nil {
// 		// TODO
// 		return
// 	}

// 	if from.hasGuessed {
// 		for i, p := range r.players {
// 			if p.hasGuessed || i == r.drawerIndex {
// 				p.inbox <- bytesPacket
// 			}
// 		}
// 	} else {
// 		for _, p := range r.players {
// 			p.inbox <- bytesPacket
// 		}
// 	}
// }

// func (r *room) handleTick(now time.Time) {
// 	if now.Before(r.nextTick) {
// 		return
// 	}

// 	switch r.phase {
// 	case PHASE_PENDING:
// 		r.transitionToChoosingWord()
// 	case PHASE_CHOOSING_WORD:
// 		r.transitionToDrawing()
// 	case PHASE_DRAWING:
// 		r.transitionToTurnSummary()
// 	case PHASE_TURN_SUMMARY:
// 		r.transitionToChoosingWord()
// 	case PHASE_GAMEEND:
// 	}
// }

// func (r *room) transitionToChoosingWord() {
// 	r.phase = PHASE_CHOOSING_WORD
// 	r.currentWord = ""
// 	r.guessersCount = 0
// 	for _, p := range r.players {
// 		p.hasGuessed = false
// 	}
// 	if r.currentDrawer == nil {
// 		r.drawerIndex = len(r.players) - 1
// 	} else if r.drawerIndex == 0 {
// 		r.transitionToNextRound()
// 		return
// 	} else {
// 		r.drawerIndex--
// 	}
// 	r.currentDrawer = r.players[r.drawerIndex]

// 	words := r.randomWordsGenerator.Generate(r.wordsCount)
// 	r.wordChoices = words

// 	plzChoose := &protobuf.ServerPacket{
// 		Payload: &protobuf.ServerPacket_PleaseChooseAWord_{
// 			PleaseChooseAWord: &protobuf.ServerPacket_PleaseChooseAWord{
// 				Words: words,
// 			},
// 		},
// 	}

// 	playerIsChoosing := &protobuf.ServerPacket{
// 		Payload: &protobuf.ServerPacket_PlayerIsChoosingWord_{
// 			PlayerIsChoosingWord: &protobuf.ServerPacket_PlayerIsChoosingWord{
// 				Username: r.currentDrawer.username,
// 			},
// 		},
// 	}

// 	r.broadcastTo(plzChoose, r.currentDrawer)
// 	r.broadcastToAllExcept(playerIsChoosing, r.currentDrawer)
// 	r.nextTick = time.Now().Add(r.choosingWordDuration)
// }

// func (r *room) transitionToDrawing() {
// 	r.phase = PHASE_DRAWING
// 	if r.currentWord == "" {
// 		r.currentWord = r.wordChoices[0]
// 	}

// 	drawer := r.currentDrawer

// 	playerStartedDrawing := &protobuf.ServerPacket{
// 		Payload: &protobuf.ServerPacket_PlayerIsDrawing_{
// 			PlayerIsDrawing: &protobuf.ServerPacket_PlayerIsDrawing{
// 				Username: drawer.username,
// 			},
// 		},
// 	}

// 	yourTurn := &protobuf.ServerPacket{
// 		Payload: &protobuf.ServerPacket_YourTurnToDraw_{
// 			YourTurnToDraw: &protobuf.ServerPacket_YourTurnToDraw{
// 				Word: r.currentWord,
// 			},
// 		},
// 	}

// 	r.broadcastToAllExcept(playerStartedDrawing, drawer)
// 	r.broadcastTo(yourTurn, drawer)
// 	r.nextTick = time.Now().Add(r.drawingDuration)
// }

// func (r *room) transitionToTurnSummary() {
// 	r.phase = PHASE_TURN_SUMMARY
// 	clear(r.drawingHistory)
// 	r.drawingHistory = r.drawingHistory[:0]

// 	x := &protobuf.ServerPacket_TurnSummary{
// 		WordReveal: r.currentWord,
// 		Deltas:     []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{},
// 	}
// 	turnSummary := &protobuf.ServerPacket{
// 		Payload: &protobuf.ServerPacket_TurnSummary_{
// 			TurnSummary: x,
// 		},
// 	}

// 	for _, p := range r.players {
// 		x.Deltas = append(x.Deltas, &protobuf.ServerPacket_TurnSummary_ScoreDeltas{
// 			ScoreDelta: int64(p.scoreIncrement),
// 			Username:   p.username,
// 		})
// 	}

// 	r.broadcastToAll(turnSummary)
// 	r.nextTick = time.Now().Add(5 * time.Second)
// }

// func (r *room) transitionToNextRound() {
// 	r.round++
// 	if r.round > r.roundsCount {
// 		r.transitionToGameEnd()
// 		return
// 	}
// 	nextRound := &protobuf.ServerPacket{
// 		Payload: &protobuf.ServerPacket_RoundUpdate_{
// 			RoundUpdate: &protobuf.ServerPacket_RoundUpdate{
// 				RoundNumber: int64(r.round),
// 			},
// 		},
// 	}

// 	r.broadcastToAll(nextRound)
// 	r.transitionToChoosingWord()
// }

// func (r *room) transitionToGameEnd() {
// 	r.phase = PHASE_GAMEEND
// 	leaderboard := &protobuf.ServerPacket{
// 		Payload: &protobuf.ServerPacket_Leaderboard{},
// 	}

// 	r.broadcastToAll(leaderboard)
// 	r.clearResources()
// }

// func (r *room) clearResources() {
// 	for _, p := range r.players {
// 		p.cancelCtx()
// 		close(p.inbox)
// 		close(p.pingChan)
// 		p.removeMe = nil
// 		p.roomChan = nil
// 	}
// 	// safe to close
// 	close(r.inbox)
// 	r.removeMe <- r
// 	r.players = nil
// 	r.wordChoices = nil
// 	r.drawingHistory = nil
// }

// func (r *room) broadcastToAll(serverPacket *protobuf.ServerPacket) {
// 	bytesPacket, err := proto.Marshal(serverPacket)

// 	if err != nil {
// 		// TODO
// 		return
// 	}

// 	for _, p := range r.players {
// 		select {
// 		case p.inbox <- bytesPacket:
// 		default:
// 			r.removePlayer(p)
// 		}
// 	}
// }

// func (r *room) broadcastTo(serverPacket *protobuf.ServerPacket, player *player) {
// 	bytesPacket, err := proto.Marshal(serverPacket)

// 	if err != nil {
// 		// TODO
// 		return
// 	}

// 	for _, p := range r.players {
// 		if p == player {
// 			select {
// 			case p.inbox <- bytesPacket:
// 			default:
// 				r.removePlayer(p)
// 			}
// 			return
// 		}
// 	}
// }

// func (r *room) broadcastToAllExcept(serverPacket *protobuf.ServerPacket, player *player) {
// 	bytesPacket, err := proto.Marshal(serverPacket)

// 	if err != nil {
// 		// TODO
// 		return
// 	}

// 	for _, p := range r.players {
// 		if p != player {
// 			select {
// 			case p.inbox <- bytesPacket:
// 			default:
// 				r.removePlayer(p)
// 			}
// 		}
// 	}
// }

// func (r *room) updateDescription() {
// 	if r.private {
// 		return
// 	}
// 	desc := roomDescription{
// 		id:           r.id,
// 		playersCount: len(r.players),
// 		maxPlayers:   r.maxPlayers,
// 		started:      r.phase != PHASE_PENDING,
// 	}
// 	select {
// 	case r.updateDescriptionChan <- desc:
// 	default:
// 	}
// }
