package httpapi

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"
	_ "uptime_app/backend/docs"
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
	mux.HandleFunc("GET /api/v1/health", healthHandler)
	mux.Handle("GET /swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))
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
		mux.Handle("GET /api/v1/auth/me", requireAuth(service, http.HandlerFunc(meHandler)))
	}
	return cors(mux, cfg.PublicURL)
}

// @Summary Check API health
// @Tags Health
// @Produce json
// @Success 200 {object} healthResponse
// @Router /health [get]
func healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, healthResponse{Status: "ok"})
}

// @Summary Get current user
// @Tags Profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} userDTO
// @Failure 401 {object} errorResponse
// @Failure 503 {object} errorResponse
// @Router /auth/me [get]
func meHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, dto(currentIdentity(r).User))
}
