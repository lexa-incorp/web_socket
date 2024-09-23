package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Message represents a chat message.
type Message struct {
	User    string `json:"user"`
	Message string `json:"message"`
}

// ChatManager manages chat connections and broadcasting messages.
type ChatManager struct {
	connections map[*websocket.Conn]bool
	broadcast   chan Message
	mu          sync.Mutex
}

// NewChatManager creates a new ChatManager.
func NewChatManager() *ChatManager {
	return &ChatManager{
		connections: make(map[*websocket.Conn]bool),
		broadcast:   make(chan Message),
	}
}

// Run starts the chat manager.
func (manager *ChatManager) Run() {
	for {
		select {
		case message := <-manager.broadcast:
			manager.mu.Lock()
			for conn := range manager.connections {
				err := conn.WriteJSON(message)
				if err != nil {
					log.Printf("Error broadcasting message: %v", err)
					manager.removeConnection(conn)
				}
			}
			manager.mu.Unlock()
		}
	}
}

// AddConnection adds a new connection to the manager.
func (manager *ChatManager) AddConnection(conn *websocket.Conn) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.connections[conn] = true
}

// RemoveConnection removes a connection from the manager.
func (manager *ChatManager) removeConnection(conn *websocket.Conn) {
	if _, ok := manager.connections[conn]; ok {
		delete(manager.connections, conn)
		conn.Close()
	}
}

// BroadcastMessage broadcasts a message to all connections.
func (manager *ChatManager) BroadcastMessage(message Message) {
	manager.broadcast <- message
}

// handleConnections handles new WebSocket connections.
func handleConnections(manager *ChatManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Error upgrading connection: %v", err)
			return
		}
		defer conn.Close()

		manager.AddConnection(conn)

		for {
			var message Message
			err := conn.ReadJSON(&message)
			if err != nil {
				manager.removeConnection(conn)
				break
			}
			message.User = "User" // Replace with actual user identification
			manager.BroadcastMessage(message)
		}
	}
}

func main() {
	router := gin.Default()
	chatManager := NewChatManager()

	// Start the chat manager in a goroutine
	go chatManager.Run()

	// Define the route for the WebSocket endpoint
	router.GET("/ws", handleConnections(chatManager))

	// Define the route for the ping endpoint
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Start the HTTP server
	log.Fatal(router.Run(":8080"))
}
