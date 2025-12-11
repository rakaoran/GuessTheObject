//lint:file-ignore U1000 This file contains legacy code that cannot be refactored at this time.

package game

import (
	"api/internal/shared/logger"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

// --- Game States ---
// These constants define the different states a game room can be in.
const (
	STATE_PENDING       = iota // Waiting for players to join before starting.
	STATE_CHOOSING_WORD        // A player is currently choosing a word to draw.
	STATE_DRAWING              // A player is drawing, and others are guessing.
	STATE_TURN_SUMMARY         // Showing the results of the turn (who guessed, scores).
	STATE_LEADERBOARD          // Game has ended, showing the final scores.
)

const (
	SERIAL_EVENT = iota
	SERIAL_MESSAGE
	SERIAL_DRAWING
	SERIAL_WORD_CHOICE
	SERIAL_TURN_SUMMARY
	SERIAL_INITIAL_PLAYERS_AND_SCORES
	SERIAL_GAME_ID
)

const (
	EVENT_PLAYER_JOINED = iota
	EVENT_PLAYER_RECONNECTED
	EVENT_PLAYER_LEFT
	EVENT_PlAYER_CHOOSING_WORD
	EVENT_GAME_STARTED
	EVENT_PLEASE_CHOOSE_WORD
	EVENT_PLAYER_STARTED_DRAWING
	EVENT_PLAYER_HAS_GUESSED_THE_WORD
	EVENT_TURN_SUMMARY
	EVENT_NEXT_ROUND
	EVENT_LEADERBOARD
)

// --- Game Constants ---
const MAX_PLAYERS = 8                           // Maximum number of players allowed in a room.
const CHOOSING_WORD_DURATION = time.Second * 10 // Time the drawer has to choose a word.
const DRAWING_DURATION = time.Second * 80       // Time the drawer has to draw the word.
const TURN_SUMMARY_DURATION = time.Second * 5   // How long the turn summary is shown.
const ROUNDS_COUNT = 3                          // Total number of rounds in a game.
const PENDING_DURATION = time.Second * 3600

// GameRoom represents a single game session.
type GameRoom struct {
	players          []*Player    // List of all players in the room.
	id               string       // Unique identifier for the game room.
	drawerIndex      int          // Index in the `players` slice for the current drawer.
	wordChoices      []string     // The 3 words offered to the drawer.
	currentWord      string       // The word that was chosen and is being drawn.
	state            int          // The current state of the game (e.g., STATE_DRAWING).
	round            int          // The current round number.
	locker           sync.RWMutex // Mutex to prevent race conditions when accessing game state.
	startOnce        sync.Once    // Ensures the game start logic runs only once.
	endOnce          sync.Once    // Ensures the game end logic runs only once.
	nextTick         time.Time    // The timestamp for the next scheduled state change (e.g., end of turn).
	matchmakingIndex int
	guessersCount    int
	history          [][]byte
}

func newGameroom() *GameRoom {
	id := uuid.NewString()
	logger.Infof("[GameRoom %s] Creating gameroom on order", id)
	return &GameRoom{
		players:          make([]*Player, 0, MAX_PLAYERS),
		id:               id,
		state:            STATE_PENDING,
		round:            0,
		nextTick:         time.Now().Add(PENDING_DURATION),
		matchmakingIndex: -1,
	}
}

func (gameroom *GameRoom) setMatchmakingIndex(index int) {
	gameroom.matchmakingIndex = index
	logger.Debugf("[GameRoom %s] Matchmaking index set to %d", gameroom.id, index)
}

func (gameroom *GameRoom) Lock() {
	logger.Infof("[GameRoom %s] LOCK acquired", gameroom.id)
	gameroom.locker.Lock()
}
func (gameroom *GameRoom) Unlock() {
	logger.Infof("[GameRoom %s] LOCK released", gameroom.id)
	gameroom.locker.Unlock()
}
func (gameroom *GameRoom) RLock() {
	logger.Infof("[GameRoom %s] RLOCK acquired", gameroom.id)
	gameroom.locker.RLock()
}
func (gameroom *GameRoom) RUnlock() {
	logger.Infof("[GameRoom %s] RLOCK released", gameroom.id)
	gameroom.locker.RUnlock()
}

func (gameroom *GameRoom) numPlayers() int {
	return len(gameroom.players)
}

func (gameroom *GameRoom) start() {
	gameroom.startOnce.Do(func() {
		gameStarted, err := proto.Marshal(&Event{Type: EVENT_GAME_STARTED})
		if err != nil {
			logger.Criticalf("[GameRoom %s] Failed to marshal game started event: %v", gameroom.id, err)
			return
		}
		gameStarted = append(gameStarted, SERIAL_EVENT)
		for _, player := range gameroom.players {
			player.writeChan <- gameStarted
		}
		gameroom.transitionToNextRound()
		logger.Infof("[GameRoom %s] Gameroom has started successfully", gameroom.id)
	})
}

func (gameroom *GameRoom) broadcastMessage(message *Message, senderGuessed bool) {
	// logger.Debugf("[GameRoom %s] Broadcasting message from %s (SenderGuessed: %v)", gameroom.id, message.From, senderGuessed)
	bytes, err := proto.Marshal(message)
	if err != nil {
		logger.Criticalf("[GameRoom %s] Failed to marshal message: %v", gameroom.id, err)
		return
	}

	bytes = append(bytes, SERIAL_MESSAGE)
	for _, player := range gameroom.players {
		if (!player.guessedTheWord && senderGuessed) || player.username == message.From {
			continue
		}
		player.writeChan <- bytes
	}
}

func (gameroom *GameRoom) broadcastDrawing(bytes []byte, drawer *Player) {
	// High frequency log, keep commented out unless debugging drawing specifically
	// logger.Debugf("[GameRoom %s] Broadcasting drawing data from %s", gameroom.id, drawer.username)

	for _, player := range gameroom.players {
		if drawer != player {
			player.writeChan <- bytes
		}
	}

	gameroom.history = append(gameroom.history, bytes)
}

func (gameroom *GameRoom) broadcastEvent(event *Event) {
	logger.Infof("[GameRoom %s] Broadcasting Event Type: %d, Data: %s", gameroom.id, event.Type, event.Data)
	bytes, err := proto.Marshal(event)
	if err != nil {
		logger.Criticalf("[GameRoom %s] Failed to marshal event type %d: %v", gameroom.id, event.Type, err)
		return
	}
	bytes = append(bytes, SERIAL_EVENT)
	for _, player := range gameroom.players {
		player.writeChan <- bytes
	}

	if event.Type == EVENT_PLAYER_HAS_GUESSED_THE_WORD || event.Type == EVENT_PlAYER_CHOOSING_WORD || event.Type == EVENT_PLAYER_STARTED_DRAWING {
		gameroom.history = append(gameroom.history, bytes)
	}
}

func (gameroom *GameRoom) broadcastTurnSummary(summary *TurnSummary) {
	logger.Infof("[GameRoom %s] Broadcasting Turn Summary. Scores: %v", gameroom.id, summary.Scores)
	summaryBytes, err := proto.Marshal(summary)
	if err != nil {
		logger.Criticalf("[GameRoom %s] Failed to marshal turn summary: %v", gameroom.id, err)
		return
	}
	summaryBytes = append(summaryBytes, SERIAL_TURN_SUMMARY)
	for _, player := range gameroom.players {
		player.writeChan <- summaryBytes
	}
	gameroom.history = append(gameroom.history, summaryBytes)
}

func (gameroom *GameRoom) addPlayer(player *Player) error {
	gameroom.players = append(gameroom.players, player)
	for _, p := range gameroom.players {
		if p.socket != player.socket && p.username == player.username {
			println("[GameRoom %s] Kicking zombie player %s", gameroom.id, p.username)
			gameroom.removePlayer(p)
			globalPlayers.removePlayerLocked(p)
			break
		}
	}
	if len(gameroom.players) >= MAX_PLAYERS {
		return errors.New("full")
	}
	logger.Infof("[GameRoom %s] Adding player: %s. Current count before add: %d", gameroom.id, player.username, len(gameroom.players))
	player.room = gameroom
	gameroom.broadcastEvent(&Event{Type: EVENT_PLAYER_JOINED, Data: player.username})

	if len(gameroom.players) >= 2 {
		logger.Infof("[GameRoom %s] Room full (or enough players), triggering start.", gameroom.id)
		initialPlayersAndScores := &TurnSummary{}

		for _, player := range gameroom.players {
			initialPlayersAndScores.Scores = append(initialPlayersAndScores.Scores, int32(player.score))
			initialPlayersAndScores.Usernames = append(initialPlayersAndScores.Usernames, player.username)
		}
		logger.Infof("[GameRoom %s] Broadcasting Turn Summary. Scores: %v", gameroom.id, initialPlayersAndScores.Scores)
		initialPlayersAndScoresBytes, _ := proto.Marshal(initialPlayersAndScores)

		initialPlayersAndScoresBytes = append(initialPlayersAndScoresBytes, SERIAL_INITIAL_PLAYERS_AND_SCORES)

		player.writeChan <- initialPlayersAndScoresBytes
		for _, move := range gameroom.history {
			player.writeChan <- move
		}

		gameroom.start()
	} else {
		logger.Infof("[GameRoom %s] Waiting for more players. Running sync.", gameroom.id)

	}

	globalPlayers.addPlayerLocked(player)
	gameidMessage := &Message{
		Content: gameroom.id,
	}

	gameidBytes, _ := proto.Marshal(gameidMessage)
	gameidBytes = append(gameidBytes, SERIAL_GAME_ID)
	player.writeChan <- gameidBytes

	return nil
}

func (gameroom *GameRoom) removePlayer(player *Player) {
	logger.Infof("[GameRoom %s] Attempting to remove player: %s", gameroom.id, player.username)
	for i, p := range gameroom.players {
		if p == player {
			gameroom.players = append(gameroom.players[:i], gameroom.players[i+1:]...)
			logger.Infof("[GameRoom %s] Player %s removed. Remaining count: %d", gameroom.id, player.username, len(gameroom.players))

			if len(gameroom.players) <= 1 {
				logger.Infof("[GameRoom %s] Not enough players left. Removing game from matchmaking.", gameroom.id)
				matchmaking.removeGame(gameroom)
				gameroom.endGame()
			} else {
				logger.Infof("[GameRoom %s] Rebalancing matchmaking for index %d", gameroom.id, gameroom.matchmakingIndex)
				matchmaking.rebalanceAfterRemovingPlayer(gameroom.matchmakingIndex)
			}

			if p.guessedTheWord {
				gameroom.guessersCount--
				logger.Infof("[GameRoom %s] Removed player was a guesser. New guessers count: %d", gameroom.id, gameroom.guessersCount)
			}
			if i == gameroom.drawerIndex {
				logger.Infof("[GameRoom %s] Removed player was the drawer. Transitioning to summary.", gameroom.id)
				gameroom.transitionToTurnSummary()
			} else if i < gameroom.drawerIndex {
				gameroom.drawerIndex--
				logger.Infof("[GameRoom %s] Removed player was before drawer. Adjusted drawerIndex to %d", gameroom.id, gameroom.drawerIndex)
			}
			if gameroom.guessersCount == gameroom.numPlayers() {
				gameroom.transitionToTurnSummary()
			}
			break
		}
	}
	globalPlayers.removePlayerLocked(player)
	player.Destroy()
	gameroom.broadcastEvent(&Event{Type: EVENT_PLAYER_LEFT, Data: player.username})
}

func (gameroom *GameRoom) transitionToChoosingWord() {
	logger.Infof("[GameRoom %s] Attempting transition to CHOOSING_WORD. Current State: %d", gameroom.id, gameroom.state)
	defer logger.Infof("[GameRoom %s] Transitioned to CHOOSING_WORD completed.", gameroom.id)

	if gameroom.state != STATE_PENDING && gameroom.state != STATE_TURN_SUMMARY {
		logger.Warningf("[GameRoom %s] Invalid state transition to CHOOSING_WORD from %d", gameroom.id, gameroom.state)
		return
	}

	if gameroom.state == STATE_CHOOSING_WORD {
		logger.Warningf("[GameRoom %s] Already in CHOOSING_WORD state", gameroom.id)
		return
	}

	gameroom.drawerIndex--
	logger.Infof("[GameRoom %s] Decremented drawerIndex. New value: %d", gameroom.id, gameroom.drawerIndex)

	if gameroom.drawerIndex < 0 {
		logger.Infof("[GameRoom %s] DrawerIndex < 0. Round cycle complete. Transitioning to next round.", gameroom.id)
		gameroom.transitionToNextRound()
		return
	}
	gameroom.state = STATE_CHOOSING_WORD

	for _, player := range gameroom.players {
		player.guessedTheWord = false
	}
	gameroom.guessersCount = 0

	// Sanity check for empty words list
	if len(wordsList) == 0 {
		logger.Criticalf("[GameRoom %s] CRITICAL: wordsList is empty!", gameroom.id)
		return
	}

	gameroom.wordChoices = []string{
		wordsList[rand.Intn(len(wordsList))],
		wordsList[rand.Intn(len(wordsList))],
		wordsList[rand.Intn(len(wordsList))],
	}
	logger.Infof("[GameRoom %s] Generated words: %v", gameroom.id, gameroom.wordChoices)

	drawer := gameroom.players[gameroom.drawerIndex]
	logger.Infof("[GameRoom %s] New Drawer is: %s", gameroom.id, drawer.username)

	gameroom.currentWord = gameroom.wordChoices[0]

	pleaseChooseWordBytes, _ := proto.Marshal(&Event{
		Type: EVENT_PLEASE_CHOOSE_WORD,
		Data: gameroom.wordChoices[0] + ":" + gameroom.wordChoices[1] + ":" + gameroom.wordChoices[2],
	})
	pleaseChooseWordBytes = append(pleaseChooseWordBytes, SERIAL_EVENT)

	playerIsChoosingBytes, _ := proto.Marshal(&Event{
		Type: EVENT_PlAYER_CHOOSING_WORD,
		Data: drawer.username,
	})
	playerIsChoosingBytes = append(playerIsChoosingBytes, SERIAL_EVENT)

	drawer.writeChan <- pleaseChooseWordBytes
	drawer.guessedTheWord = true
	gameroom.guessersCount++
	gameroom.history = append(gameroom.history, playerIsChoosingBytes)
	// Keeping your original log but cleaning it up slightly
	logger.Warningf("[GameRoom %s] written words: %s, %s, %s", gameroom.id, gameroom.wordChoices[0], gameroom.wordChoices[1], gameroom.wordChoices[2])

	for _, player := range gameroom.players {
		if player.username != drawer.username {
			player.writeChan <- playerIsChoosingBytes
		}
	}

	gameroom.nextTick = time.Now().Add(CHOOSING_WORD_DURATION)
	logger.Infof("[GameRoom %s] Next tick set for CHOOSING_WORD: %v", gameroom.id, gameroom.nextTick)
}

func (gameroom *GameRoom) transitionToDrawing() {
	logger.Infof("[GameRoom %s] Attempting transition to DRAWING. Current State: %d", gameroom.id, gameroom.state)
	defer logger.Infof("[GameRoom %s] Transitioned to DRAWING completed", gameroom.id)

	if gameroom.state != STATE_CHOOSING_WORD {
		logger.Criticalf("[GameRoom %s] PANIC: Transitioning to drawing from invalid state %d", gameroom.id, gameroom.state)
		panic("Transitioning to drawing from a state that isn't choosing word")
	}

	if gameroom.state == STATE_DRAWING {
		logger.Warningf("[GameRoom %s] Already in DRAWING state", gameroom.id)
		return
	}

	gameroom.state = STATE_DRAWING
	println(gameroom.currentWord)
	logger.Infof("[GameRoom %s] Word chosen was: %s", gameroom.id, gameroom.currentWord)

	playerDrawing := &Event{
		Type: EVENT_PLAYER_STARTED_DRAWING,
		Data: gameroom.players[gameroom.drawerIndex].username,
	}
	gameroom.broadcastEvent(playerDrawing)
	gameroom.nextTick = time.Now().Add(DRAWING_DURATION)
	logger.Infof("[GameRoom %s] Next tick set for DRAWING: %v", gameroom.id, gameroom.nextTick)
}

func (gameroom *GameRoom) transitionToTurnSummary() {
	logger.Infof("[GameRoom %s] Attempting transition to TURN_SUMMARY. Current State: %d", gameroom.id, gameroom.state)
	defer logger.Infof("[GameRoom %s] Transitioned to TURN_SUMMARY completed", gameroom.id)

	if gameroom.state == STATE_TURN_SUMMARY {
		logger.Warningf("[GameRoom %s] Already in TURN_SUMMARY state", gameroom.id)
		return
	}
	gameroom.history = make([][]byte, 0)
	gameroom.state = STATE_TURN_SUMMARY

	turnSummary := &TurnSummary{}

	for _, player := range gameroom.players {
		turnSummary.Scores = append(turnSummary.Scores, int32(player.score))
		turnSummary.Usernames = append(turnSummary.Usernames, player.username)
	}

	gameroom.broadcastTurnSummary(turnSummary)

	gameroom.nextTick = time.Now().Add(TURN_SUMMARY_DURATION)
	logger.Infof("[GameRoom %s] Next tick set for TURN_SUMMARY: %v", gameroom.id, gameroom.nextTick)
}

func (gameroom *GameRoom) transitionToNextRound() {
	logger.Infof("[GameRoom %s] Attempting transition to NEXT_ROUND. Current Round: %d", gameroom.id, gameroom.round)
	defer logger.Infof("[GameRoom %s] Transitioned to NEXT_ROUND completed", gameroom.id)

	gameroom.drawerIndex = len(gameroom.players)
	gameroom.round++
	logger.Infof("[GameRoom %s] Starting Round %d. Drawer Index reset to %d", gameroom.id, gameroom.round, gameroom.drawerIndex)

	if gameroom.round > ROUNDS_COUNT {
		logger.Infof("[GameRoom %s] Max rounds (%d) reached. Going to Leaderboard.", gameroom.id, ROUNDS_COUNT)
		gameroom.transitionToLeaderboard()
		return
	}

	gameroom.broadcastEvent(&Event{
		Type: EVENT_NEXT_ROUND,
	})

	gameroom.transitionToChoosingWord()
}

func (gameroom *GameRoom) transitionToLeaderboard() {
	logger.Infof("[GameRoom %s] Transitioning to LEADERBOARD.", gameroom.id)
	defer logger.Infof("[GameRoom %s] Transitioned to LEADERBOARD completed", gameroom.id)

	if gameroom.state == STATE_LEADERBOARD {
		return
	}
	gameroom.broadcastEvent(&Event{Type: EVENT_LEADERBOARD})
	time.Sleep(50 * time.Millisecond)
	gameroom.endGame()
}

func (gameroom *GameRoom) endGame() {
	logger.Infof("[GameRoom %s] ENDING GAME. Cleaning up...", gameroom.id)
	matchmaking.removeGame(gameroom)

	for _, p := range gameroom.players {
		p.Destroy()
	}
	gameroom.players = nil
	logger.Infof("[GameRoom %s] Game ended and players cleared.", gameroom.id)
}

func (gameroom *GameRoom) handleTick() {

	if gameroom.nextTick.After(time.Now()) {
		// logger.Debugf("[GameRoom %s] Tick check: Not yet time. Remaining: %v", gameroom.id, time.Until(gameroom.nextTick))
		return
	}

	logger.Infof("[GameRoom %s] Timer expired for State %d. Triggering transition.", gameroom.id, gameroom.state)

	switch gameroom.state {
	case STATE_PENDING:
		logger.Infof("[GameRoom %s] PENDING timer expired. Removing game.", gameroom.id)
		matchmaking.Lock()
		matchmaking.removeGame(gameroom)
		matchmaking.Unlock()
	case STATE_CHOOSING_WORD:
		logger.Infof("[GameRoom %s] CHOOSING_WORD timer expired. Force transition to Drawing.", gameroom.id)
		gameroom.transitionToDrawing()
	case STATE_DRAWING:
		logger.Infof("[GameRoom %s] DRAWING timer expired. Transitioning to Summary.", gameroom.id)
		gameroom.transitionToTurnSummary()
	case STATE_TURN_SUMMARY:
		logger.Infof("[GameRoom %s] TURN_SUMMARY timer expired. Transitioning to Choosing Word.", gameroom.id)
		gameroom.transitionToChoosingWord()
	}
}
