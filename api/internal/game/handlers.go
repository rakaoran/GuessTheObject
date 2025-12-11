package game

import (
	"api/internal/shared/authorization"
	"api/internal/shared/configs"
	"api/internal/shared/database"
	"api/internal/shared/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func matchmakingHandler(ctx *gin.Context) {

	cookie, err := ctx.Cookie(configs.JWTCookie.Name)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "player-not-logged-in"})
		return
	}

	jwtData, ok := authorization.VerifyJWT(cookie)

	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad-token"})
		return
	}
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "websocket-upgrade-failed"})
	}

	playerData, _ := database.GetUserById(jwtData.Id)
	logger.Infof("Player: %s connected.", playerData.Username)
	// Setting up the player struct wrapper of the socket
	player := newPlayer(playerData.Username, conn, playerData.Id)
	matchmaking.Lock()
	matchmaking.assignPlayer(player)
	matchmaking.Unlock()

	go player.ReadPump()
	go player.WritePump()
}

func joinByLink(ctx *gin.Context) {
	gameid := ctx.Param("gameid")
	// Checking authentication and upgrading to websocket
	cookie, err := ctx.Cookie(configs.JWTCookie.Name)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "player-not-logged-in"})
		return
	}

	jwtData, ok := authorization.VerifyJWT(cookie)

	if !ok {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad-token"})
		return
	}
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "websocket-upgrade-failed"})
	}

	playerData, _ := database.GetUserById(jwtData.Id)
	logger.Infof("Player: %s connected.", playerData.Username)
	// Setting up the player struct wrapper of the socket
	player := newPlayer(playerData.Username, conn, playerData.Id)

	var foundRoom bool
	var roomFull bool = false

	matchmaking.Lock()
	for _, gr := range matchmaking.games() {
		if gr.id == gameid {
			err := matchmaking.assignPlayerTo(player, gr)
			foundRoom = true
			if err != nil {
				roomFull = true
			}
			break
		}
	}
	matchmaking.Unlock()

	if !foundRoom {
		// Close connection properly since we won't start pumps
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "room-not-found"))
		player.Destroy()
		return
	}

	if roomFull {
		// Close connection properly since we won't start pumps
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "room-full"))
		player.Destroy()
		return
	}

	go player.ReadPump()
	go player.WritePump()
}
