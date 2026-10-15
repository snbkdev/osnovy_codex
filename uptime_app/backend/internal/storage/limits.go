package storage

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/uptrace/bun"
	"uptime_app/backend/internal/auth"
)

var errRateLimited = errors.New("rate_limited")

type RateLimiter struct{ DB *bun.DB }

// All keys are locked in the same order. Hashing bounds storage and hides emails.
func (l *RateLimiter) Allow(ctx context.Context, keys []string, max int, window time.Duration) (bool, int, error) {
	hashes := make([]string, len(keys))
	for i, key := range keys {
		hashes[i] = auth.TokenHash(key)
	}
	sort.Strings(hashes)
	retry := 0
	err := l.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var now time.Time
		if err := tx.NewRaw("SELECT clock_timestamp()").Scan(ctx, &now); err != nil {
			return err
		}
		for _, key := range hashes {
			if _, err := tx.ExecContext(ctx, "INSERT INTO rate_limits (key,count,until_at) VALUES (?,0,?) ON CONFLICT DO NOTHING", key, now.Add(window)); err != nil {
				return err
			}
			var row struct {
				Count   int
				UntilAt time.Time
			}
			if err := tx.NewRaw("SELECT count,until_at FROM rate_limits WHERE key=? FOR UPDATE", key).Scan(ctx, &row); err != nil {
				return err
			}
			if row.UntilAt.After(now) && row.Count >= max {
				seconds := int(row.UntilAt.Sub(now).Seconds()) + 1
				if seconds > retry {
					retry = seconds
				}
			}
		}
		if retry > 0 {
			return errRateLimited
		}
		for _, key := range hashes {
			if _, err := tx.ExecContext(ctx, "UPDATE rate_limits SET count=CASE WHEN until_at<=? THEN 1 ELSE count+1 END, until_at=CASE WHEN until_at<=? THEN ? ELSE until_at END WHERE key=?", now, now, now.Add(window), key); err != nil {
				return err
			}
		}
		return nil
	})
	if errors.Is(err, errRateLimited) {
		return false, retry, nil
	}
	return err == nil, retry, err
}
