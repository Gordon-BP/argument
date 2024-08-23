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

		if response.Type == "Results" &&
			len(response.Channel.Alternatives) > 0 {
			for _, alternative := range response.Channel.Alternatives {
				if alternative.Transcript != "" {
					outChan <- alternative.Transcript // Send the transcript through the channel
					log.Println("Transcript sent to channel:", alternative.Transcript)
				}
			}
		}
	}
}

// StreamToDeepgram sends audio data to an existing Deepgram WebSocket connection
func StreamToDeepgram(conn *websocket.Conn, audioData []byte) error {
	log.Printf("Ready to send %d bytes of data to Deepgram", len(audioData))
	// Send audio data
	if err := conn.WriteMessage(websocket.BinaryMessage, audioData); err != nil {
		return fmt.Errorf("failed to send audio data to Deepgram: %w",
			err)
	}
	log.Println("Audio data sent, waiting for responses...")
	return nil
}
