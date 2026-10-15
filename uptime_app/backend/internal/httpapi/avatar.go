package httpapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"
	"uptime_app/backend/internal/avatar"

	"uptime_app/backend/internal/auth"
)

type avatarService interface {
	SaveAvatar(context.Context, auth.Identity, io.Reader) (auth.User, error)
}

func avatarHandler(service avatarService) http.HandlerFunc {
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
		user, err := service.SaveAvatar(r.Context(), currentIdentity(r), file)
		if err != nil {
			if errors.Is(err, auth.ErrInvalid) {
				apiError(w, 400, "invalid_input", "Выберите корректный JPEG или PNG до 5 МБ и 4096×4096 пикселей.")
			} else {
				serviceError(w, err)
			}
			return
		}
		jsonResponse(w, 200, dto(user))
	}
}

func serveAvatar(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("filename")
		if !avatar.Filename.MatchString(name) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeFile(w, r, filepath.Join(dir, name))
	}
}

func limitAvatar(shared RateLimiter, next http.Handler) http.Handler {
	local := &limiter{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := currentIdentity(r).User
		keys := []string{"avatar:user:" + user.ID}
		var ok bool
		var retry int
		if shared != nil {
			var err error
			ok, retry, err = shared.Allow(r.Context(), keys, 10, time.Hour)
			if err != nil {
				serviceError(w, err)
				return
			}
		} else {
			ok, retry = local.allow(keys, 10, time.Hour, time.Now())
		}
		if !ok {
			w.Header().Set("Retry-After", strconv.Itoa(retry))
			apiError(w, 429, "rate_limited", "Слишком много загрузок. Попробуйте позже.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
