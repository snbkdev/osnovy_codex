package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"uptime_app/backend/internal/auth"
	"uptime_app/backend/internal/config"
	"uptime_app/backend/internal/monitor"
)

type monitorRepositoryStub struct {
	values []monitor.Monitor
	owner  string
	err    error
}

func (s *monitorRepositoryStub) Create(_ context.Context, value monitor.Monitor) (monitor.Monitor, error) {
	s.values = append(s.values, value)
	return value, s.err
}
func (s *monitorRepositoryStub) List(_ context.Context, owner string) ([]monitor.Monitor, error) {
	s.owner = owner
	return s.values, s.err
}

func TestMonitorHTTP(t *testing.T) {
	users := &stubService{}
	repo := &monitorRepositoryStub{}
	handler := NewApplicationRouter(users, &monitor.Service{Repo: repo}, config.Config{})
	request := func(method, body string, authenticated bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "/api/v1/monitors", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if authenticated {
			r.Header.Set("Authorization", "Bearer token")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	w := request("GET", "", true)
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" || repo.owner != "id" {
		t.Fatal(w.Code, w.Body.String(), repo.owner)
	}
	for _, method := range []string{"POST", "GET"} {
		if w := request(method, `{"url":"https://example.com","interval_seconds":5}`, false); w.Code != 401 {
			t.Fatal("anonymous request accepted", w.Code)
		}
	}
	w = request("POST", `{"url":"https://example.com","interval_seconds":420}`, true)
	var created monitor.Monitor
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if w.Code != 201 || created.URL != "https://example.com/" || created.IntervalSeconds != 420 || len(repo.values) != 1 || repo.values[0].UserID != "id" || strings.Contains(w.Body.String(), "user_id") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String())
	}
	w = request("GET", "", true)
	var list []monitor.Monitor
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || w.Code != 200 || len(list) != 1 || list[0].ID != created.ID {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, body := range []string{
		`{}`, `null`, `[]`, `{"url":"ftp://example.com","interval_seconds":5}`,
		`{"url":"https://example.com","interval_seconds":4}`,
		`{"url":"https://example.com","interval_seconds":5.5}`,
		`{"url":"https://example.com","interval_seconds":"60"}`,
		`{"url":"https://example.com","interval_seconds":5,"user_id":"other"}`,
		`{"url":"https://example.com","interval_seconds":5} {}`,
		`{"url":"https://example.com/` + strings.Repeat("a", 5000) + `","interval_seconds":5}`,
	} {
		if w := request("POST", body, true); w.Code != 400 {
			t.Fatal("invalid input", body, w.Code, w.Body.String())
		}
	}
	if len(repo.values) != 1 {
		t.Fatal("invalid request created a monitor")
	}
	repo.err = errors.New("database unavailable")
	for _, method := range []string{"POST", "GET"} {
		if w := request(method, `{"url":"https://example.com","interval_seconds":5}`, true); w.Code != 503 {
			t.Fatal("missing storage error", w.Code)
		}
	}
	repo.err = nil
	users.err = auth.ErrUnauthorized
	count := len(repo.values)
	if w := request("POST", `{"url":"https://example.com","interval_seconds":5}`, true); w.Code != 401 || len(repo.values) != count {
		t.Fatal("revoked session accepted", w.Code)
	}
}
