package api

import (
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

type MessageObj struct {
	Content *string `json:"content,omitempty"` // Optional field
	Name    *string `json:"name,omitempty"`    // Optional field
	Role    *string `json:"role,omitempty"`    // Optional field
}

type GroqPostData struct {
	Messages []MessageObj `json:"messages"` // Change to a slice directly
	Model    string       `json:"model"`    // Make field exported with JSON tag
	Stream   bool         `json:"stream"`   // Make field exported with JSON tag
}
type Choice struct {
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
}
type Data struct {
	Choices []Choice `json:"choices"`
}

func StreamResponses(messages []MessageObj, resultsChan chan<- string) {
	url := "https://api.groq.com/openai/v1/chat/completions" // Replace with your endpoint

	// Create the GroqPostData object with messages directly as a slice
	groqPostData := GroqPostData{
		Messages: messages, // Set messages directly as a slice
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
		log.Fatal("Error loading .env file:", err)
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

	// Initialize a string buffer to collect content
	var contentBuffer strings.Builder

	// Read the response body incrementally
	reader := io.Reader(resp.Body)

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(reader)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}
	rawResponse := buf.String()

	for _, line := range strings.Split(rawResponse, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue // Skip lines that don’t start with "data: "
		}

		line = strings.TrimSpace(line[6:]) // Remove the "data: " prefix and trim whitespace
		if line == "" {
			continue // Skip empty lines
		}

		var data Data
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			log.Printf("Failed to decode JSON: %v", err)
			continue
		}

		// Ensure we only send JSON for the first choice available
		if len(data.Choices) > 0 {
			messageResponse := map[string]string{
				"text": data.Choices[0].Delta.Content,
			}

			// Marshal the response to JSON and send it back through the channel
			responseJson, err := json.Marshal(messageResponse)
			if err != nil {
				log.Printf("Failed to marshal JSON response: %v", err)
				continue
			}

			resultsChan <- string(responseJson) // Send the JSON string back without additional parsing
		}
	}
	// Close the results channel when done to signal completion
	close(resultsChan)

	// Print the accumulated content to stdout
	log.Println(contentBuffer.String())
}
