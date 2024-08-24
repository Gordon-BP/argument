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
	"strings"
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

func StreamResponses(conversationId string, userMessage string,
	resultsChan chan<- string) {
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
		close(resultsChan)
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
			botResponseBuffer.WriteString(data.Choices[0].Delta.Content) // Accumulate the bot's response
			resultsChan <- string(data.Choices[0].Delta.Content)
		}
	}
	// Save the bot's response to the database
	botResponse := botResponseBuffer.String()
	if botResponse != "" {
		err := utils.SaveMessage(conversationId, nextIndex, "assistant", "assistant", botResponse)
		if err != nil {
			log.Printf("Failed to save bot response: %v", err)
		} else {
			log.Printf("Bot response saved successfully. Length: %d",
				len(botResponse))
		}
	} else {
		log.Println("Warning: Bot response was empty")
	}
	// Close the results channel when done to signal completion
	close(resultsChan)
}
