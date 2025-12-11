package game

import (
	"api/internal/shared/logger"
	"time"
)

func StartTickers() {
	matchmakingTicker := time.NewTicker(time.Second)
	playersPingTicker := time.NewTicker(4 * time.Second)
	go func() {
		for range matchmakingTicker.C {
			matchmaking.Lock()
			rooms := make([]*GameRoom, 0, len(matchmaking.games()))
			rooms = append(rooms, matchmaking.games()...)
			matchmaking.Unlock()
			logger.Infof("Matchmaking size %d", len(matchmaking.games()))

			for _, room := range rooms {
				room.Lock()
				room.handleTick()
				room.Unlock()
			}
		}
	}()

	go func() {
		for range playersPingTicker.C {
			globalPlayers.ping()
		}
	}()
}
