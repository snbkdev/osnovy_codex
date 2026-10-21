package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
func (s *monitorRepositoryStub) List(_ context.Context, owner string, cursor ...string) ([]monitor.Monitor, error) {
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
	repo.err = monitor.ErrQuota
	if w := request("POST", `{"url":"https://example.com","interval_seconds":5}`, true); w.Code != 409 {
		t.Fatal("quota error not mapped", w.Code)
	}
	repo.err = nil
	users.err = auth.ErrUnauthorized
	count := len(repo.values)
	if w := request("POST", `{"url":"https://example.com","interval_seconds":5}`, true); w.Code != 401 || len(repo.values) != count {
		t.Fatal("revoked session accepted", w.Code)
	}
}

func (r *monitorRepositoryStub) Update(_ context.Context, owner, id, url string, interval int) (monitor.Monitor, error) {
	if r.err != nil {
		return monitor.Monitor{}, r.err
	}
	for i, value := range r.values {
		if value.ID == id && value.UserID == owner {
			value.URL = url
			value.IntervalSeconds = interval
			r.values[i] = value
			return value, nil
		}
	}
	return monitor.Monitor{}, monitor.ErrNotFound
}
func (r *monitorRepositoryStub) Delete(_ context.Context, owner, id string) error {
	if r.err != nil {
		return r.err
	}
	for i, value := range r.values {
		if value.ID == id && value.UserID == owner {
			r.values = append(r.values[:i], r.values[i+1:]...)
			return nil
		}
	}
	return monitor.ErrNotFound
}

func TestMonitorMutationHTTP(t *testing.T) {
	repo := &monitorRepositoryStub{values: []monitor.Monitor{{ID: "site", UserID: "id", URL: "https://example.com/", IntervalSeconds: 60}, {ID: "foreign", UserID: "other"}}}
	handler := NewApplicationRouter(&stubService{}, &monitor.Service{Repo: repo}, config.Config{})
	request := func(method, id, body string, authenticated bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "/api/v1/monitors/"+id, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if authenticated {
			r.Header.Set("Authorization", "Bearer token")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	valid := `{"url":"https://NEW.example.com","interval_seconds":5}`
	for _, method := range []string{"PUT", "DELETE"} {
		if w := request(method, "site", valid, false); w.Code != 401 {
			t.Fatal(w.Code, w.Body.String())
		}
		for _, id := range []string{"missing", "foreign"} {
			if w := request(method, id, valid, true); w.Code != 404 || !strings.Contains(w.Body.String(), "monitor_not_found") {
				t.Fatal(w.Code, w.Body.String())
			}
		}
	}
	for _, body := range []string{`{}`, `null`, `[]`, `{`, `{"url":"https://example.com","interval_seconds":5.5}`, `{"url":"https://example.com","interval_seconds":4}`, valid + ` {}`, `{"url":"https://example.com","interval_seconds":5,"user_id":"other"}`} {
		if w := request("PUT", "site", body, true); w.Code != 400 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	w := request("PUT", "site", valid, true)
	var updated monitor.Monitor
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil || w.Code != 200 || updated.URL != "https://new.example.com/" || updated.IntervalSeconds != 5 {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	repo.err = errors.New("unavailable")
	for _, method := range []string{"PUT", "DELETE"} {
		if w := request(method, "site", valid, true); w.Code != 503 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	repo.err = nil
	w = request("DELETE", "site", "", true)
	if w.Code != 204 || w.Body.Len() != 0 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request("DELETE", "site", "", true); w.Code != 404 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func (r *monitorRepositoryStub) History(ctx context.Context, owner, id string, from, to time.Time) (monitor.History, error) {
	if r.err != nil {
		return monitor.History{}, r.err
	}
	values := r.values
	for _, value := range values {
		if value.ID == id && value.UserID == owner {
			return monitor.History{Monitor: value, From: from, To: to, BucketSeconds: 300, Buckets: []monitor.HistoryBucket{}}, nil
		}
	}
	return monitor.History{}, monitor.ErrNotFound
}

func TestMonitorHistoryHTTP(t *testing.T) {
	repo := &monitorRepositoryStub{values: []monitor.Monitor{{ID: "site", UserID: "id", URL: "https://example.com/"}, {ID: "foreign", UserID: "other"}}}
	handler := NewApplicationRouter(&stubService{}, &monitor.Service{Repo: repo}, config.Config{})
	request := func(id string, authenticated bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/api/v1/monitors/"+id+"/history", nil)
		if authenticated {
			r.Header.Set("Authorization", "Bearer token")
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := request("site", false); w.Code != 401 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, id := range []string{"foreign", "missing"} {
		if w := request(id, true); w.Code != 404 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	w := request("site", true)
	var history monitor.History
	if err := json.Unmarshal(w.Body.Bytes(), &history); err != nil || w.Code != 200 || history.Monitor.ID != "site" || history.BucketSeconds != 300 || history.Buckets == nil || history.To.Sub(history.From) != 24*time.Hour || strings.Contains(w.Body.String(), "user_id") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	repo.err = errors.New("storage unavailable")
	if w := request("site", true); w.Code != 503 {
		t.Fatal(w.Code, w.Body.String())
	}
}
