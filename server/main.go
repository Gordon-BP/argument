package main

import (
	"encoding/json"
	"fmt"
	"go-websocket-server/api"
	"go-websocket-server/utils"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

type Message struct {
	Text           string `json:"text"`
	ConversationID string `json:"conversationId"`
}

func main() {
	utils.InitDB("./conversation.db")

	http.HandleFunc("/ws", handleWebSocket)

	fmt.Println("Server is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		var message Message
		if err := json.Unmarshal(p, &message); err != nil {
			log.Println("Error unmarshaling message:", err)
			continue
		}

		if message.ConversationID == "" {
			log.Println("Error: ConversationID is empty")
			continue
		}

		resultsChan := make(chan string)
		go api.StreamResponses(message.ConversationID, message.Text,
			resultsChan)

		for result := range resultsChan {
			if err := conn.WriteMessage(websocket.TextMessage,
				[]byte(result)); err != nil {
				log.Println(err)
				return
			}
		}
	}
}
