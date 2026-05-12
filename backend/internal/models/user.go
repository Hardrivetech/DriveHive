package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	AvatarURL    string    `json:"avatar_url"`
	Bio          string    `json:"bio"`
	Password     string    `json:"password,omitempty"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
