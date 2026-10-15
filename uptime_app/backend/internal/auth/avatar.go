package auth

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"time"
)

const MaxAvatarBytes = 5 * 1024 * 1024

// SaveAvatar validates and re-encodes uploads so only actual PNG image data is served.
func (s *Service) SaveAvatar(ctx context.Context, identity Identity, input io.Reader) (User, error) {
	now := time.Now()
	if !identity.Valid() {
		return User{}, ErrUnauthorized
	}
	if s.Avatars == nil {
		return User{}, errors.New("avatar store unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(input, MaxAvatarBytes+1))
	if err != nil {
		return User{}, err
	}
	if len(data) > MaxAvatarBytes {
		return User{}, ErrInvalid
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 4096 || cfg.Height > 4096 || cfg.Width*cfg.Height > 16000000 {
		return User{}, ErrInvalid
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return User{}, ErrInvalid
	}
	// Normalize to a square 256px avatar, even for direct API clients.
	side := min(cfg.Width, cfg.Height)
	x0, y0 := (cfg.Width-side)/2, (cfg.Height-side)/2
	dst := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			dst.Set(x, y, src.At(x0+x*side/256, y0+y*side/256))
		}
	}
	url, err := s.Avatars.Save(ctx, dst)
	if err != nil {
		return User{}, err
	}
	user, err := s.Repo.UpdateAvatar(ctx, identity.User.ID, identity.SessionID, url, now)
	if err != nil {
		// The request may have been cancelled; cleanup must still run.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		err = errors.Join(err, s.Avatars.Delete(cleanupCtx, url))
	}
	return user, err
}
