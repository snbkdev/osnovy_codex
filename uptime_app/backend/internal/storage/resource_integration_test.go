package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"uptime_app/backend/internal/monitor"
)

func TestPostgresSharedRateLimit(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Separate instances must share an atomic budget.
			ok, _, err := (&RateLimiter{DB: db}).Allow(ctx, []string{"ip", "email"}, 5, time.Minute)
			if err != nil {
				t.Error(err)
				return
			}
			if ok {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if allowed.Load() != 5 {
		t.Fatal("budget exceeded", allowed.Load())
	}
	ok, retry, err := (&RateLimiter{DB: db}).Allow(ctx, []string{"ip", "new-email"}, 5, time.Minute)
	if err != nil || ok || retry < 1 {
		t.Fatal(ok, retry, err)
	}
	var count int
	if err := db.NewRaw("SELECT count(*) FROM rate_limits").Scan(ctx, &count); err != nil || count != 2 {
		t.Fatal("denied request retained new keys", count, err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE rate_limits SET until_at=now()-interval '1 second'"); err != nil {
		t.Fatal(err)
	}
	ok, _, err = (&RateLimiter{DB: db}).Allow(ctx, []string{"email", "ip"}, 5, time.Minute)
	if err != nil || !ok {
		t.Fatal("window did not reset", ok, err)
	}
}

func TestPostgresMonitorQuotaAndPages(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	service := serviceFor(t, db)
	if err := service.Register(ctx, "quota@example.com", password); err != nil {
		t.Fatal(err)
	}
	user, err := service.Repo.FindUser(ctx, "quota@example.com")
	if err != nil {
		t.Fatal(err)
	}
	repo := &MonitorRepository{DB: db}
	// Equal timestamps exercise the ID tiebreaker in the cursor.
	now := time.Now().UTC().Truncate(time.Microsecond)
	for i := 0; i < monitor.MaxMonitors-1; i++ {
		value := monitor.Monitor{ID: strings.Repeat("a", i+1), UserID: user.ID, URL: "https://example.com/", IntervalSeconds: 5, CreatedAt: now}
		if _, err := repo.Create(ctx, value); err != nil {
			t.Fatal(err)
		}
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for _, id := range []string{"final-one", "final-two"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, err := repo.Create(ctx, monitor.Monitor{ID: id, UserID: user.ID, URL: "https://example.com/", IntervalSeconds: 5, CreatedAt: now})
			if err == nil {
				successes.Add(1)
			} else if !errors.Is(err, monitor.ErrQuota) {
				t.Error(err)
			}
		}(id)
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatal("non-atomic quota", successes.Load())
	}
	seen := map[string]bool{}
	cursor := ""
	for {
		page, err := repo.List(ctx, user.ID, cursor)
		if err != nil || len(page) > monitor.PageSize {
			t.Fatal(len(page), err)
		}
		for _, item := range page {
			if seen[item.ID] {
				t.Fatal("duplicate cursor item", item.ID)
			}
			seen[item.ID] = true
		}
		if len(page) < monitor.PageSize {
			break
		}
		cursor = page[len(page)-1].ID
	}
	if len(seen) != monitor.MaxMonitors {
		t.Fatal("pagination lost items", len(seen))
	}
	page, err := repo.List(ctx, "other-user", "final-one")
	if err != nil || len(page) != 0 {
		t.Fatal("cursor leaked another user's monitors", page, err)
	}
}

func TestPostgresAvatarCleanup(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	service := serviceFor(t, db)
	if err := service.Register(ctx, "avatar-cleanup@example.com", password); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	now := time.Now()
	live := strings.Repeat("a", 32) + ".png"
	orphan := strings.Repeat("b", 32) + ".png"
	recent := strings.Repeat("c", 32) + ".png"
	for _, name := range []string{live, orphan, recent, "keep.txt"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
			t.Fatal(err)
		}
		if name != recent {
			if err := os.Chtimes(path, now.Add(-48*time.Hour), now.Add(-48*time.Hour)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := db.ExecContext(ctx, "UPDATE users SET avatar_url=? WHERE email=?", "/api/v1/avatars/"+live, "avatar-cleanup@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := CleanupAvatars(ctx, db, dir, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, orphan)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("orphan retained", err)
	}
	for _, name := range []string{live, recent, "keep.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal("retained file removed", name, err)
		}
	}
	if err := CleanupAvatars(ctx, db, dir, now); err != nil {
		t.Fatal("cleanup not repeatable", err)
	}
}
