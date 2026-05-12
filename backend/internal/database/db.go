package database

import (
	"database/sql"
	"drivehive-backend/internal/models"
	"time"

	_ "modernc.org/sqlite"
)

func InitDB(filepath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		return nil, err
	}

	// Create messages table
	query := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		room_id TEXT,
		type TEXT,
		sender TEXT,
		content TEXT,
		timestamp DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_room_id ON messages(room_id);`

	_, err = db.Exec(query)
	return db, err
}

func SaveMessage(db *sql.DB, msg models.Message) error {
	stmt, err := db.Prepare("INSERT INTO messages(room_id, type, sender, content, timestamp) VALUES(?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(msg.RoomID, msg.Type, msg.Sender, msg.Content, time.Now())
	return err
}

func GetRecentMessages(db *sql.DB, roomID string, limit int) ([]models.Message, error) {
	rows, err := db.Query("SELECT room_id, type, sender, content, timestamp FROM messages WHERE room_id = ? ORDER BY id DESC LIMIT ?", roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		var ts time.Time
		if err := rows.Scan(&msg.RoomID, &msg.Type, &msg.Sender, &msg.Content, &ts); err != nil {
			return nil, err
		}
		msg.Timestamp = ts
		// Prepend to maintain chronological order for the UI
		messages = append([]models.Message{msg}, messages...)
	}

	return messages, nil
}
