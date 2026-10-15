package storage

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"time"
	"uptime_app/backend/internal/avatar"

	"github.com/uptrace/bun"
)

// Grace exceeds request timeouts, so a new upload cannot be collected before commit.
func CleanupAvatars(ctx context.Context, db *bun.DB, dir string, now time.Time) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.Type().IsRegular() || !avatar.Filename.MatchString(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.ModTime().Before(now.Add(-24 * time.Hour)) {
			continue
		}
		var used bool
		if err := db.NewRaw("SELECT EXISTS(SELECT 1 FROM users WHERE avatar_url=?)", avatar.URLPrefix+entry.Name()).Scan(ctx, &used); err != nil {
			return err
		}
		if !used {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return nil
}

func RunMaintenance(ctx context.Context, db *bun.DB, dir string) {
	run := func() {
		workCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		if _, err := db.ExecContext(workCtx, "DELETE FROM rate_limits WHERE until_at < now() - interval '1 hour'"); err != nil && ctx.Err() == nil {
			slog.Error("rate limit cleanup failed")
		}
		if err := CleanupAvatars(workCtx, db, dir, time.Now()); err != nil && ctx.Err() == nil {
			slog.Error("avatar cleanup failed")
		}
	}
	run()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
