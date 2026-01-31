package game

import (
	"api/domain"
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Origin check is handled by the main CSRF middleware
		return true
	},
}

func NewGameHandler(
	lobby Lobby,
	userGetter UserGetter,
	randomWordsGenerator RandomWordsGenerator,
) *GameHandler {
	return &GameHandler{
		lobby:                lobby,
		userGetter:           userGetter,
		randomWordsGenerator: randomWordsGenerator,
	}
}

func validateCreateGameRequest(req CreateGameRequest) error {
	if req.MaxPlayers < 2 {
		return errors.New("maxPlayers must be at least 2")
	}
	if req.MaxPlayers > 20 {
		return errors.New("maxPlayers cannot exceed 20")
	}
	if req.RoundsCount < 1 {
		return errors.New("roundsCount must be at least 1")
	}
	if req.RoundsCount > 10 {
		return errors.New("roundsCount cannot exceed 10")
	}
	if req.WordsCount < 1 {
		return errors.New("wordsCount must be at least 1")
	}
	if req.WordsCount > 5 {
		return errors.New("wordsCount cannot exceed 5")
	}
	if req.ChoosingWordDuration < 5 {
		return errors.New("choosingWordDuration must be at least 5 seconds")
	}
	if req.ChoosingWordDuration > 120 {
		return errors.New("choosingWordDuration cannot exceed 120 seconds")
	}
	if req.DrawingDuration < 30 {
		return errors.New("drawingDuration must be at least 30 seconds")
	}
	if req.DrawingDuration > 300 {
		return errors.New("drawingDuration cannot exceed 300 seconds")
	}
	return nil
}

type CreateGameRequest struct {
	Private              bool  `form:"private"`
	MaxPlayers           int   `form:"maxPlayers"`
	RoundsCount          int   `form:"roundsCount"`
	WordsCount           int   `form:"wordsCount"`
	ChoosingWordDuration int64 `form:"choosingWordDuration"` // in seconds
	DrawingDuration      int64 `form:"drawingDuration"`      // in seconds
}

func (gh *GameHandler) CreateGameHandler(ctx *gin.Context) {
	userId, exists := ctx.Get("id")
	if !exists {
		ctx.String(http.StatusUnauthorized, "unauthenticated")
		return
	}

	userIdStr, ok := userId.(string)
	if !ok {
		ctx.String(http.StatusInternalServerError, "invalid-user-id")
		return
	}

	var req CreateGameRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.String(http.StatusBadRequest, "invalid-request-format")
		return
	}

	if err := validateCreateGameRequest(req); err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	user, err := gh.userGetter.GetUserById(ctx.Request.Context(), userIdStr)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			ctx.String(http.StatusUnauthorized, "user-not-found")
			return
		}
		ctx.String(http.StatusInternalServerError, "failed-to-get-user")
		return
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}
	wsConn := NewGorillaWebSocketWrapper(conn)
	player := NewPlayer(userIdStr, user.Username)

	room := NewRoom(
		player,
		req.Private,
		req.MaxPlayers,
		req.RoundsCount,
		req.WordsCount,
		time.Duration(req.ChoosingWordDuration)*time.Second,
		time.Duration(req.DrawingDuration)*time.Second,
		gh.randomWordsGenerator,
	)

	gh.lobby.RequestAddAndRunRoom(ctx.Request.Context(), room)

	go player.ReadPump(wsConn)
	go player.WritePump(wsConn)
}

func (gh *GameHandler) JoinGameHandler(ctx *gin.Context) {
	userId, exists := ctx.Get("id")
	if !exists {
		ctx.String(http.StatusUnauthorized, "unauthenticated")
		return
	}

	userIdStr, ok := userId.(string)
	if !ok {
		ctx.String(http.StatusInternalServerError, "invalid-user-id")
		return
	}

	roomId := ctx.Param("roomid")
	if roomId == "" {
		ctx.String(http.StatusBadRequest, "missing-room-id")
		return
	}

	user, err := gh.userGetter.GetUserById(ctx.Request.Context(), userIdStr)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			ctx.String(http.StatusUnauthorized, "user-not-found")
			return
		}
		ctx.String(http.StatusInternalServerError, "failed-to-get-user")
		return
	}

	player := NewPlayer(userIdStr, user.Username)

	errChan := make(chan error, 1)
	joinReq := roomJoinRequest{
		ctx:     context.Background(),
		roomId:  roomId,
		player:  player,
		errChan: errChan,
	}

	gh.lobby.ForwardPlayerJoinRequestToRoom(ctx.Request.Context(), joinReq)

	err = <-errChan
	if err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		ctx.Abort()
		return
	}
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}

	wsConn := NewGorillaWebSocketWrapper(conn)
	go player.ReadPump(wsConn)
	go player.WritePump(wsConn)
}

type PublicGameResponse struct {
	ID           string `json:"id"`
	Private      bool   `json:"private"`
	PlayersCount int    `json:"playersCount"`
	MaxPlayers   int    `json:"maxPlayers"`
	Started      bool   `json:"started"`
}

func (gh *GameHandler) GetPublicGamesHandler(ctx *gin.Context) {
	userId, exists := ctx.Get("id")
	if !exists {
		ctx.String(http.StatusUnauthorized, "unauthenticated")
		return
	}

	_, ok := userId.(string)
	if !ok {
		ctx.String(http.StatusInternalServerError, "invalid-user-id")
		return
	}

	games := gh.lobby.GetPublicGames(ctx.Request.Context())
	response := make([]PublicGameResponse, 0, len(games))
	for _, g := range games {
		response = append(response, PublicGameResponse{
			ID:           g.id,
			Private:      g.private,
			PlayersCount: g.playersCount,
			MaxPlayers:   g.maxPlayers,
			Started:      g.started,
		})
	}

	ctx.JSON(http.StatusOK, response)
}

func NewGorillaWebSocketWrapper(conn *websocket.Conn) *GorillaWebSocketWrapper {
	return &GorillaWebSocketWrapper{conn: conn}
}

func (g *GorillaWebSocketWrapper) Close() {
	g.conn.Close()
}

func (g *GorillaWebSocketWrapper) Write(data []byte) error {
	return g.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (g *GorillaWebSocketWrapper) Read() ([]byte, error) {
	_, message, err := g.conn.ReadMessage()
	return message, err
}

func (g *GorillaWebSocketWrapper) Ping() error {
	return g.conn.WriteMessage(websocket.PingMessage, nil)
}
