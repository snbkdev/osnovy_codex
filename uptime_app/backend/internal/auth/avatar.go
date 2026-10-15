package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"time"
)

const MaxAvatarBytes = 5 * 1024 * 1024

// SaveAvatar validates and re-encodes uploads so only actual PNG image data is served.
func (s *Service) SaveAvatar(ctx context.Context, bearer, dir string, input io.Reader) (User, error) {
	now := time.Now()
	claims, err := s.Signer.Parse(bearer, now)
	if err != nil {
		return User{}, err
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
	if err := os.MkdirAll(dir, 0700); err != nil {
		return User{}, err
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return User{}, err
	}
	filename := hex.EncodeToString(id[:]) + ".png"
	path := filepath.Join(dir, filename)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return User{}, err
	}
	encodeErr := png.Encode(f, dst)
	closeErr := f.Close()
	if encodeErr != nil || closeErr != nil {
		os.Remove(path)
		if encodeErr != nil {
			return User{}, encodeErr
		}
		return User{}, closeErr
	}
	user, err := s.Repo.UpdateAvatar(ctx, claims.Subject, claims.SessionID, "/api/v1/avatars/"+filename, now)
	if err != nil {
		os.Remove(path)
	}
	return user, err
}
