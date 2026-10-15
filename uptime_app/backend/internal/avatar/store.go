package avatar

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Store interface {
	Save(context.Context, image.Image) (string, error)
	Delete(context.Context, string) error
}

type LocalStore struct{ Dir string }

var Filename = regexp.MustCompile(`^[a-f0-9]{32}\.png$`)

const URLPrefix = "/api/v1/avatars/"

func (s LocalStore) Save(ctx context.Context, img image.Image) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(s.Dir, 0700); err != nil {
		return "", err
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	name := hex.EncodeToString(id[:]) + ".png"
	path := filepath.Join(s.Dir, name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	encodeErr := png.Encode(f, img)
	closeErr := f.Close()
	if err := errors.Join(encodeErr, closeErr, ctx.Err()); err != nil {
		return "", errors.Join(err, os.Remove(path))
	}
	return URLPrefix + name, nil
}

func (s LocalStore) Delete(ctx context.Context, url string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	name, ok := strings.CutPrefix(url, URLPrefix)
	if !ok || !Filename.MatchString(name) {
		return errors.New("invalid avatar path")
	}
	err := os.Remove(filepath.Join(s.Dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
