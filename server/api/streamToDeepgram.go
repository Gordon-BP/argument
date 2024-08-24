package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"go-websocket-server/utils"
	"log"
	"net/http"
	"os"
)

type Response struct {
	Type    string `json:"type"`
	Channel struct {
		Alternatives []struct {
			Transcript string `json:"transcript"`
		} `json:"alternatives"`
	} `json:"channel"`
}

// NewDeepgramConnection initializes and returns a WebSocket connection to Deepgram
func NewDeepgramConnection(outChan chan<- string) (*websocket.Conn, error) {
	apiErr := utils.LoadEnv(".env")
	if apiErr != nil {
		return nil, fmt.Errorf("error loading .env file: %w", apiErr)
	}

	apiKey := os.Getenv("DEEPGRAM_API_KEY")
	headers := http.Header{}
	headers.Set("Authorization", "Token "+apiKey)

	// Establish a WebSocket connection to the Deepgram API
	const wsURL = "wss://api.deepgram.com/v1/listen"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		log.Fatal("Dial:", err)
		return nil, err
	}
	log.Println("Connected to Deepgram")

	go listenForResponses(conn, outChan)
	return conn, nil
}

// listenForResponses listens for responses from Deepgram
func listenForResponses(conn *websocket.Conn, outChan chan<- string) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			return
		}

		log.Printf("Received message from Deepgram: %s", message) //Log raw messages

		var response Response
		if err := json.Unmarshal(message, &response); err != nil {
			log.Printf("Error decoding JSON message: %v", err)
			continue
		}

		if response.Type == "Results" && len(response.Channel.Alternatives) > 0 {
			for _, alternative := range response.Channel.Alternatives {
				if alternative.Transcript != "" {
					log.Println("Transcript sent to channel:", alternative.Transcript)
					outChan <- alternative.Transcript // Send the transcript through the channel
				}
			}
		}
	}
}

// Send transcript output back to the client in the right data shape
func SendToClient(transcriptChan chan string, conn *websocket.Conn, doneChan chan string) {
	var fullTranscript string // Accumulate the transcript

	for result := range transcriptChan {
		log.Println("Transcript:", result)
		fullTranscript += result

		// Create the JSON structure
		response := utils.MessageObj{
			Content: result,
			Role:    "user",
			Name:    "user",
		}

		// Marshal the struct to JSON
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			log.Println("Error marshaling JSON:", err)
			continue
		}

		// Send the JSON response over the WebSocket
		if err := conn.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
			log.Println("Error sending transcript:", err)
			break // Stop processing if there's an error sending the message
		}
	}

	// After the loop ends, send the full transcript to the doneChan
	log.Println("Sending final transcript to doneChan")
	doneChan <- fullTranscript
	log.Println("Final transcript sent, closing doneChan")
	close(doneChan) // Close the doneChan to signal completion
}
