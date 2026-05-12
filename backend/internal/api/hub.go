package api

import (
	"database/sql"
	"drivehive-backend/internal/database"
	"drivehive-backend/internal/models"
	"encoding/json"
	"log"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Database connection
	DB *sql.DB

	// Inbound messages from the clients.
	Broadcast chan models.Message

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client
}

func NewHub(db *sql.DB) *Hub {
	return &Hub{
		Broadcast:  make(chan models.Message),
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
		case msg := <-h.Broadcast:
			// Special case: Initial join loads history
			if msg.Type == "join" {
				for client := range h.clients {
					if client.Username == msg.Sender {
						// Verify membership before allowing join/history access
						authorized, err := database.IsUserInChannel(h.DB, client.UserID, msg.RoomID)
						if err != nil || !authorized {
							log.Printf("Unauthorized join attempt: user %s to room %s", client.Username, msg.RoomID)
							// Optionally send a system message back to the client about the error
							continue
						}

						client.RoomID = msg.RoomID
						history, err := database.GetRecentMessages(h.DB, msg.RoomID, time.Time{}, 50)
						if err != nil {
							log.Printf("Error fetching history for room %s: %v", msg.RoomID, err)
							continue
						}
						for _, oldMsg := range history {
							data, _ := json.Marshal(oldMsg)
							client.Send <- data
						}
					}
				}
				// Don't 'continue' here anymore; we want to broadcast the join event
				// to other people in the room so their UI can show the user as "active"
			}

			// Typing indicators and presence updates are "volatile"
			// We broadcast them to the room but DO NOT save them to the database.
			isVolatile := msg.Type == "typing" || msg.Type == "presence" || msg.Type == "join"

			// Normal chat: Save to DB
			if !isVolatile {
				if err := database.SaveMessage(h.DB, msg); err != nil {
					log.Printf("DB Save Error: %v", err)
				}
			}

			// Prepare data once for all clients
			messageData, _ := json.Marshal(msg)

			// Broadcast only to clients in the same room
			for client := range h.clients {
				if client.RoomID == msg.RoomID {
					select {
					case client.Send <- messageData:
					default:
						close(client.Send)
						delete(h.clients, client)
					}
				}
			}
		}
	}
}
