package httpapi

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"uptime_app/backend/internal/auth"
	"uptime_app/backend/internal/config"
)

type avatarStub struct {
	stubService
	uploadErr error
	uploads   int
}

func (s *avatarStub) SaveAvatar(_ context.Context, _ string, _ string, input io.Reader) (auth.User, error) {
	s.uploads++
	io.Copy(io.Discard, input)
	return auth.User{ID: "user", AvatarURL: "/api/v1/avatars/" + strings.Repeat("a", 32) + ".png"}, s.uploadErr
}
func TestAvatarHTTP(t *testing.T) {
	service := &avatarStub{}
	dir := t.TempDir()
	handler := NewAuthRouter(service, config.Config{AvatarDir: dir})
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, _ := writer.CreateFormFile("avatar", "photo.png")
	file.Write([]byte("image"))
	writer.Close()
	send := func(data []byte, contentType string, authenticated bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/v1/auth/avatar", bytes.NewReader(data))
		r.Header.Set("Content-Type", contentType)
		if authenticated {
			r.Header.Set("Authorization", "Bearer token")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	w := send(body.Bytes(), writer.FormDataContentType(), true)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"avatar_url"`) || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := send(body.Bytes(), writer.FormDataContentType(), false); w.Code != 401 || service.uploads != 1 {
		t.Fatal("anonymous upload", w.Code)
	}
	if w := send([]byte(`{}`), "application/json", true); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w := send(bytes.Repeat([]byte("x"), auth.MaxAvatarBytes+65537), writer.FormDataContentType(), true); w.Code != 400 {
		t.Fatal(w.Code)
	}
	service.uploadErr = auth.ErrInvalid
	if w := send(body.Bytes(), writer.FormDataContentType(), true); w.Code != 400 {
		t.Fatal(w.Code)
	}
	name := strings.Repeat("a", 32) + ".png"
	os.WriteFile(filepath.Join(dir, name), []byte("PNG"), 0600)
	for _, tc := range []struct {
		path   string
		status int
	}{{name, 200}, {"secret.txt", 404}, {strings.Repeat("b", 32) + ".png", 404}, {"", 404}} {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/avatars/"+tc.path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatal(tc.path, w.Code)
		}
		if tc.status == 200 && (w.Header().Get("Content-Type") != "image/png" || w.Header().Get("X-Content-Type-Options") != "nosniff") {
			t.Fatal(w.Header())
		}
	}
}
