package ws

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
)

func NewWebsocketClient(username string) *websocket.Conn {
	socketUrl := fmt.Sprintf("ws://localhost:8080/chat?username=%s", username)
	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		log.Fatal("Error connecting to Websocket Server:", err)
	}

	return conn
}
