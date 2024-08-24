package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"go-websocket-server/api"   // Import the api package
	"go-websocket-server/utils" // Import utils for DB initialization
	"log"
	"net/http"
	"time"
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

	// Goroutine to handle transcript results separately
	doneChan := make(chan string)
	go api.SendToClient(transcriptChan, conn, doneChan)

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		if messageType == websocket.TextMessage {
			var message Message
			if err := json.Unmarshal(p, &message); err != nil {
				log.Println("Error unmarshaling message:", err)
				continue
			}
			if message.Type == "audioEnd" {
				log.Println("Received audioEnd message, waiting for final transcripts")
				time.Sleep(2 * time.Second)
				close(transcriptChan)
				// Wait for all transcripts to be processed and returned
				fullTranscript := <-doneChan
				message.Text = fullTranscript
				log.Printf("Full transcript is %s", fullTranscript)
				// Properly close the Deepgram WebSocket and the client WebSocket
				go func() {
					time.Sleep(2 * time.Second) // Optional sleep for final processing
					log.Println("Closing Deepgram WebSocket connection")
					if err := deepgramConn.Close(); err != nil {
						log.Println("Error closing Deepgram WebSocket:", err)
					}

					log.Println("Closing client WebSocket connection")
					if err := conn.Close(); err != nil {
						log.Println("Error closing client WebSocket:", err)
					}
				}()
			}
			log.Printf("Sending text to llama: %s", message.Text)

			if message.ConversationID == "" {
				log.Println("Error: ConversationID is empty")
				continue
			}

			resultsChan := make(chan string)
			go api.StreamResponses(message.ConversationID, message.Text, resultsChan)

			log.Println("Streaming response back to client...")
			for result := range resultsChan {
				if err := conn.WriteMessage(websocket.TextMessage, []byte(result)); err != nil {
					log.Println(err)
					return
				}
			}
		} else if messageType == websocket.BinaryMessage {
			log.Printf("Received %d bytes of audio data", len(p))

			// Send the audio chunk to Deepgram directly
			err := deepgramConn.WriteMessage(websocket.BinaryMessage, p)
			if err != nil {
				log.Println("Error sending chunk to Deepgram:", err)
			} else {
				log.Println("Successfully sent chunk to Deepgram")
			}
		}
	}
}
