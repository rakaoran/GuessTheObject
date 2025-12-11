//lint:file-ignore U1000 This file contains legacy code that cannot be refactored at this time.

package game

import (
	"api/internal/shared/logger"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type Player struct {
	username       string          // Display name of the player
	socket         *websocket.Conn // WebSocket connection for real-time communication
	id             string          // Unique identifier for the player
	room           *GameRoom       // Current game room the player is in (nil if not in a room)
	limiter        rateLimiter     // Rate limiter to prevent message spam
	score          int             // Player's current score in the game
	guessedTheWord bool            // Flag indicating if player has guessed the word in current round
	writeChan      chan []byte
	index          int
	closeOnce      sync.Once
}

func newPlayer(username string, socket *websocket.Conn, id string) *Player {
	logger.Infof("[Player %s] Creating new player instance: %s", id, username)
	socket.SetPongHandler(func(appData string) error {
		socket.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	return &Player{
		username:  username,
		socket:    socket,
		id:        id,
		writeChan: make(chan []byte, 800),
		index:     -1,
	}
}

func (player *Player) Destroy() {
	player.closeOnce.Do(func() {
		logger.Infof("[Player %s] Destroying player instance.", player.username)
		player.socket.Close()
		player.room = nil
		player.socket = nil
		close(player.writeChan)
		player.writeChan = nil
	})
}

func (player *Player) ReadPump() {
	playersRoom := player.room
	logger.Infof("[Player %s] Starting ReadPump in Room %s", player.username, playersRoom.id)

	defer logger.Infof("[Player %s] ReadPump stopped.", player.username)
	socket := player.socket
	for {
		_, readData, err := socket.ReadMessage()
		if err != nil {
			println("READ ERROR SOCKET likely due to pong missing")
			matchmaking.Lock()
			playersRoom.Lock()
			playersRoom.removePlayer(player)
			playersRoom.Unlock()
			matchmaking.Unlock()
			break
		}

		if len(readData) == 0 {
			continue
		}

		messageType := readData[len(readData)-1]

		switch messageType {

		case SERIAL_MESSAGE:
			if !player.limiter.rateLimiterAllows() {
				logger.Warningf("[Player %s] Rate limit exceeded. Ignoring message.", player.username)
				// TODO: kick the hell out of him
				continue
			}
			actualMessage := &Message{}
			err := proto.Unmarshal(readData[:len(readData)-1], actualMessage)
			if err != nil {
				logger.Criticalf("[Player %s] Failed to unmarshal SERIAL_MESSAGE: %v", player.username, err)
				continue
			}

			playersRoom.Lock()
			// Check if the message is actually the hidden word (Guessing logic)
			if playersRoom.state == STATE_DRAWING &&
				strings.EqualFold(actualMessage.Content, playersRoom.currentWord) &&
				!player.guessedTheWord &&
				player != playersRoom.players[playersRoom.drawerIndex] {

				logger.Infof("[Player %s] GUESS CORRECT! Word: %s", player.username, playersRoom.currentWord)

				player.guessedTheWord = true
				playersRoom.guessersCount++
				playersRoom.broadcastEvent(&Event{Type: EVENT_PLAYER_HAS_GUESSED_THE_WORD, Data: player.username})
				player.score += 10 * (playersRoom.numPlayers() - playersRoom.guessersCount + 1)
				playersRoom.players[playersRoom.drawerIndex].score += 5
				logger.Infof("[Player %s] Score updated to %d. Guessers count: %d/%d", player.username, player.score, playersRoom.guessersCount, len(playersRoom.players))

				if playersRoom.guessersCount == len(playersRoom.players) {
					logger.Infof("[Player %s] All players guessed. Triggering Turn Summary.", player.username)
					playersRoom.transitionToTurnSummary()
				}
				playersRoom.Unlock()
			} else {
				// Regular chat message
				// logger.Debugf("[Player %s] Chat message: %s", player.id, actualMessage.Content)
				actualMessage.From = player.username
				playersRoom.broadcastMessage(actualMessage, player.guessedTheWord)
				playersRoom.Unlock()
			}

		case SERIAL_WORD_CHOICE:
			wordchoice := &WordChoice{}
			err := proto.Unmarshal(readData[:len(readData)-1], wordchoice)
			if err != nil {
				logger.Criticalf("[Player %s] Failed to unmarshal SERIAL_WORD_CHOICE: %v", player.username, err)
				continue
			}

			playersRoom.Lock()

			// Validation: Is it the right state? Is this player the drawer? Is index valid?
			if playersRoom.state != STATE_CHOOSING_WORD || player != playersRoom.players[playersRoom.drawerIndex] || wordchoice.WordIndex >= 3 || wordchoice.WordIndex < 0 {
				playersRoom.Unlock()
				continue
			}

			playersRoom.currentWord = playersRoom.wordChoices[wordchoice.WordIndex]
			logger.Infof("[Player %s] Word selected: %s (Index %d). Transitioning to DRAWING.", player.username, playersRoom.currentWord, wordchoice.WordIndex)

			playersRoom.transitionToDrawing()
			playersRoom.Unlock()

		case SERIAL_DRAWING:
			// High frequency log - careful with this one
			// logger.Debugf("[Player %s] Received drawing path bytes", player.id)
			if player.room.state != STATE_DRAWING || player.room.players[player.room.drawerIndex] != player {
				// logger.Warningf("[Player %s] Ignored drawing path. Not drawer or wrong state.", player.id)
				continue
			}
			playersRoom.Lock()
			playersRoom.broadcastDrawing(readData, player)
			playersRoom.Unlock()

		default:
			logger.Warningf("[Player %s] Received unknown message serial type: %d", player.id, messageType)
		}
	}

}
func (player *Player) WritePump() {
	logger.Infof("[Player %s] Starting WritePump", player.username)
	defer logger.Infof("[Player %s] WritePump stopped", player.username)
	playersRoom := player.room
	for {
		bytes, ok := <-player.writeChan
		if !ok { // channel was closed, nothing to do, player destroyed
			return
		}

		if len(bytes) == 0 { // 0 for ping sending
			err := player.socket.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				println("WRITE ERROR SOCKET likely due to disconnection")
				matchmaking.Lock()
				playersRoom.Lock()
				playersRoom.removePlayer(player)
				playersRoom.Unlock()
				matchmaking.Unlock()
				break
			}
			continue
		}
		err := player.socket.WriteMessage(websocket.BinaryMessage, bytes)
		if err != nil {
			println("WRITE ERROR SOCKET likely due to disconnection")
			matchmaking.Lock()
			playersRoom.Lock()
			playersRoom.removePlayer(player)
			playersRoom.Unlock()
			matchmaking.Unlock()
			break
		}
	}
}

type rateLimiter struct {
	lastAllow time.Time // Timestamp of the last allowed action
}

func (r *rateLimiter) rateLimiterAllows() bool {
	timeSinceLastAllow := time.Since(r.lastAllow)

	// Check if at least 500ms has passed since the last allowed action
	if timeSinceLastAllow > 500*time.Millisecond {
		// Keeping this debug log, but you might want to comment it out in prod to save space
		// logger.Infof("[rateLimiter] Action allowed (%.2f seconds since last action)", timeSinceLastAllow.Seconds())
		r.lastAllow = time.Now()
		return true
	} else {
		logger.Warningf("[rateLimiter] Action blocked (only %.2f seconds since last action, need 0.5s)",
			timeSinceLastAllow.Seconds())
		return false
	}
}

type globalPlayersType struct {
	players map[string]*Player
	locker  sync.RWMutex
}

var globalPlayers = globalPlayersType{
	players: make(map[string]*Player),
}

func (gp *globalPlayersType) addPlayerLocked(p *Player) {
	gp.locker.Lock()
	defer gp.locker.Unlock()

	gp.players[p.username] = p
}

func (gp *globalPlayersType) removePlayerLocked(p *Player) {
	gp.locker.Lock()
	defer gp.locker.Unlock()

	delete(gp.players, p.username)
}

func (gp *globalPlayersType) ping() {
	gp.locker.Lock()
	defer gp.locker.Unlock()

	for _, p := range gp.players {
		p.writeChan <- make([]byte, 0) // empty packet, your “ping”
	}
}

func (gp *globalPlayersType) getPlayer(username string) *Player {
	gp.locker.RLock()
	defer gp.locker.RUnlock()

	return gp.players[username]
}
