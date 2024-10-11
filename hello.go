package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// Настройки для обновления HTTP-запроса до WebSocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Разрешаем любые подключения для простоты
		return true
	},
}

// Храним подключения клиентов
var clients = make(map[*websocket.Conn]bool)

// Канал для трансляции сообщений всем клиентам
var broadcast = make(chan []byte)

// Хранение подключенных клиентов  для каждой комнаты
var roomClients = make(map[string]map[*websocket.Conn]bool)

// Функция для обработки сообщений
func handleMessages() {
	for {
		// Получаем следующее сообщение из канала
		message := <-broadcast
		//Отправляем сообщение каждому подключенному клиенту в каждой комнате
		for room, clients := range roomClients {
			for client := range clients {
				if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
					fmt.Println("Ошибка при отправке клиенту сообщение", err)
					client.Close()
					delete(roomClients[room], client)
				}
			}
		}
		// Отправляем сообщение каждому подключённому клиенту
		for client := range clients {
			if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
				fmt.Println("Ошибка при отправке сообщения клиенту:", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func main() {
	//----------------------
	http.HandleFunc("/room/{roomCode}", func(w http.ResponseWriter, r *http.Request) {
		roomCode := r.URL.Query().Get("roomCode")

		// Обновляем HTTP-запрос до WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("Ошибка при обновлении запроса:", err)
			return
		}

		// Добавляем нового клиента в соответствующую комнату
		if _, ok := roomClients[roomCode]; !ok {
			roomClients[roomCode] = make(map[*websocket.Conn]bool)
		}
		roomClients[roomCode][conn] = true

		// Приветственное сообщение для новых подключений
		welcomeMessage := []byte("Добро пожаловать в комнату!")
		broadcast <- welcomeMessage

		// Слушаем сообщения от клиента
		for {
			// Читаем сообщение от клиента
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("Ошибка при чтении сообщения: %v\n", err)
				delete(roomClients[roomCode], conn)
				conn.Close()
				break
			}

			// Логируем полученное сообщение на сервере
			fmt.Printf("Пользователь %s отправил: %s\n", conn.RemoteAddr(), string(msg))

			// Отправляем сообщение через канал для трансляции всем клиентам в комнате
			broadcast <- msg
		}
	})
	//----------------------

	// WebSocket endpoint
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		// Обновляем HTTP-запрос до WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("Ошибка при обновлении запроса:", err)
			return
		}

		// Добавляем нового клиента в карту клиентов
		clients[conn] = true

		// Приветственное сообщение для новых подключений
		welcomeMessage := []byte("Добро пожаловать в чат!")
		broadcast <- welcomeMessage

		// Слушаем сообщения от клиента
		for {
			// Читаем сообщение от клиента
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("Ошибка при чтении сообщения: %v\n", err)
				delete(clients, conn)
				conn.Close()
				break
			}

			// Логируем полученное сообщение на сервере
			fmt.Printf("Пользователь %s отправил: %s\n", conn.RemoteAddr(), string(msg))

			// Отправляем сообщение через канал для трансляции всем клиентам
			broadcast <- msg
		}
	})

	// Служим статический HTML-файл для подключения клиента
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	// Запускаем обработку сообщений в отдельной горутине
	go handleMessages()

	fmt.Println("Сервер запущен!!!")
	http.ListenAndServe(":8080", nil)
}
