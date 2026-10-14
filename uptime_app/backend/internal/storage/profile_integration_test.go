package storage

import (
	"context"
	"testing"
	"uptime_app/backend/internal/auth"
)

func TestPostgresProfile(t *testing.T) {
	db := testDB(t)
	service := serviceFor(t, db)
	ctx := context.Background()
	if err := service.Register(ctx, "profile@example.com", password); err != nil {
		t.Fatal(err)
	}
	_, token := lastLetter(t, db, service)
	if err := service.Verify(ctx, token); err != nil {
		t.Fatal(err)
	}
	session, err := service.Login(ctx, "profile@example.com", password)
	if err != nil {
		t.Fatal(err)
	}
	birthday := "1990-05-12"
	user, err := service.UpdateProfile(ctx, session.AccessToken, "  Иванов Иван Иванович  ", &birthday)
	if err != nil || user.FullName != "Иванов Иван Иванович" || user.DateOfBirth == nil || user.DateOfBirth.Format("2006-01-02") != birthday {
		t.Fatalf("%+v %v", user, err)
	}
	current, err := service.Me(ctx, session.AccessToken)
	if err != nil || current.FullName != user.FullName || current.DateOfBirth.Format("2006-01-02") != birthday {
		t.Fatal("profile not persisted", err)
	}
	future := "2999-01-01"
	_, err = service.UpdateProfile(ctx, session.AccessToken, "changed", &future)
	requireError(t, err, auth.ErrInvalid)
	current, _ = service.Me(ctx, session.AccessToken)
	if current.FullName != user.FullName {
		t.Fatal("invalid update changed user")
	}
	_, err = service.UpdateProfile(ctx, session.AccessToken, "   ", nil)
	requireError(t, err, auth.ErrInvalid)
	current, err = service.Me(ctx, session.AccessToken)
	if err != nil || current.FullName != user.FullName || current.DateOfBirth == nil {
		t.Fatal("whitespace update changed profile", err)
	}
	cleared, err := service.UpdateProfile(ctx, session.AccessToken, user.FullName, nil)
	if err != nil || cleared.FullName != user.FullName || cleared.DateOfBirth != nil {
		t.Fatal("clear failed", err)
	}
	if err := service.Logout(ctx, session.RefreshToken); err != nil {
		t.Fatal(err)
	}
	_, err = service.UpdateProfile(ctx, session.AccessToken, "revoked", nil)
	requireError(t, err, auth.ErrUnauthorized)
}
