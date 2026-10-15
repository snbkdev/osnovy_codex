package avatar

import (
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStoreBoundary(t *testing.T) {
	dir := t.TempDir()
	store := LocalStore{Dir: dir}
	outside := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{"/api/v1/avatars/../keep.txt", "keep.txt", "https://example.com/avatar.png"} {
		if err := store.Delete(context.Background(), url); err == nil {
			t.Fatal("untrusted path accepted", url)
		}
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("unrelated file removed", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Save(ctx, image.NewRGBA(image.Rect(0, 0, 1, 1))); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	url, err := store.Save(context.Background(), image.NewRGBA(image.Rect(0, 0, 1, 1)))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := store.Delete(context.Background(), url); err != nil {
			t.Fatal("delete not repeatable", err)
		}
	}
}
