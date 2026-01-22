package game

import (
	"api/domain/protobuf"
	"context"
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
		playerStates: []*playerGameState{
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
		joinReqs:              make(chan roomJoinRequest, maxPlayers),
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
	case r.playerRemovalRequests <- p:
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

func (r *room) CloseAndRelease() {
	close(r.joinReqs)
	close(r.pingPlayers)
	close(r.ticks)

}

func (r *room) Description() roomDescription {
	return roomDescription{
		id:           r.id,
		private:      r.private,
		playersCount: len(r.playerStates),
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

func (r *room) GameLoop() {
	for {
		if r.phase == PHASE_GAMEEND {
			return
		}
		select {
		case _, ok := <-r.pingPlayers:
			if !ok {
				return
			}
			r.handlePingPlayers()

		case envelope := <-r.inbox:
			r.handleEnvelope(envelope)

		case now, ok := <-r.ticks:
			if !ok {
				return
			}
			r.handleTick(now)

		case p, ok := <-r.playerRemovalRequests:
			if !ok {
				return
			}
			r.handleRemovePlayer(p)
		}

		r.executeAndClearTasks()
	}
}

func (r *room) executeAndClearTasks() {

	for i := range r.pingSendTasks {
		err := r.pingSendTasks[i].to.Ping()
		if err != nil {
			r.handleRemovePlayer(r.pingSendTasks[i].to)
		}
	}

	i := 0
	for i < len(r.dataSendTasks) {
		to := r.dataSendTasks[i].to
		data := r.dataSendTasks[i].data
		err := to.Send(data)
		if err != nil {
			r.handleRemovePlayer(to)
		}
		i++
	}

	clear(r.dataSendTasks)
	r.dataSendTasks = r.dataSendTasks[:0]
	clear(r.pingSendTasks)
	r.pingSendTasks = r.pingSendTasks[:0]
}

func (r *room) handlePingPlayers() {
	for _, ps := range r.playerStates {
		r.pingSendTasks = append(r.pingSendTasks, pingSendTask{to: ps.player})
	}
}

func (r *room) addPlayer(p Player) error {
	if len(r.playerStates) >= r.maxPlayers {
		return ErrRoomFull
	}
	r.playerStates = append(r.playerStates, &playerGameState{username: p.Username(), player: p})
	p.SetRoom(r)
	x := &protobuf.ServerPacket_InitialRoomSnapshot{
		PlayersStates: make([]*protobuf.ServerPacket_InitialRoomSnapshot_PlayerState, len(r.playerStates)),
	}
	initialRoomSnapshot := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_InitialRoomSnapshot_{
			InitialRoomSnapshot: x,
		},
	}
	for _, ps := range r.playerStates {
		x.PlayersStates = append(x.PlayersStates, &protobuf.ServerPacket_InitialRoomSnapshot_PlayerState{
			Username:  ps.username,
			Score:     int64(ps.score),
			IsGuesser: ps.hasGuessed,
		})
	}
	x.CurrentDrawer = r.currentDrawer
	x.CurrentRound = int32(r.round)
	x.DrawingHistory = r.drawingHistory

	r.broadcastTo(initialRoomSnapshot, p)
	playerJoined := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_PlayerJoined_{
			PlayerJoined: &protobuf.ServerPacket_PlayerJoined{
				Username: p.Username(),
			},
		},
	}
	r.broadcastToAll(playerJoined)
	r.updateDescription()
	return nil
}

func (r *room) handleRemovePlayer(toRemove Player) {
	for i, ps := range r.playerStates {
		if ps.player == toRemove {
			r.playerStates = append(r.playerStates[0:i], r.playerStates[i+1:]...)

			if i < r.drawerIndex {
				r.drawerIndex--
			} else if i == r.drawerIndex {
				r.transitionToChoosingWord()
			}
			if len(r.playerStates) <= 1 && r.phase != PHASE_PENDING {
				r.transitionToGameEnd()
			}
			toRemove.CancelAndRelease()

			playerLeft := &protobuf.ServerPacket{
				Payload: &protobuf.ServerPacket_PlayerLeft_{
					PlayerLeft: &protobuf.ServerPacket_PlayerLeft{
						Username: ps.username,
					},
				},
			}
			r.broadcastToAll(playerLeft)
			r.updateDescription()
			return
		}
	}
}

func (r *room) handleEnvelope(env ClientPacketEnvelope) {
	switch payload := env.clientPacket.Payload.(type) {
	case *protobuf.ClientPacket_DrawingData:
		r.handleDrawingDataEnvelope(payload.DrawingData, env.from)
	case *protobuf.ClientPacket_StartGame_:
		r.handleStartGameEnvelope(env.from)
	case *protobuf.ClientPacket_WordChoice_:
		r.handleWordChoiceEnvelope(payload.WordChoice, env.from)
	case *protobuf.ClientPacket_PlayerMessage_:
		r.handlePlayerMessageEnvelope(payload.PlayerMessage, env.from)
	}
}

func (r *room) handleDrawingDataEnvelope(drawingData *protobuf.DrawingData, from string) {
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

func (r *room) handleStartGameEnvelope(from string) {
	if r.phase != PHASE_PENDING {
		return
	}
	r.updateDescription()
	if r.host != from {
		return
	}

	pkt := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_GameStarted_{
			GameStarted: &protobuf.ServerPacket_GameStarted{},
		},
	}
	r.broadcastToAll(pkt)
	r.updateDescription()
}

func (r *room) handleWordChoiceEnvelope(wordChoice *protobuf.ClientPacket_WordChoice, from string) {
	if r.phase != PHASE_CHOOSING_WORD || from != r.currentDrawer {
		return
	}

	var n int64 = int64(len(r.wordChoices))
	choiceIndex := wordChoice.Choice

	if choiceIndex < 0 || choiceIndex >= n {
		return
	}
	r.currentWord = r.wordChoices[choiceIndex]
}

func (r *room) handlePlayerMessageEnvelope(clientMessage *protobuf.ClientPacket_PlayerMessage, from string) {
	senderIndex := 0
	for i, ps := range r.playerStates {
		if ps.username == from {
			senderIndex = i
			return
		}
	}
	if clientMessage.Message == r.currentWord && !r.playerStates[senderIndex].hasGuessed && r.phase == PHASE_DRAWING {
		serverPacket := &protobuf.ServerPacket{
			Payload: &protobuf.ServerPacket_PlayerGuessedTheWord_{
				PlayerGuessedTheWord: &protobuf.ServerPacket_PlayerGuessedTheWord{
					Username: from,
				},
			},
		}
		r.playerStates[senderIndex].scoreIncrement = (len(r.playerStates) - r.guessersCount) * 100
		r.playerStates[senderIndex].hasGuessed = true
		r.guessersCount++
		r.broadcastToAll(serverPacket)
		if len(r.playerStates) == r.guessersCount {
			r.transitionToTurnSummary()
		}
		return
	}
	serverPacket := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_PlayerMessage_{
			PlayerMessage: &protobuf.ServerPacket_PlayerMessage{
				From:    from,
				Message: clientMessage.Message,
			},
		},
		ServerTimestamp: time.Now().UnixMilli(),
	}

	bytesPacket, err := proto.Marshal(serverPacket)

	if err != nil {
		return
	}

	if r.playerStates[senderIndex].hasGuessed {
		for i, ps := range r.playerStates {
			if ps.hasGuessed || i == r.drawerIndex {
				r.dataSendTasks = append(r.dataSendTasks, dataSendTask{to: ps.player, data: bytesPacket})
			}
		}
	} else {
		for _, ps := range r.playerStates {
			r.dataSendTasks = append(r.dataSendTasks, dataSendTask{to: ps.player, data: bytesPacket})
		}
	}
}

func (r *room) handleTick(now time.Time) {
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

func (r *room) transitionToChoosingWord() {
	r.phase = PHASE_CHOOSING_WORD
	r.currentWord = ""
	r.guessersCount = 0
	for _, ps := range r.playerStates {
		ps.hasGuessed = false
	}
	if r.currentDrawer == "" {
		r.drawerIndex = len(r.playerStates) - 1
	} else if r.drawerIndex == 0 {
		r.transitionToNextRound()
		return
	} else {
		r.drawerIndex--
	}
	r.currentDrawer = r.playerStates[r.drawerIndex].username

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
				Username: r.currentDrawer,
			},
		},
	}

	r.broadcastTo(plzChoose, r.playerStates[r.drawerIndex].player)
	r.broadcastToAllExcept(playerIsChoosing, r.playerStates[r.drawerIndex].player)
	r.nextTick = time.Now().Add(r.choosingWordDuration)
}

func (r *room) transitionToDrawing() {
	r.phase = PHASE_DRAWING
	if r.currentWord == "" {
		r.currentWord = r.wordChoices[0]
	}

	drawerState := r.playerStates[r.drawerIndex]

	playerStartedDrawing := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_PlayerIsDrawing_{
			PlayerIsDrawing: &protobuf.ServerPacket_PlayerIsDrawing{
				Username: drawerState.username,
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

	r.broadcastToAllExcept(playerStartedDrawing, drawerState.player)
	r.broadcastTo(yourTurn, drawerState.player)
	r.nextTick = time.Now().Add(r.drawingDuration)
}

func (r *room) transitionToTurnSummary() {
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

	for _, ps := range r.playerStates {
		x.Deltas = append(x.Deltas, &protobuf.ServerPacket_TurnSummary_ScoreDeltas{
			ScoreDelta: int64(ps.scoreIncrement),
			Username:   ps.username,
		})
	}

	r.broadcastToAll(turnSummary)
	r.nextTick = time.Now().Add(5 * time.Second)
}

func (r *room) transitionToNextRound() {
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

func (r *room) transitionToGameEnd() {
	r.phase = PHASE_GAMEEND
	leaderboard := &protobuf.ServerPacket{
		Payload: &protobuf.ServerPacket_Leaderboard{},
	}

	r.broadcastToAll(leaderboard)
	r.clearResources()
}

func (r *room) clearResources() {
	for _, ps := range r.playerStates {
		ps.player.CancelAndRelease()
	}

	r.parentLobby.RemoveRoom(r.id)
	r.playerStates = nil
	r.wordChoices = nil
	r.drawingHistory = nil
}

func (r *room) broadcastToAll(serverPacket *protobuf.ServerPacket) {
	bytesPacket, err := proto.Marshal(serverPacket)

	if err != nil {
		return
	}

	for _, ps := range r.playerStates {
		r.dataSendTasks = append(r.dataSendTasks, dataSendTask{to: ps.player, data: bytesPacket})
	}
}

func (r *room) broadcastTo(serverPacket *protobuf.ServerPacket, player Player) {
	bytesPacket, err := proto.Marshal(serverPacket)

	if err != nil {
		return
	}

	for _, ps := range r.playerStates {
		if ps.player == player {
			r.dataSendTasks = append(r.dataSendTasks, dataSendTask{to: ps.player, data: bytesPacket})
			return
		}
	}
}

func (r *room) broadcastToAllExcept(serverPacket *protobuf.ServerPacket, player Player) {
	bytesPacket, err := proto.Marshal(serverPacket)

	if err != nil {
		return
	}

	for _, ps := range r.playerStates {
		if ps.player != player {
			r.dataSendTasks = append(r.dataSendTasks, dataSendTask{to: ps.player, data: bytesPacket})
		}
	}
}

func (r *room) updateDescription() {
	if r.private {
		return
	}
	desc := roomDescription{
		id:           r.id,
		playersCount: len(r.playerStates),
		maxPlayers:   r.maxPlayers,
		started:      r.phase != PHASE_PENDING,
	}
	r.parentLobby.RequestUpdateDescription(desc)
}
