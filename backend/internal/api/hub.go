package api

import (
	"database/sql"
	"drivehive-backend/internal/database"
	"drivehive-backend/internal/models"
	"encoding/json"
	"log"
	"time"
)

// HubMessage wraps a message with its source client
type HubMessage struct {
	Client  *Client
	Message models.Message
}

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Database connection
	DB *sql.DB

	// Inbound messages from the clients.
	Broadcast chan HubMessage

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client
}

func NewHub(db *sql.DB) *Hub {
	return &Hub{
		Broadcast:  make(chan HubMessage),
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
				// Notify others that the user has left
				if client.RoomID != "" {
					leaveMsg := models.Message{
						RoomID:    client.RoomID,
						Type:      "presence",
						Sender:    client.Username,
						Content:   "offline",
						Timestamp: time.Now(),
					}
					messageData, _ := json.Marshal(leaveMsg)
					h.broadcastToRoom(client.RoomID, messageData)
				}

				delete(h.clients, client)
				close(client.Send)
			}
		case hbm := <-h.Broadcast:
			client := hbm.Client
			msg := hbm.Message

			// Always enforce the sender from the authenticated session
			msg.Sender = client.Username
			msg.Timestamp = time.Now()

			// Special case: Initial join loads history
			if msg.Type == "join" {
				// Verify membership before allowing join/history access
				authorized, err := database.IsUserInChannel(h.DB, client.UserID, msg.RoomID)
				if err != nil || !authorized {
					log.Printf("Unauthorized join attempt: user %s to room %s", client.Username, msg.RoomID)
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

			// If the frontend didn't specify a room, use the client's current room
			if msg.RoomID == "" {
				msg.RoomID = client.RoomID
			}

			isVolatile := msg.Type == "typing" || msg.Type == "presence" || msg.Type == "join"

			if !isVolatile {
				if err := database.SaveMessage(h.DB, msg); err != nil {
					log.Printf("DB Save Error: %v", err)
				}
			}

			// Prepare data once for all clients
			messageData, _ := json.Marshal(msg)

			// Defensive: don't broadcast if no room is assigned
			if msg.RoomID == "" {
				continue
			}

			h.broadcastToRoom(msg.RoomID, messageData)
		}
	}
}

// Helper to broadcast to a specific room
func (h *Hub) broadcastToRoom(roomID string, data []byte) {
	for client := range h.clients {
		if client.RoomID == roomID {
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(h.clients, client)
			}
		}
	}
}
