package httpapi

import (
	"net/http"
	"strings"
)

// cors uses the same trusted origin as the cookie authentication handlers.
func cors(next http.Handler, publicURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		preflight := r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
		if preflight {
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")
		}
		origins := r.Header.Values("Origin")
		if len(origins) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		if len(origins) != 1 || publicURL == "" || origins[0] != publicURL {
			apiError(w, http.StatusForbidden, "invalid_origin", "Недопустимый источник запроса.")
			return
		}
		if preflight {
			switch r.Header.Get("Access-Control-Request-Method") {
			case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodDelete:
			default:
				apiError(w, http.StatusForbidden, "invalid_cors_request", "Недопустимый CORS-запрос.")
				return
			}
			for _, value := range r.Header.Values("Access-Control-Request-Headers") {
				for _, name := range strings.Split(value, ",") {
					switch strings.ToLower(strings.TrimSpace(name)) {
					case "accept", "accept-language", "content-language", "content-type", "authorization", "x-uptime-refresh-request":
					default:
						apiError(w, http.StatusForbidden, "invalid_cors_request", "Недопустимый CORS-запрос.")
						return
					}
				}
			}
		}
		w.Header().Set("Access-Control-Allow-Origin", publicURL)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "Retry-After")
		if preflight {
			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Language, Content-Type, Authorization, X-Uptime-Refresh-Request")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
