package auth

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type avatarRepo struct {
	Repository
	err   error
	calls int
}

func (r *avatarRepo) UpdateAvatar(_ context.Context, uid, sid, url string, _ time.Time) (User, error) {
	r.calls++
	if uid != "user" || sid != "session" {
		panic("wrong ownership")
	}
	return User{ID: uid, AvatarURL: url}, r.err
}
func TestSaveAvatar(t *testing.T) {
	signer := Signer{Key: bytes.Repeat([]byte{1}, 32), Issuer: "test", Audience: "web"}
	token, err := signer.Issue("user", "session", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"jpeg", "png"} {
		t.Run(format, func(t *testing.T) {
			repo := &avatarRepo{}
			service := &Service{Repo: repo, Signer: signer}
			dir := t.TempDir()
			var input bytes.Buffer
			img := image.NewRGBA(image.Rect(0, 0, 80, 40))
			if format == "jpeg" {
				err = jpeg.Encode(&input, img, nil)
			} else {
				err = png.Encode(&input, img)
			}
			if err != nil {
				t.Fatal(err)
			}
			user, err := service.SaveAvatar(context.Background(), token, dir, bytes.NewReader(input.Bytes()))
			if err != nil || repo.calls != 1 || !strings.HasPrefix(user.AvatarURL, "/api/v1/avatars/") {
				t.Fatal(user, err)
			}
			data, err := os.ReadFile(filepath.Join(dir, filepath.Base(user.AvatarURL)))
			if err != nil {
				t.Fatal(err)
			}
			cfg, kind, err := image.DecodeConfig(bytes.NewReader(data))
			if err != nil || kind != "png" || cfg.Width != 256 || cfg.Height != 256 {
				t.Fatal(cfg, kind, err)
			}
			repo.err = ErrUnauthorized
			_, err = service.SaveAvatar(context.Background(), token, dir, bytes.NewReader(input.Bytes()))
			entries, _ := os.ReadDir(dir)
			if !errors.Is(err, ErrUnauthorized) || len(entries) != 1 {
				t.Fatal("failed upload leaked file", err, len(entries))
			}
		})
	}
	repo := &avatarRepo{}
	service := &Service{Repo: repo, Signer: signer}
	dir := t.TempDir()
	var large bytes.Buffer
	png.Encode(&large, image.NewRGBA(image.Rect(0, 0, 4097, 1)))
	var valid bytes.Buffer
	png.Encode(&valid, image.NewRGBA(image.Rect(0, 0, 8, 8)))
	for _, input := range [][]byte{[]byte("<svg/>"), bytes.Repeat([]byte{0}, MaxAvatarBytes+1), large.Bytes(), valid.Bytes()[:len(valid.Bytes())/2]} {
		_, err := service.SaveAvatar(context.Background(), token, dir, bytes.NewReader(input))
		if !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid image accepted", err)
		}
	}
	_, err = service.SaveAvatar(context.Background(), "invalid", dir, bytes.NewReader(valid.Bytes()))
	if !errors.Is(err, ErrUnauthorized) || repo.calls != 0 {
		t.Fatal("invalid upload reached repository", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatal("invalid upload wrote files")
	}
}
