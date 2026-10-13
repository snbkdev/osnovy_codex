package httpapi

import (
	"context"
	"net/http"
	"strings"
	"uptime_app/backend/internal/auth"
)

type userContextKey struct{}

func requireAuth(service authService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		bearer := r.Header.Get("Authorization")
		if !strings.HasPrefix(bearer, "Bearer ") {
			serviceError(w, auth.ErrUnauthorized)
			return
		}
		user, err := service.Me(r.Context(), strings.TrimPrefix(bearer, "Bearer "))
		if err != nil {
			serviceError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey{}, user)))
	})
}
