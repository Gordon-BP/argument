package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"go-websocket-server/api"   // Import the api package
	"go-websocket-server/utils" // Import utils for DB initialization
	"log"
	"net/http"
)

// Upgrader for handling WebSocket connections.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

// Message structure to define the shape of messages passed between the server and the frontend.
type Message struct {
	Text           string `json:"text"`
	ConversationID string `json:"conversationId"`
	Type           string `json:"type"`
}

func main() {
	// Initialize the SQLite database.
	utils.InitDB("./conversation.db")
	// Handle WebSocket connections at the /ws endpoint.
	http.HandleFunc("/ws", handleWebSocket)

	fmt.Println("Server is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// handleWebSocket handles incoming WebSocket data packets.
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil) // Upgrade to a WebSocket connection.
	if err != nil {
		log.Println(err)
		return
	}
	// Initialize Deepgram WebSocket connection
	transcriptChan := make(chan string) // channel for audio transcript
	deepgramConn, err := api.NewDeepgramConnection(transcriptChan)
	if err != nil {
		log.Fatalf("Failed to connect to Deepgram: %v", err)
	}

	defer conn.Close() // Ensure the connection is closed when done.
	defer deepgramConn.Close()

	for {
		// Infinite loop to keep listening for messages.
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		if messageType == websocket.TextMessage {
			var message Message // Declare a Message struct to hold received data.
			if err := json.Unmarshal(p, &message); err != nil {
				log.Println("Error unmarshaling message:", err)
				continue
			}

			if message.ConversationID == "" {
				log.Println("Error: ConversationID is empty")
				continue
			}

			resultsChan := make(chan string) // Create a channel for results.
			go api.StreamResponses(message.ConversationID, message.Text, resultsChan)

			for result := range resultsChan {
				if err := conn.WriteMessage(websocket.TextMessage, []byte(result)); err != nil {
					log.Println(err)
					return
				}
			}
		} else if messageType == websocket.BinaryMessage {
			log.Printf("Received %d bytes of audio data", len(p))

			// Start sending audio data to Deepgram without waiting for previous processing
			go func(data []byte) {
				if err := api.StreamToDeepgram(deepgramConn, data); err != nil {
					log.Println("Error while streaming to Deepgram:", err)
				}
			}(p) // Pass the received audio data to the goroutine

			// Continuously read messages from transcriptChan
			for result := range transcriptChan {
				log.Println(result)
				if err := conn.WriteMessage(websocket.TextMessage, []byte(result)); err != nil {
					log.Println("Error sending transcript:", err)
					return
				}
			}

		}
		continue
	}
}
