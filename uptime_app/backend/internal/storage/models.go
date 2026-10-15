package storage

import "time"

type userRow struct {
	ID            string
	Email         string
	PasswordHash  string
	EmailVerified bool
	FullName      string
	AvatarURL     string
	DateOfBirth   *time.Time
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
	RequestHash *string
	NextHash    *string
	Hash        string
	SessionID   string
	UsedAt      *time.Time
	CreatedAt   time.Time
}
type tokenRow struct {
	Hash      string
	UserID    string
	Purpose   string
	ExpiresAt time.Time
}
type mailRow struct {
	Status      string
	ErrorCode   *string
	ID          int64
	Payload     []byte
	Attempts    int
	AvailableAt time.Time
	FinishedAt  *time.Time
}
