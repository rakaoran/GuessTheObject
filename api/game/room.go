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
	hUsername := host.Username()
	r := &room{
		private: private,
		host:    hUsername,
		playerStates: []*playerGameState{
			{player: host, username: hUsername},
		},
		drawerIndex:           0,
		maxPlayers:            maxPlayers,
		roundsCount:           roundsCount,
		wordsCount:            wordsCount,
		phase:                 PHASE_PENDING,
		guessersCount:         0,
		nextTick:              time.Now().Add(time.Hour * 24),
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

	host.SetRoom(r)

	return r
}

/*
	Room interface implementation
*/

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
	m := protobuf.MakePacketInitialRoomSnapshot(nil, nil, "", 0, r.id, 0, 0, int64(r.choosingWordDuration.Seconds()), int64(r.drawingDuration.Seconds()))
	mb, _ := proto.Marshal(m)
	r.playerStates[0].player.Send(mb)
	for {
		if r.phase == PHASE_GAMEEND {
			return
		}
		select {
		case _, ok := <-r.pingPlayers:
			if !ok {
				return
			}
			r.bufferPingTasks()

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

		case jreq, ok := <-r.joinReqs:
			if !ok {
				return
			}

			r.handleJoinRequest(jreq)
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

func (r *room) bufferPingTasks() {
	for _, ps := range r.playerStates {
		r.pingSendTasks = append(r.pingSendTasks, pingSendTask{to: ps.player})
	}
}

func (r *room) handleJoinRequest(jreq roomJoinRequest) {
	if r.maxPlayers > len(r.playerStates) {
		r.addPlayer(jreq.player)
		close(jreq.errChan)
	} else {
		jreq.errChan <- ErrRoomFull
		close(jreq.errChan)
	}
}

func (r *room) addPlayer(p Player) error {
	if len(r.playerStates) >= r.maxPlayers {
		return ErrRoomFull
	}
	pUsername := p.Username()

	for _, ps := range r.playerStates {
		if ps.username == pUsername {
			r.handleRemovePlayer(ps.player)
			break
		}
	}

	pStates := make([]*protobuf.ServerPacket_InitialRoomSnapshot_PlayerState, 0, len(r.playerStates))
	for _, ps := range r.playerStates {
		pStates = append(pStates, &protobuf.ServerPacket_InitialRoomSnapshot_PlayerState{
			Username:  ps.username,
			Score:     int64(ps.score),
			IsGuesser: ps.hasGuessed,
		})
	}
	playerJoined := protobuf.MakePacketPlayerJoined(pUsername)
	r.broadcastToAll(playerJoined)
	initialRoomSnapshot := protobuf.MakePacketInitialRoomSnapshot(pStates, r.drawingHistory, r.currentDrawer, int32(r.round), r.id, int32(r.phase), r.nextTick.UnixMilli(), int64(r.choosingWordDuration.Seconds()), int64(r.drawingDuration.Seconds()))

	r.playerStates = append(r.playerStates, &playerGameState{username: pUsername, player: p})
	p.SetRoom(r)

	r.broadcastTo(initialRoomSnapshot, p)

	r.updateDescription()
	return nil
}

func (r *room) handleRemovePlayer(toRemove Player) {
	for i, ps := range r.playerStates {
		if ps.player == toRemove {
			r.playerStates = append(r.playerStates[0:i], r.playerStates[i+1:]...)
			toRemove.CancelAndRelease()
			if len(r.playerStates) <= 1 && r.phase != PHASE_PENDING {
				r.transitionToGameEnd()
				return
			}

			if len(r.playerStates) == 0 {
				r.transitionToGameEnd()
				return
			}

			if i < r.drawerIndex {
				r.drawerIndex--
			} else if i == r.drawerIndex {
				r.drawerIndex--
				if r.phase != PHASE_PENDING {
					r.transitionToChoosingWord()
				}
			}

			playerLeft := &protobuf.ServerPacket{
				Payload: &protobuf.ServerPacket_PlayerLeft_{
					PlayerLeft: &protobuf.ServerPacket_PlayerLeft{
						Username: ps.username,
					},
				},
				ServerTimestamp: time.Now().UnixMilli(),
			}
			r.broadcastToAll(playerLeft)
			r.updateDescription()
			return
		}
	}
}
func (r *room) handleTick(now time.Time) {
	if now.Before(r.nextTick) {
		return
	}

	switch r.phase {
	case PHASE_CHOOSING_WORD:
		r.transitionToDrawing()
	case PHASE_DRAWING:
		r.transitionToTurnSummary()
	case PHASE_TURN_SUMMARY:
		r.transitionToChoosingWord()
	case PHASE_GAMEEND:
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

/*
	Envelope handlers
*/

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
		pkt := protobuf.MakePacketDrawingData(drawingData.Data)

		r.broadcastToAll(pkt)
		r.drawingHistory = append(r.drawingHistory, drawingData.Data)
		return
	}

}

func (r *room) handleStartGameEnvelope(from string) {
	if r.phase != PHASE_PENDING {
		return
	}
	if r.host != from {
		return
	}

	pkt := protobuf.MakePacketGameStarted()
	r.broadcastToAll(pkt)
	r.round = 1
	r.transitionToChoosingWord()
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
	r.transitionToDrawing()
}

func (r *room) handlePlayerMessageEnvelope(clientMessage *protobuf.ClientPacket_PlayerMessage, from string) {
	senderIndex := 0
	for i, ps := range r.playerStates {
		if ps.username == from {
			senderIndex = i
			break
		}
	}
	if clientMessage.Message == r.currentWord && !r.playerStates[senderIndex].hasGuessed && r.phase == PHASE_DRAWING {
		serverPacket := protobuf.MakePacketPlayerGuessedTheWord(from)
		r.playerStates[senderIndex].scoreIncrement = (len(r.playerStates) - 1 - r.guessersCount) * 100
		r.playerStates[senderIndex].hasGuessed = true
		r.guessersCount++
		r.broadcastToAll(serverPacket)
		if len(r.playerStates)-1 == r.guessersCount {
			r.transitionToTurnSummary()
		}
		return
	}
	serverPacket := protobuf.MakePacketPlayerMessage(from, clientMessage.Message)

	bytesPacket, err := proto.Marshal(serverPacket)

	if err != nil {
		return
	}

	if r.playerStates[senderIndex].hasGuessed {
		for i, ps := range r.playerStates {
			if ps.username == from {
				continue
			}
			if ps.hasGuessed || i == r.drawerIndex {
				r.dataSendTasks = append(r.dataSendTasks, dataSendTask{to: ps.player, data: bytesPacket})
			}
		}
	} else {
		for _, ps := range r.playerStates {
			if ps.username == from {
				continue
			}
			r.dataSendTasks = append(r.dataSendTasks, dataSendTask{to: ps.player, data: bytesPacket})
		}
	}
}

/*
	Broadcasting Functions
*/

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

/*
	TRANSITIONS between states
*/

func (r *room) transitionToChoosingWord() {
	r.phase = PHASE_CHOOSING_WORD
	r.currentWord = ""
	r.guessersCount = 0
	for _, ps := range r.playerStates {
		ps.hasGuessed = false
		ps.score += ps.scoreIncrement
		ps.scoreIncrement = 0
	}
	if r.currentDrawer == "" {
		r.drawerIndex = len(r.playerStates) - 1
	} else if r.drawerIndex <= 0 {
		r.transitionToNextRound()
		return
	} else {
		r.drawerIndex--
	}
	r.currentDrawer = r.playerStates[r.drawerIndex].username

	words := r.randomWordsGenerator.Generate(r.wordsCount)
	r.wordChoices = words

	plzChoose := protobuf.MakePacketPleaseChooseAWord(words)

	playerIsChoosing := protobuf.MakePacketPlayerIsChoosingWord(r.currentDrawer)

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

	playerStartedDrawing := protobuf.MakePacketPlayerIsDrawing(drawerState.username)

	yourTurn := protobuf.MakePacketYourTurnToDraw(r.currentWord)

	r.broadcastToAllExcept(playerStartedDrawing, drawerState.player)
	r.broadcastTo(yourTurn, drawerState.player)
	r.nextTick = time.Now().Add(r.drawingDuration)
}

func (r *room) transitionToTurnSummary() {
	r.phase = PHASE_TURN_SUMMARY
	clear(r.drawingHistory)
	r.drawingHistory = r.drawingHistory[:0]

	deltas := []*protobuf.ServerPacket_TurnSummary_ScoreDeltas{}

	for _, ps := range r.playerStates {
		deltas = append(deltas, &protobuf.ServerPacket_TurnSummary_ScoreDeltas{
			ScoreDelta: int64(ps.scoreIncrement),
			Username:   ps.username,
		})
	}

	turnSummary := protobuf.MakePacketTurnSummary(r.currentWord, deltas)

	r.broadcastToAll(turnSummary)
	r.nextTick = time.Now().Add(5 * time.Second)
}

func (r *room) transitionToNextRound() {
	r.round++
	if r.round > r.roundsCount {
		r.transitionToGameEnd()
		return
	}
	nextRound := protobuf.MakePacketRoundUpdate(int64(r.round))

	r.currentDrawer = ""

	r.broadcastToAll(nextRound)
	r.transitionToChoosingWord()
}

func (r *room) transitionToGameEnd() {
	r.phase = PHASE_GAMEEND
	leaderboard := protobuf.MakePacketLeaderBoard()

	r.broadcastToAll(leaderboard)

	for _, ps := range r.playerStates {
		ps.player.CancelAndRelease()
	}

	r.parentLobby.RemoveRoom(r.id)
	r.playerStates = nil
	r.wordChoices = nil
	r.drawingHistory = nil
}
