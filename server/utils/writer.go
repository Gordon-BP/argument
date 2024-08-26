package utils

import (
	"github.com/gorilla/websocket"
	"log"
)

// MessageType represents the type of data being sent
type MessageType int

const (
	TextMessage MessageType = iota
	BinaryMessage
)

// WebSocketPacket is a struct to hold data to be sent over the WebSocket
type WebSocketPacket struct {
	Type MessageType
	Data []byte
}

// Helper function to coordinate client-bound data that gets written to the websocket
func WriteToWebsocket(writeChan <-chan WebSocketPacket, clientConn *websocket.Conn) {
	for packet := range writeChan {
		switch packet.Type {
		case TextMessage:
			if err := clientConn.WriteMessage(websocket.TextMessage, packet.Data); err != nil {
				log.Println("Error sending text message:", err)
			}
		case BinaryMessage:
			if err := clientConn.WriteMessage(websocket.BinaryMessage, packet.Data); err != nil {
				log.Println("Error sending audio message:", err)
			}
		}
	}

}
