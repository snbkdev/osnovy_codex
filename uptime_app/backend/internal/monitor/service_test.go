package monitor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type stubRepository struct {
	created    []Monitor
	listedUser string
	err        error
}

func (r *stubRepository) Create(_ context.Context, value Monitor) (Monitor, error) {
	r.created = append(r.created, value)
	return value, r.err
}
func (r *stubRepository) List(_ context.Context, userID string, cursor ...string) ([]Monitor, error) {
	r.listedUser = userID
	return r.created, r.err
}

func TestCreateMonitor(t *testing.T) {
	repo := &stubRepository{}
	service := &Service{Repo: repo}
	for _, interval := range []int{5, 60, 300, 420, 3600, MaxIntervalSeconds} {
		value, err := service.Create(context.Background(), "owner", "  https://EXAMPLE.com/status?check=1#section  ", interval)
		if err != nil || value.ID == "" || value.UserID != "owner" || value.URL != "https://example.com/status?check=1" || value.IntervalSeconds != interval || value.CreatedAt.IsZero() {
			t.Fatalf("%+v %v", value, err)
		}
	}
	for _, interval := range []int{-1, 0, 1, 4, MaxIntervalSeconds + 1} {
		if _, err := service.Create(context.Background(), "owner", "https://example.com", interval); !errors.Is(err, ErrInvalid) {
			t.Fatal("accepted interval", interval, err)
		}
	}
	if len(repo.created) != 6 {
		t.Fatal("invalid input reached repository")
	}
	if _, err := service.Create(context.Background(), "", "https://example.com", 60); !errors.Is(err, ErrUnauthorized) {
		t.Fatal(err)
	}
	repo.err = errors.New("database unavailable")
	if _, err := service.Create(context.Background(), "owner", "https://example.com", 60); !errors.Is(err, repo.err) {
		t.Fatal("storage error lost", err)
	}
}

func TestNormalizeMonitorURL(t *testing.T) {
	for input, want := range map[string]string{
		"https://EXAMPLE.com":               "https://example.com/",
		"http://localhost:8080/health":      "http://localhost:8080/health",
		"https://example.com/a%20b?q=1&x=2": "https://example.com/a%20b?q=1&x=2",
		"http://[::1]:8080/":                "http://[::1]:8080/",
	} {
		got, err := normalizeURL(input)
		if err != nil || got != want {
			t.Fatal(input, got, err)
		}
	}
	for _, input := range []string{"", "example.com", "//example.com", "ftp://example.com", "javascript:alert(1)", "https://", "https://user:pass@example.com", "https://example.com:0", "https://example.com:65536", "https://example.com:abc", "https://example.com:", "https://example.com/a b", "https://example.com/a\nb", "https://bad_host.test", "https://-bad.test", "https://bad..test", "https://example.com/" + strings.Repeat("a", 2048), "https://example.com/" + strings.Repeat("я", 600)} {
		if _, err := normalizeURL(input); !errors.Is(err, ErrInvalid) {
			t.Fatal("accepted URL", input, err)
		}
	}
}

func TestListMonitors(t *testing.T) {
	repo := &stubRepository{}
	service := &Service{Repo: repo}
	if _, err := service.List(context.Background(), "owner"); err != nil || repo.listedUser != "owner" {
		t.Fatal("wrong owner", err)
	}
	if _, err := service.List(context.Background(), ""); !errors.Is(err, ErrUnauthorized) {
		t.Fatal(err)
	}
	repo.err = errors.New("database unavailable")
	if _, err := service.List(context.Background(), "owner"); !errors.Is(err, repo.err) {
		t.Fatal(err)
	}
}

func (r *stubRepository) Update(_ context.Context, owner, id, url string, interval int) (Monitor, error) {
	if r.err != nil {
		return Monitor{}, r.err
	}
	for i, value := range r.created {
		if value.ID == id && value.UserID == owner {
			value.URL = url
			value.IntervalSeconds = interval
			r.created[i] = value
			return value, nil
		}
	}
	return Monitor{}, ErrNotFound
}
func (r *stubRepository) Delete(_ context.Context, owner, id string) error {
	if r.err != nil {
		return r.err
	}
	for i, value := range r.created {
		if value.ID == id && value.UserID == owner {
			r.created = append(r.created[:i], r.created[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func TestMonitorMutations(t *testing.T) {
	ctx := context.Background()
	repo := &stubRepository{}
	service := &Service{Repo: repo}
	original, err := service.Create(ctx, "owner", "https://example.com", 60)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.Update(ctx, "owner", original.ID, " https://NEW.example.com#fragment ", 5)
	if err != nil || updated.URL != "https://new.example.com/" || updated.ID != original.ID || !updated.CreatedAt.Equal(original.CreatedAt) {
		t.Fatal(updated, err)
	}
	for _, interval := range []int{0, 4, MaxIntervalSeconds + 1} {
		if _, err := service.Update(ctx, "owner", original.ID, "https://example.com", interval); !errors.Is(err, ErrInvalid) {
			t.Fatal(err)
		}
	}
	if _, err := service.Update(ctx, "owner", original.ID, "ftp://example.com", 60); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := service.Update(ctx, "", original.ID, "https://example.com", 60); !errors.Is(err, ErrUnauthorized) {
		t.Fatal(err)
	}
	if err := service.Delete(ctx, "", original.ID); !errors.Is(err, ErrUnauthorized) {
		t.Fatal(err)
	}
	if _, err := service.Update(ctx, "other", original.ID, "https://example.com", 60); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if err := service.Delete(ctx, "other", original.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	repo.err = errors.New("unavailable")
	if _, err := service.Update(ctx, "owner", original.ID, "https://example.com", 60); !errors.Is(err, repo.err) {
		t.Fatal(err)
	}
	if err := service.Delete(ctx, "owner", original.ID); !errors.Is(err, repo.err) {
		t.Fatal(err)
	}
	repo.err = nil
	if err := service.Delete(ctx, "owner", original.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(ctx, "owner", original.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func (r *stubRepository) History(ctx context.Context, owner, id string, from, to time.Time) (History, error) {
	if r.err != nil {
		return History{}, r.err
	}
	values := r.created
	for _, value := range values {
		if value.ID == id && value.UserID == owner {
			return History{Monitor: value, From: from, To: to, BucketSeconds: 300, Buckets: []HistoryBucket{}}, nil
		}
	}
	return History{}, ErrNotFound
}
