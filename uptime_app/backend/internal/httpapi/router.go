package httpapi

import (
	"net/http"
	"uptime_app/backend/internal/config"
)

func NewRouter() http.Handler { return newRouter(nil, config.Config{}) }

func NewAuthRouter(service authService, cfg config.Config) http.Handler {
	return newRouter(service, cfg)
}

func newRouter(service authService, cfg config.Config) http.Handler {
	return NewApplicationRouter(service, nil, cfg)
}

func NewApplicationRouter(service authService, monitors monitorService, cfg config.Config) http.Handler {
	return NewLimitedApplicationRouter(service, monitors, cfg, nil)
}
func NewLimitedApplicationRouter(service authService, monitors monitorService, cfg config.Config, limits RateLimiter) http.Handler {
	mux := http.NewServeMux()
	if cfg.AvatarDir == "" {
		cfg.AvatarDir = "uploads/avatars"
	}
	mux.HandleFunc("GET /api/v1/avatars/{filename}", serveAvatar(cfg.AvatarDir))
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}` + "\n"))
	})
	if service != nil {
		if monitors != nil {
			mux.Handle("POST /api/v1/monitors", requireAuth(service, createMonitorHandler(monitors)))
			mux.Handle("GET /api/v1/monitors", requireAuth(service, listMonitorsHandler(monitors)))
		}
		if avatars, ok := service.(avatarService); ok {
			mux.Handle("POST /api/v1/auth/avatar", requireAuth(service, limitAvatar(limits, avatarHandler(avatars))))
		}
		mux.Handle("/api/v1/auth/", &authHandler{service: service, config: cfg, shared: limits})
		mux.Handle("POST /api/v1/auth/profile", requireAuth(service, http.HandlerFunc(profileHandler(service))))
		mux.Handle("GET /api/v1/auth/me", requireAuth(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := currentIdentity(r).User
			jsonResponse(w, 200, dto(user))
		})))
	}
	return cors(mux, cfg.PublicURL)
}
