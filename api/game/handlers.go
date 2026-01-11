package game

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type websocketConnection struct {
	socket *websocket.Conn
}

func (wc *websocketConnection) Write(data []byte) error {
	return wc.socket.WriteMessage(websocket.BinaryMessage, data)
}

func (wc *websocketConnection) Ping() error {
	return wc.socket.WriteMessage(websocket.PingMessage, nil)
}

func (wc *websocketConnection) Read() ([]byte, error) {
	_, p, err := wc.socket.ReadMessage()
	return p, err
}

func (wc *websocketConnection) Close(errCode string) {
	wc.socket.SetWriteDeadline(time.Now().Add(time.Second * 20))
	wc.socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, errCode))
	wc.socket.Close()
}

func NewWebsocketConnection(conn *websocket.Conn) websocketConnection {
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(time.Minute))
		return nil
	})
	return websocketConnection{conn}
}

type GameHanler struct {
	gameService *service
}

func (h *GameHanler) CreateRoomHandler(ctx *gin.Context) {
	id := ctx.GetString("id")

	if id == "" {
		slog.Error("Unexpected error, id not found. What is the middleware doing?",
			"ip", ctx.ClientIP(),
			"user_agent", ctx.Request.UserAgent(),
		)

		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "unknown-error"})
		return
	}

	configs := RoomConfigs{}

	err := ctx.ShouldBindJSON(&configs)

	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid-configs"})
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)

	if err != nil {
		slog.Error("WS upgrade failed ")
	}

	socketConn := NewWebsocketConnection(conn)
	private := ctx.Query("private") == "true"
	h.gameService.CreateRoom(ctx.Request.Context(), id, &socketConn, private, configs)
}
