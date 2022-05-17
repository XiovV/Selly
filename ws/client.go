package ws

import (
	"fmt"
	"github.com/gorilla/websocket"
)

func NewWebsocketClient(username string) (*websocket.Conn, error) {
	socketUrl := fmt.Sprintf("ws://localhost:8080/chat?jwt=%s", username)
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
