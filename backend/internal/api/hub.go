package api

import (
	"database/sql"
	"drivehive-backend/internal/database"
	"drivehive-backend/internal/models"
	"encoding/json"
	"log"
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Database connection
	DB *sql.DB

	// Inbound messages from the clients.
	Broadcast chan []byte

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client
}

func NewHub(db *sql.DB) *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		DB:         db,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.clients[client] = true
		case client := <-h.Unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
		case message := <-h.Broadcast:
			var msg models.Message
			if err := json.Unmarshal(message, &msg); err == nil {
				// Special case: Initial join loads history
				if msg.Type == "join" {
					// In a real-world scenario, we'd track which client sent this via a session.
					// For now, we find the client whose connection matches the sender ID
					// or simply apply the RoomID to the intended client.
					for client := range h.clients {
						// If we identify the sender (msg.Sender currently holds "self" or IP)
						// we assign them to the room and push history.
						client.RoomID = msg.RoomID
						history, _ := database.GetRecentMessages(h.DB, msg.RoomID, 50)
						for _, oldMsg := range history {
							data, _ := json.Marshal(oldMsg)
							client.Send <- data
						}
					}
					continue
				}

				// Normal chat: Save to DB
				if err := database.SaveMessage(h.DB, msg); err != nil {
					log.Printf("DB Save Error: %v", err)
				}

				// Broadcast only to clients in the same room
				for client := range h.clients {
					if client.RoomID == msg.RoomID {
						select {
						case client.Send <- message:
						default:
							close(client.Send)
							delete(h.clients, client)
						}
					}
				}
			}
		}
	}
}
