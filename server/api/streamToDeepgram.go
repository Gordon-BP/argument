package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"go-websocket-server/utils"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
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
func NewDeepgramConnection(outChan chan<- string, stopChan <-chan bool) (*websocket.Conn, error) {
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

	go listenForResponses(conn, outChan, stopChan)
	return conn, nil
}

// listenForResponses listens for responses from Deepgram
func listenForResponses(conn *websocket.Conn, outChan chan<- string, stopChan <-chan bool) {
	for {
		// Check if it is time to stop
		select {
		case <-stopChan:
			fmt.Printf("Stopping deepgram websocket listener")
			return
		default:
			// Otherwise, listen for incoming responses
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Error listening for responses:", err)
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
}

// Send transcript output back to the client in the right data shape
// Receives a stream of text from input channel. The text is then put into the right
// shape to send to the client as a user message.
// Then the raw text is collected into a single full transcript
// and this transcript is pushed into the output channel
func SendTranscriptToClient(inputChannel chan string, outputChannel chan string, writeChan chan<- utils.WebSocketPacket) {
	var fullTranscript string // Accumulate the transcript

	for result := range inputChannel {
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

		// Send the JSON to be written to the websocket
		writeChan <- utils.WebSocketPacket{
			Type: utils.TextMessage,
			Data: jsonResponse,
		}
	}

	// After the loop ends, send the full transcript to the doneChan
	log.Println("Sending final transcript to output")
	outputChannel <- string(fullTranscript)
	log.Println("Final transcript sent, closing output")
	close(outputChannel) // Close the doneChan to signal completion
}
func SendToDeepgramTTS(text string, outChan chan<- []byte) {
	url := "https://api.deepgram.com/v1/speak?model=aura-helios-en"

	apiKey := os.Getenv("DEEPGRAM_API_KEY")
	req, err := http.NewRequest("POST", url, strings.NewReader(text))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Token "+apiKey)
	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	log.Println("Sending text to deepgram TTS:", text)
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Deepgram API returned status:", resp.Status)
		return
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	log.Printf("Successfully received %d bytes from deepgram", len(audioData))
	outChan <- audioData

}
func SendAudioToClient(inputChannel chan []byte, writeChan chan<- utils.WebSocketPacket) {
	for audio := range inputChannel {
		log.Printf("Sending %d bytes of audio to client", len(audio))
		writeChan <- utils.WebSocketPacket{
			Type: utils.BinaryMessage,
			Data: audio,
		}
	}
}
