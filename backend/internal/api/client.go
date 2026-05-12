package api

import (
	"drivehive-backend/internal/auth"
	"drivehive-backend/internal/models"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, validate against tauri://localhost or similar
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5173" || origin == "tauri://localhost" {
			return true
		}
		// Allow empty origin for local development tools/testing
		if origin == "" {
			return true
		}
		return false
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Hub      *Hub
	Conn     *websocket.Conn
	Username string
	UserID   int
	RoomID   string // Track which hive the user is currently in
	Send     chan []byte
}

// ReadPump pumps messages from the websocket connection to the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		var msg models.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("error unmarshaling message: %v", err)
			continue
		}

		c.Hub.Broadcast <- msg
	}
}

// WritePump pumps messages from the hub to the websocket connection.
func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()
	for message := range c.Send {
		w, err := c.Conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}
		w.Write(message)

		if err := w.Close(); err != nil {
			return
		}
	}
	c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
}

// ServeWs handles websocket requests from the peer.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Extract and verify token from query parameter
	token := r.URL.Query().Get("token")
	claims, err := auth.VerifyToken(token)
	if err != nil {
		log.Printf("Unauthorized WS connection attempt: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{Hub: hub, Conn: conn, Username: claims.Username, UserID: claims.UserID, Send: make(chan []byte, 256)}
	client.Hub.Register <- client
	go client.WritePump()
	go client.ReadPump()
}
