package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uptime_app/backend/internal/auth"
	"uptime_app/backend/internal/config"
)

func TestProfileHTTP(t *testing.T) {
	service := &stubService{}
	handler := NewAuthRouter(service, config.Config{})
	w := req(handler, "POST", "profile", `{"full_name":"Иванов Иван","date_of_birth":"1990-05-12"}`, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"full_name":"Иванов Иван"`) || strings.Contains(w.Body.String(), "password") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, body := range []string{`{"user_id":"other"}`, `{} {}`, `{"date_of_birth":123}`, `{"full_name":"` + strings.Repeat("a", 17000) + `"}`} {
		if w := req(handler, "POST", "profile", body, ""); w.Code != 400 {
			t.Fatal("invalid DTO accepted", w.Code)
		}
	}
	calls := service.calls
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/auth/profile", strings.NewReader(`{}`)))
	if w.Code != 401 || service.calls != calls {
		t.Fatal("anonymous update accepted")
	}
	service.err = auth.ErrInvalid
	w = req(handler, "POST", "profile", `{}`, "")
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}
