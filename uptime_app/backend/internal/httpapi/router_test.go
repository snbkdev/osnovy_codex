package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	recorder := httptest.NewRecorder()
	NewRouter().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if recorder.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatal("expected JSON")
	}
}

func TestSwagger(t *testing.T) {
	router := NewRouter()
	for _, path := range []string{"/swagger/index.html", "/swagger/doc.json"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s: status = %d", path, recorder.Code)
		}
		if path == "/swagger/doc.json" {
			var spec struct {
				Swagger  string                                `json:"swagger"`
				BasePath string                                `json:"basePath"`
				Paths    map[string]map[string]json.RawMessage `json:"paths"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &spec); err != nil {
				t.Fatal(err)
			}
			if spec.Swagger != "2.0" || spec.BasePath != "/api/v1" {
				t.Fatalf("unexpected API metadata: %+v", spec)
			}
			expected := map[string][]string{
				"/health": {"get"}, "/avatars/{filename}": {"get"},
				"/auth/register": {"post"}, "/auth/verify-email": {"post"},
				"/auth/resend-verification": {"post"}, "/auth/forgot-password": {"post"},
				"/auth/reset-password": {"post"}, "/auth/login": {"post"},
				"/auth/refresh": {"post"}, "/auth/logout": {"post"},
				"/auth/me": {"get"}, "/auth/profile": {"post"}, "/auth/avatar": {"post"},
				"/monitors":              {"get", "post"},
				"/monitors/{id}":         {"put", "delete"},
				"/monitors/{id}/history": {"get"},
			}
			if len(spec.Paths) != len(expected) {
				t.Fatalf("documented paths = %d, want %d", len(spec.Paths), len(expected))
			}
			for route, methods := range expected {
				for _, method := range methods {
					if _, ok := spec.Paths[route][method]; !ok {
						t.Errorf("missing operation: %s %s", method, route)
					}
				}
			}
		}
	}
}
