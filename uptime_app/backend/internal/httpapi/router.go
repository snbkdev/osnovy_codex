package httpapi

import (
	"net/http"
	"uptime_app/backend/internal/auth"
	"uptime_app/backend/internal/config"
)

func NewRouter() http.Handler { return newRouter(nil, config.Config{}) }

func NewAuthRouter(service authService, cfg config.Config) http.Handler {
	return newRouter(service, cfg)
}

func newRouter(service authService, cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}` + "\n"))
	})
	if service != nil {
		mux.Handle("/api/v1/auth/", &authHandler{service: service, config: cfg})
		mux.Handle("POST /api/v1/auth/profile", requireAuth(service, http.HandlerFunc(profileHandler(service))))
		mux.Handle("GET /api/v1/auth/me", requireAuth(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := r.Context().Value(userContextKey{}).(auth.User)
			jsonResponse(w, 200, dto(user))
		})))
	}
	return mux
}
