package wspool

import (
	"github.com/gorilla/websocket"
	"log"
)

type Pool struct {
	// Registered clients.
	clients map[*websocket.Conn]bool

	// Inbound messages from the clients.
	Broadcast chan bool

	// Register requests from the clients.
	Register chan *websocket.Conn

	// Unregister requests from clients.
	Unregister chan *websocket.Conn
}

func NewPool() *Pool {
	log.Println("client pool created")

	return &Pool{
		Broadcast: make(chan bool),
		Register:  make(chan *websocket.Conn),
		//Unregister: make(chan *websocket.Conn),
		clients: make(map[*websocket.Conn]bool),
	}
}

func (h *Pool) Run() {
	log.Println("client pool started")
	for {
		select {
		case client := <-h.Register:
			h.clients[client] = true
			log.Println("client registered")
		//case client := <-h.Unregister:
		//	if _, ok := h.clients[client]; ok {
		//		delete(h.clients, client)
		//		log.Println("client unregistered")
		//	}
		case <-h.Broadcast:
			for client := range h.clients {
				if err := client.WriteMessage(websocket.TextMessage, []byte("hey")); err != nil {
					log.Printf("write error: %v", err)
					delete(h.clients, client)
					log.Println("client unregistered")
				}

			}
			log.Printf("broadcast to %d clients", len(h.clients))
		}
	}
}
