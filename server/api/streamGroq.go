package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go-websocket-server/utils"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type GroqPostData struct {
	Messages []utils.MessageObj `json:"messages"` // Change to a slice directly
	Model    string             `json:"model"`    // Make field exported with JSON tag
	Stream   bool               `json:"stream"`   // Make field exported with JSON tag
}
type Choice struct {
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
}
type Data struct {
	Choices []Choice `json:"choices"`
}

// Main function to interact with the LLM
// Fetches history from sqlite
// then sends the whole packet to the LLM
// and streams its response into outputChan
// The completed response is then sent to deepgram TTS
// which will output to audioChan
func AskLlama(conversationId string, userMessage string, textForClient chan<- string, textForTTS chan<- string) {
	url := "https://api.groq.com/openai/v1/chat/completions"

	// Get conversation history
	history, err := utils.GetConversationHistory(conversationId)
	if err != nil {
		log.Printf("Failed to get conversation history: %v", err)
		history = []utils.MessageObj{}
	}
	// Get the next message index
	nextIndex, err := utils.GetNextMessageIndex(conversationId)
	if err != nil {
		log.Printf("Failed to get next message index: %v", err)
		nextIndex = 0
	}

	// Add the new user message
	userMsg := utils.MessageObj{
		Role:    "user",
		Name:    "user",
		Content: userMessage,
	}
	messages := append(history, userMsg)

	// Save the user message to the database
	err = utils.SaveMessage(conversationId, nextIndex, userMsg.Role, userMsg.Name, userMsg.Content)
	if err != nil {
		log.Printf("Failed to save user message: %v", err)
	}
	nextIndex++

	// Convert db.MessageObj to MessageObj for the API request
	apiMessages := make([]utils.MessageObj, len(messages))
	for i, msg := range messages {
		apiMessages[i] = utils.MessageObj{
			Role:    msg.Role,
			Name:    msg.Name,
			Content: msg.Content,
		}
	}

	groqPostData := GroqPostData{
		Messages: apiMessages,
		Model:    "llama-3.1-8b-instant",
		Stream:   true,
	}

	jsonData, err := json.Marshal(groqPostData)
	fmt.Println(string(jsonData))
	// Create a POST request with JSON body
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	// Set the Content-Type header
	req.Header.Set("Content-Type", "application/json")

	// Add the API key to the header
	apiErr := utils.LoadEnv(".env")
	if apiErr != nil {
		log.Fatal("Error loading .env file:", apiErr)
	}

	apiKey := os.Getenv("GROQ_API_KEY")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Perform the POST request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("API request failed with status code: %d",
			resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Response body: %s", string(body))
		close(textForClient)
		close(textForTTS)
		return
	}
	// Initialize a string buffer to collect the entire bot response
	var botResponseBuffer strings.Builder
	reader := bufio.NewReader(resp.Body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Error reading response: %v", err)
			break
		}

		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		line = strings.TrimPrefix(line, "data: ")
		if line == "[DONE]" {
			break
		}

		var data Data
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			log.Printf("Failed to decode JSON: %v", err)
			continue
		}

		if len(data.Choices) > 0 && data.Choices[0].Delta.Content !=
			"" {
			// Accumulate the bot's response in a buffer
			botResponseBuffer.WriteString(data.Choices[0].Delta.Content)
			// Stream data to text out channels
			textForClient <- string(data.Choices[0].Delta.Content)
			textForTTS <- string(data.Choices[0].Delta.Content)
		}
	}
	// Save the bot's response to the database
	botResponse := botResponseBuffer.String()
	if botResponse != "" {
		err := utils.SaveMessage(conversationId, nextIndex, "assistant", "assistant", botResponse)
		if err != nil {
			log.Printf("Failed to save bot response: %v", err)
		} else {
			log.Println("Bot response saved successfully: ", botResponse)
		}
	} else {
		log.Println("Warning: Bot response was empty")
	}
	// Close the results channel when done to signal completion
	close(textForClient)
	close(textForTTS)
}

// Function that takes a stream of text as an input
// Buffers it, then sends each full sentence to Deepgram TTS
func BufferTextForTTS(inputStream chan string, audioOut chan<- []byte) {
	rateLimitTicker := time.NewTicker(1 * time.Second)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var textBuffer string
	var eosRegex = regexp.MustCompile("([^!?\n]+[.!?\n])")
	for text := range inputStream {
		// Accumulate text in a per-sentence buffer
		textBuffer += text
		// Split sentence buffer by sentence (if applicable)
		sentences := eosRegex.FindAllString(textBuffer, -1)
		if len(sentences) > 1 {
			//merge all but the last partial sentence into one text
			text := strings.Trim(strings.Join(sentences[:len(sentences)-1], " "), " ")
			// set the text buffer to the partial sentence and clear the sentences array
			textBuffer = sentences[len(sentences)-1]
			clear(sentences)
			log.Println("Chunked sentence: ", text)
			wg.Add(1)
			go func(text string) {
				defer wg.Done()
				SendToDeepgramTTS(text, rateLimitTicker, &mu, audioOut)
			}(text)
		}
	}
	// Send whatever is left to TTS
	log.Println("Remaining text: ", textBuffer)
	wg.Add(1)
	go func(text string) {
		defer wg.Done()
		SendToDeepgramTTS(text, rateLimitTicker, &mu, audioOut)
	}(textBuffer)
	wg.Wait() // Wait for all goroutines to finish
	close(audioOut)
	rateLimitTicker.Stop()
}

// Function that takes a stream of text as an input
// Then puts the text in the right shape for a bot message before sending it to the client.
func SendTextToClient(inputChannel chan string, writeChan chan<- utils.WebSocketPacket) {
	fullTranscript := ""
	for text := range inputChannel {
		fullTranscript += text
		// Put text into the right shape to send back to the frontend
		msg := utils.MessageObj{
			Content: text,
			Role:    "bot",
			Name:    "bot",
		}
		msgJSON, err := json.Marshal(msg)
		if err != nil {
			log.Println("Error marshalling JSON:", err)
		}
		writeChan <- utils.WebSocketPacket{
			Type: utils.TextMessage,
			Data: msgJSON,
		}

	}
}
