package models

import "time"

type Message struct {
	Type      string    `json:"type"`    // e.g., "chat", "system", "join"
	RoomID    string    `json:"room_id"` // The channel/hive ID
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
