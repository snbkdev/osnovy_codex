package storage

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"testing"
	"uptime_app/backend/internal/auth"
	avatarstore "uptime_app/backend/internal/avatar"
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
	identity, err := service.Authenticate(ctx, session.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	birthday := "1990-05-12"
	user, err := service.UpdateProfile(ctx, identity, "  Иванов Иван Иванович  ", &birthday)
	if err != nil || user.FullName != "Иванов Иван Иванович" || user.DateOfBirth == nil || user.DateOfBirth.Format("2006-01-02") != birthday {
		t.Fatalf("%+v %v", user, err)
	}
	current, err := service.Me(ctx, session.AccessToken)
	if err != nil || current.FullName != user.FullName || current.DateOfBirth.Format("2006-01-02") != birthday {
		t.Fatal("profile not persisted", err)
	}
	future := "2999-01-01"
	_, err = service.UpdateProfile(ctx, identity, "changed", &future)
	requireError(t, err, auth.ErrInvalid)
	current, _ = service.Me(ctx, session.AccessToken)
	if current.FullName != user.FullName {
		t.Fatal("invalid update changed user")
	}
	_, err = service.UpdateProfile(ctx, identity, "   ", nil)
	requireError(t, err, auth.ErrInvalid)
	current, err = service.Me(ctx, session.AccessToken)
	if err != nil || current.FullName != user.FullName || current.DateOfBirth == nil {
		t.Fatal("whitespace update changed profile", err)
	}
	cleared, err := service.UpdateProfile(ctx, identity, user.FullName, nil)
	if err != nil || cleared.FullName != user.FullName || cleared.DateOfBirth != nil {
		t.Fatal("clear failed", err)
	}
	var avatar bytes.Buffer
	if err := png.Encode(&avatar, image.NewRGBA(image.Rect(0, 0, 32, 32))); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	service.Avatars = avatarstore.LocalStore{Dir: dir}
	uploaded, err := service.SaveAvatar(ctx, identity, bytes.NewReader(avatar.Bytes()))
	if err != nil || uploaded.AvatarURL == "" || uploaded.FullName != user.FullName {
		t.Fatal("upload failed", uploaded, err)
	}
	current, err = service.Me(ctx, session.AccessToken)
	if err != nil || current.AvatarURL != uploaded.AvatarURL {
		t.Fatal("avatar not persisted", current, err)
	}
	if _, err := service.UpdateProfile(ctx, identity, "Новое имя", nil); err != nil {
		t.Fatal(err)
	}
	current, err = service.Me(ctx, session.AccessToken)
	if err != nil || current.AvatarURL != uploaded.AvatarURL {
		t.Fatal("profile update lost avatar", current, err)
	}
	// A page reload restores the user through refresh before requesting /me.
	refreshed, err := service.Refresh(ctx, session.RefreshToken)
	if err != nil || refreshed.User.AvatarURL != uploaded.AvatarURL {
		t.Fatal("session refresh lost avatar", refreshed.User, err)
	}
	current, err = service.Me(ctx, refreshed.AccessToken)
	if err != nil || current.AvatarURL != uploaded.AvatarURL {
		t.Fatal("restored session lost avatar", current, err)
	}
	session = refreshed
	if err := service.Logout(ctx, session.RefreshToken); err != nil {
		t.Fatal(err)
	}
	_, err = service.UpdateProfile(ctx, identity, "revoked", nil)
	requireError(t, err, auth.ErrUnauthorized)
	_, err = service.SaveAvatar(ctx, identity, bytes.NewReader(avatar.Bytes()))
	requireError(t, err, auth.ErrUnauthorized)
}
