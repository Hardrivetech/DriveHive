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

	// Initialize tables
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		avatar_url TEXT DEFAULT '',
		bio TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS hives (
		id TEXT PRIMARY KEY,
		name TEXT,
		owner_id INTEGER,
		FOREIGN KEY(owner_id) REFERENCES users(id)
	);
	CREATE TABLE IF NOT EXISTS channels (
		id TEXT PRIMARY KEY,
		hive_id TEXT,
		name TEXT,
		type TEXT,
		FOREIGN KEY(hive_id) REFERENCES hives(id)
	);
	CREATE TABLE IF NOT EXISTS hive_members (
		hive_id TEXT,
		user_id INTEGER,
		role TEXT DEFAULT 'member',
		PRIMARY KEY (hive_id, user_id),
		FOREIGN KEY(hive_id) REFERENCES hives(id),
		FOREIGN KEY(user_id) REFERENCES users(id)
	);
	CREATE TABLE IF NOT EXISTS invites (
		code TEXT PRIMARY KEY,
		hive_id TEXT,
		creator_id INTEGER,
		expires_at DATETIME,
		FOREIGN KEY(hive_id) REFERENCES hives(id),
		FOREIGN KEY(creator_id) REFERENCES users(id)
	);
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

func CreateHive(db *sql.DB, id, name string, ownerID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	tx.Exec("INSERT INTO hives (id, name, owner_id) VALUES (?, ?, ?)", id, name, ownerID)
	tx.Exec("INSERT INTO hive_members (hive_id, user_id, role) VALUES (?, ?, 'owner')", id, ownerID)
	return tx.Commit()
}

func CreateChannel(db *sql.DB, id, hiveID, name, cType string) error {
	_, err := db.Exec("INSERT INTO channels (id, hive_id, name, type) VALUES (?, ?, ?, ?)", id, hiveID, name, cType)
	return err
}

func AddUserToHive(db *sql.DB, hiveID string, userID int) error {
	_, err := db.Exec("INSERT OR IGNORE INTO hive_members (hive_id, user_id) VALUES (?, ?)", hiveID, userID)
	return err
}

func CreateInvite(db *sql.DB, code, hiveID string, creatorID int) error {
	expiresAt := time.Now().Add(24 * 7 * time.Hour) // 1 week expiry
	_, err := db.Exec("INSERT INTO invites (code, hive_id, creator_id, expires_at) VALUES (?, ?, ?, ?)", code, hiveID, creatorID, expiresAt)
	return err
}

func GetHiveIDByInvite(db *sql.DB, code string) (string, error) {
	var hiveID string
	err := db.QueryRow("SELECT hive_id FROM invites WHERE code = ? AND expires_at > ?", code, time.Now()).Scan(&hiveID)
	return hiveID, err
}

func GetUserRole(db *sql.DB, userID int, hiveID string) (string, error) {
	var role string
	err := db.QueryRow("SELECT role FROM hive_members WHERE user_id = ? AND hive_id = ?", userID, hiveID).Scan(&role)
	return role, err
}

func GetUser(db *sql.DB, userID int) (*models.User, error) {
	u := &models.User{}
	err := db.QueryRow("SELECT id, username, avatar_url, bio, created_at FROM users WHERE id = ?", userID).
		Scan(&u.ID, &u.Username, &u.AvatarURL, &u.Bio, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func UpdateUserProfile(db *sql.DB, userID int, avatarURL, bio string) error {
	_, err := db.Exec("UPDATE users SET avatar_url = ?, bio = ? WHERE id = ?", avatarURL, bio, userID)
	return err
}

type HiveModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ChannelModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func GetUserHives(db *sql.DB, userID int) ([]HiveModel, error) {
	rows, err := db.Query(`
		SELECT h.id, h.name 
		FROM hives h
		JOIN hive_members hm ON h.id = hm.hive_id
		WHERE hm.user_id = ?`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hives []HiveModel
	for rows.Next() {
		var h HiveModel
		rows.Scan(&h.ID, &h.Name)
		hives = append(hives, h)
	}
	return hives, nil
}

func IsUserInChannel(db *sql.DB, userID int, channelID string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM hive_members hm
			JOIN channels c ON hm.hive_id = c.hive_id
			WHERE hm.user_id = ? AND c.id = ?
		)`
	err := db.QueryRow(query, userID, channelID).Scan(&exists)
	return exists, err
}

func GetHiveMembers(db *sql.DB, hiveID string) ([]models.User, error) {
	rows, err := db.Query(`
		SELECT u.id, u.username 
		FROM users u
		JOIN hive_members hm ON u.id = hm.user_id
		WHERE hm.hive_id = ?`,
		hiveID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		rows.Scan(&u.ID, &u.Username)
		users = append(users, u)
	}
	return users, nil
}

func IsUserInHive(db *sql.DB, userID int, hiveID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM hive_members WHERE user_id = ? AND hive_id = ?)`
	err := db.QueryRow(query, userID, hiveID).Scan(&exists)
	return exists, err
}

func GetHiveChannels(db *sql.DB, hiveID string) ([]ChannelModel, error) {
	rows, err := db.Query("SELECT id, name, type FROM channels WHERE hive_id = ?", hiveID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []ChannelModel
	for rows.Next() {
		var c ChannelModel
		rows.Scan(&c.ID, &c.Name, &c.Type)
		channels = append(channels, c)
	}
	return channels, nil
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

func GetRecentMessages(db *sql.DB, roomID string, before time.Time, limit int) ([]models.Message, error) {
	var rows *sql.Rows
	var err error

	// If before is zero, we get the absolute latest messages.
	// Otherwise, we get messages older than the provided timestamp.
	if before.IsZero() {
		rows, err = db.Query("SELECT room_id, type, sender, content, timestamp FROM messages WHERE room_id = ? ORDER BY timestamp DESC LIMIT ?", roomID, limit)
	} else {
		rows, err = db.Query("SELECT room_id, type, sender, content, timestamp FROM messages WHERE room_id = ? AND timestamp < ? ORDER BY timestamp DESC LIMIT ?", roomID, before, limit)
	}

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
		messages = append(messages, msg)
	}

	// Reverse the slice so messages are returned in chronological order (ASC) for the UI
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
