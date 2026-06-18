package auth

import "time"

type User struct {
	ID           int
	Username     string
	PasswordHash string
	IsAdmin      bool
	DisplayName  *string
	CreatedAt    time.Time
}
