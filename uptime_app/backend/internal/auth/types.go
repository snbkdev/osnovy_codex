package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalid      = errors.New("invalid_input")
	ErrCredentials  = errors.New("invalid_credentials")
	ErrUnverified   = errors.New("email_not_verified")
	ErrUnauthorized = errors.New("unauthorized")
)

type User struct {
	ID, Email, PasswordHash string
	EmailVerified           bool
	FullName                string
	DateOfBirth             *time.Time
}
type Session struct {
	ID, UserID string
	ExpiresAt  time.Time
}
type Letter struct{ To, Subject, Body string }
type LoginResult struct {
	AccessToken, RefreshToken string
	User                      User
}

// Repository owns transaction boundaries; no ORM types escape storage.
type Repository interface {
	Register(context.Context, User, string, time.Time, []byte) error
	Issue(context.Context, string, string, string, time.Time, []byte) error
	Verify(context.Context, string, time.Time) error
	FindUser(context.Context, string) (User, error)
	CreateSession(context.Context, User, Session, string, time.Time) error
	Rotate(context.Context, string, string, time.Time) (User, Session, error)
	Revoke(context.Context, string, time.Time) error
	Reset(context.Context, string, string, time.Time) error
	Current(context.Context, string, string, time.Time) (User, error)
	UpdateProfile(context.Context, string, string, string, *time.Time, time.Time) (User, error)
}
