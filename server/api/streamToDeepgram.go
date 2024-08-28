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
	"time"
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
	log.Println("Connected to Deepgram STT")

	go listenForResponses(conn, outChan, stopChan)
	return conn, nil
}

// listenForResponses listens for responses from Deepgram
func listenForResponses(conn *websocket.Conn, outChan chan<- string, stopChan <-chan bool) {
	// Poll for incoming messages from the WebSocket
	pongTimeout := time.Duration(4000 * time.Millisecond)
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})
	// By deferring stopping everything, we reduce copied code
	defer func() {
		log.Println("Closing listener output channel")
		close(outChan)
		log.Printf("Stopping deepgram websocket listener")
		m := "{\"type\":\"CloseStream\"}"
		err := conn.WriteMessage(websocket.TextMessage, []byte(m))
		if err != nil {
			log.Printf("Error closing deepgram connection: %v", err)
		}
	}()
	for {
		select {
		case <-stopChan:
			log.Println("Stop signal received")
			return
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err,
					websocket.CloseGoingAway,
					websocket.CloseAbnormalClosure) {
					log.Printf("error: %v", err)
					return
				}
				log.Println("Error listening for responses:", err)
				return
			}

			log.Printf("Received message from Deepgram: %s", message) // Log raw messages
			var response Response
			if err := json.Unmarshal(message, &response); err != nil {
				log.Printf("Error decoding JSON message: %v", err)
				continue
			}

			if response.Type == "Results" && len(response.Channel.Alternatives) > 0 {
				for _, alternative := range response.Channel.Alternatives {
					if alternative.Transcript != "" {
						log.Println("Transcript sent to channel:", alternative.Transcript)
						outChan <- alternative.Transcript + " " // Send the transcript through the channel
					}
				}
				//Check for a stop signal after processing the websocket data too
				// This way we don't have to wait for another websocket packet
				// to check for a stop signal.
				select {
				case <-stopChan:
					log.Println("Stop signal received")
					return
				default:
					continue
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
func SendTranscriptToClient(inputChannel chan string, outputChannel chan string, writeChan chan<- utils.WebSocketPacket, stopChan <-chan bool) {
	var fullTranscript string // Accumulate the transcript

	for result := range inputChannel {
		select {
		case <-stopChan:
			log.Printf("Stopping transcript stream to user")
			break

		default:
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
	}

	// After the loop ends, send the full transcript to the doneChan
	log.Println("Sending final transcript to output")
	outputChannel <- fullTranscript
	log.Println("Final transcript sent, closing output")
	close(outputChannel) // Close the doneChan to signal completion
}

// Sends text in one big batch to deepgram API
// TODO: take in a stream of text, chunk it by sentence, and send each sentence to the TTS
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
	close(outChan)

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
