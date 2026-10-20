package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"uptime_app/backend/internal/config"
)

func TestCORS(t *testing.T) {
	const origin = "http://localhost:3000"
	for _, tc := range []struct {
		name, origin, method, headers string
		status                        int
		allowed, called               bool
	}{
		{"trusted", origin, "", "", 418, true, true},
		{"server client", "", "", "", 418, false, true},
		{"foreign", "https://evil.test", "", "", 403, false, false},
		{"null", "null", "", "", 403, false, false},
		{"suffix", origin + ".evil.test", "", "", 403, false, false},
		{"preflight", origin, "POST", "content-type, AUTHORIZATION, X-Uptime-Refresh-Request", 204, true, false},
		{"get preflight", origin, "GET", "", 204, true, false},
		{"foreign preflight", "https://evil.test", "POST", "", 403, false, false},
		{"put", origin, "PUT", "Authorization, Content-Type", 204, true, false},
		{"delete", origin, "DELETE", "Authorization", 204, true, false},
		{"forbidden method", origin, "PATCH", "", 403, false, false},
		{"proxy secret", origin, "POST", "X-Uptime-Proxy-Secret", 403, false, false},
		{"client ip", origin, "POST", "X-Uptime-Client-IP", 403, false, false},
		{"unknown header", origin, "POST", "X-Unknown", 403, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			h := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(418)
			}), origin)
			method := "GET"
			if tc.method != "" {
				method = "OPTIONS"
			}
			r := httptest.NewRequest(method, "/api/v1/health", nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if tc.method != "" {
				r.Header.Set("Access-Control-Request-Method", tc.method)
			}
			if tc.headers != "" {
				r.Header.Set("Access-Control-Request-Headers", tc.headers)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status || called != tc.called {
				t.Fatalf("status=%d called=%t", w.Code, called)
			}
			if got := w.Header().Get("Access-Control-Allow-Origin"); (got == origin) != tc.allowed || (!tc.allowed && got != "") {
				t.Fatalf("allow-origin=%q", got)
			}
			if tc.allowed && w.Header().Get("Access-Control-Allow-Credentials") != "true" {
				t.Fatal("credentials missing")
			}
			vary := strings.Join(w.Header().Values("Vary"), ",")
			if !strings.Contains(vary, "Origin") || (tc.method != "" && !strings.Contains(vary, "Access-Control-Request-Headers")) {
				t.Fatalf("vary=%q", vary)
			}
		})
	}
}

func TestCORSRouter(t *testing.T) {
	const origin = "http://localhost:3000"
	h := NewAuthRouter(&stubService{}, config.Config{PublicURL: origin})
	for _, path := range []string{"health", "auth/me", "missing"} {
		r := httptest.NewRequest("GET", "/api/v1/"+path, nil)
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Header().Get("Access-Control-Allow-Origin") != origin {
			t.Fatalf("missing CORS on %s (%d)", path, w.Code)
		}
	}
	for _, origins := range [][]string{{origin, "https://evil.test"}, {""}} {
		r := httptest.NewRequest("GET", "/api/v1/health", nil)
		r.Header["Origin"] = origins
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatalf("origins=%v status=%d", origins, w.Code)
		}
	}
}
