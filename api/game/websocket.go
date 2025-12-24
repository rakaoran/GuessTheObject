package game

import (
	"time"

	"github.com/gorilla/websocket"
)

type WebsocketConnection struct {
	socket *websocket.Conn
}

func (wc *WebsocketConnection) Write(data []byte) error {
	return wc.socket.WriteMessage(websocket.BinaryMessage, data)
}

func (wc *WebsocketConnection) Ping() error {
	return wc.socket.WriteMessage(websocket.PingMessage, nil)
}

func (wc *WebsocketConnection) Read() ([]byte, error) {
	_, p, err := wc.socket.ReadMessage()
	return p, err
}

func (wc *WebsocketConnection) Close(errCode string) {
	wc.socket.SetWriteDeadline(time.Now().Add(time.Second * 20))
	wc.socket.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, errCode))
	wc.socket.Close()
}

func NewWebsocketConnection(conn *websocket.Conn) WebsocketConnection {
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(time.Minute))
		return nil
	})
	return WebsocketConnection{conn}
}
