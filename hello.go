package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan []byte)

func handleMessages() {
	for {
		message := <-broadcast

		for client := range clients {
			if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
				fmt.Println("Error senting message", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func main() {

	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("Error upgrading request:", err)
			return
		}

		clients[conn] = true

		welcomeMessage := []byte("Hello, clients!")
		broadcast <- welcomeMessage

		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("Error reading message: %v\n", err)

				delete(clients, conn)
				conn.Close()
				break
			}

			err = conn.WriteMessage(msgType, msg)
			if err != nil {
				fmt.Println("Error echoing message:", err)
				delete(clients, conn)
				conn.Close()
				break
			}
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	go handleMessages()

	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}
