package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"go-websocket-server/api"
	"net/http"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for demonstration; conside rsecurity in production!
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error during connection upgrade:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Client connected")

	for {
		// Read message from the WebSocket connection
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break
		}

		// Print received message
		fmt.Printf("Received message: %s\n", msg)

		// Convert byte array to string
		messageContent := string(msg) // Convert to string

		// Prepare user message
		role := "user"
		usrMsg := api.MessageObj{
			Content: &messageContent, // Use the string
			Role:    &role,
			Name:    &role,
		}

		// Create an output channel for streaming responses
		resultsChan := make(chan string)

		// Run the streaming function as a goroutine
		go api.StreamResponses([]api.MessageObj{usrMsg}, resultsChan)

		// Continuously read from the results channel
		go func() {
			for response := range resultsChan {
				// Convert response to JSON
				responseBytes := []byte(response)

				// Send JSON response to the WebSocket
				err = conn.WriteMessage(websocket.TextMessage, responseBytes)
				if err != nil {
					fmt.Println("Error writing message:", err)
					break
				}
			}
		}()
	}

	fmt.Println("Client disconnected")
}

func main() {
	http.HandleFunc("/ws", handleWebSocket)

	fmt.Println("Server listening on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
