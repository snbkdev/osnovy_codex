package httpapi

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"uptime_app/backend/internal/auth"
)

type avatarService interface {
	SaveAvatar(context.Context, string, string, io.Reader) (auth.User, error)
}

func avatarHandler(service avatarService, dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, auth.MaxAvatarBytes+64*1024)
		if err := r.ParseMultipartForm(auth.MaxAvatarBytes + 64*1024); err != nil {
			apiError(w, 400, "invalid_input", "Выберите JPEG или PNG размером до 5 МБ.")
			return
		}
		defer r.MultipartForm.RemoveAll()
		file, _, err := r.FormFile("avatar")
		if err != nil {
			apiError(w, 400, "invalid_input", "Необходимо изображение в поле avatar.")
			return
		}
		defer file.Close()
		user, err := service.SaveAvatar(r.Context(), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), dir, file)
		if err != nil {
			if err == auth.ErrInvalid {
				apiError(w, 400, "invalid_input", "Выберите корректный JPEG или PNG до 5 МБ и 4096×4096 пикселей.")
			} else {
				serviceError(w, err)
			}
			return
		}
		jsonResponse(w, 200, dto(user))
	}
}

var avatarFilename = regexp.MustCompile(`^[a-f0-9]{32}\.png$`)

func serveAvatar(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("filename")
		if !avatarFilename.MatchString(name) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeFile(w, r, filepath.Join(dir, name))
	}
}
