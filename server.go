package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Message structure to hold message data
type Message struct {
	Content string `json:"content"`
}

// Global variables
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]bool)
var mu sync.Mutex // To handle concurrent access to the clients map
var messages []Message // Slice to hold chat messages

// Handle WebSocket connections
func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	mu.Lock()
	clients[conn] = true
	mu.Unlock()

	for {
		// Read message from the WebSocket connection
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			mu.Lock()
			delete(clients, conn)
			mu.Unlock()
			break
		}

		// Broadcast the message to all connected clients
		mu.Lock()
		newMessage := Message{Content: string(msg)}
		messages = append(messages, newMessage) // Store the message
		for client := range clients {
			if err := client.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Println(err)
				client.Close()
				delete(clients, client)
			}
		}
		mu.Unlock()
	}
}

// Get existing messages
func getMessages(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/get-messages", getMessages) // New endpoint to fetch messages
	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/chat.html")
	})

	log.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
