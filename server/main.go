package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"go-websocket-server/api"
	"go-websocket-server/utils"
	"log"
	"net/http"
)

// These are the packages that I import for this function:
// fmt, log, net/http are all part of the standard library
// gorilla is a 3rd party package for webhooks
// go-websocket-server is this project, and the two packages are ones I wrote for dealing with the Groq API and for dealing with the sqlite database

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
	// From the docs:
	// "The Upgrader upgrades the HTTP server connection to the WebSocket protocol."
	// The buffer sizes speficy how big the buffers can be, in bytes. 1024 means 1 kilobyte, so tiny!
}

type Message struct {
	// This is the shape of messages that we pass between the server and the frontend
	// Defined as a struct so that the data is always consistent
	Text           string `json:"text"`
	ConversationID string `json:"conversationId"`
}

func main() {
	// This is the function that runs when we call go run main.go
	utils.InitDB("./conversation.db")
	// We initialize the sqlite database

	http.HandleFunc("/ws", handleWebSocket)
	// We tell the server to use the handleWebSocket function
	// whenever there is a message coming in via websocket

	fmt.Println("Server is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
	// These ensure the logs are constantly printed to stdout
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// This is out main function to handle websocket data packets
	conn, err := upgrader.Upgrade(w, r, nil)
	// We need to get our connector from the http connection
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	// This says that we close the connection when the function is done
	// We could also put conn.close() at the end of the function
	// But it is better to put it here where it is with the rest of the connection code

	for {
		// This is how you do infinite loops in Go
		_, p, err := conn.ReadMessage()
		// Read the packet. The first var is the packet index, then the packet itself
		// This action is blocking, which means this loop essentially waits for messages
		if err != nil {
			log.Println(err)
			return
		}

		var message Message
		// Declare an empty message object to mutate with the data from the packet
		if err := json.Unmarshal(p, &message); err != nil {
			// Unmarshal takes JSON data and makes a struct out of it
			// because we passed in out Message struct, it will try to fill that
			// with the contents of the packet
			log.Println("Error unmarshaling message:", err)
			continue
		}

		if message.ConversationID == "" {
			// Here we make sure that there is a conversation id
			log.Println("Error: ConversationID is empty")
			continue
		}

		resultsChan := make(chan string)
		// Channels are like buffers, they're a place for data to be streamed into
		// Because Go is super typed, the channel has the string type and can only
		// accept string data. Pretty cool!
		go api.StreamResponses(message.ConversationID, message.Text,
			resultsChan)
		// This is the best part! We handle calling the API and streaming its results
		// in a goroutine, which unlocks crazy concurrency and speed

		for result := range resultsChan {
			// Here we iterate through the results channel. This channel gets filled by
			// the results of the API call go routine
			if err := conn.WriteMessage(websocket.TextMessage,
				// Here we write string data straight to the websocket
				// as long as there are no errors of course
				[]byte(result)); err != nil {
				log.Println(err)
				return
			}
		}
	}
}
