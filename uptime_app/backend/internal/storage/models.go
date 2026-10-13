package storage

import "time"

type userRow struct {
	ID            string
	Email         string
	PasswordHash  string
	EmailVerified bool
	CreatedAt     time.Time
}
type sessionRow struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}
type refreshRow struct {
	Hash      string
	SessionID string
	UsedAt    *time.Time
	CreatedAt time.Time
}
type tokenRow struct {
	Hash      string
	UserID    string
	Purpose   string
	ExpiresAt time.Time
}
type mailRow struct {
	ID          int64
	Payload     []byte
	Attempts    int
	AvailableAt time.Time
	FinishedAt  *time.Time
}
