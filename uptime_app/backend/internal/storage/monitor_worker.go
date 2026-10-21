package storage

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"uptime_app/backend/internal/identifier"
	"uptime_app/backend/internal/monitor"
)

// Claim one due monitor atomically. The lease recovers interrupted checks after 30s.
func (r *MonitorRepository) claimCheck(ctx context.Context) (monitorRow, error) {
	var row monitorRow
	err := r.DB.NewRaw(`UPDATE monitors SET next_check_at=now()+interval '30 seconds',check_token=?
 WHERE id=(SELECT id FROM monitors WHERE next_check_at<=now() ORDER BY next_check_at,id FOR UPDATE SKIP LOCKED LIMIT 1)
 RETURNING *`, identifier.New()).Scan(ctx, &row)
	return row, err
}

func (r *MonitorRepository) finishCheck(ctx context.Context, row monitorRow, result monitor.CheckResult) error {
	_, err := r.DB.NewRaw(`WITH completed AS (UPDATE monitors SET status=?,http_status=?,check_error=?,checked_at=?,
 next_check_at=now()+interval_seconds*interval '1 second',check_token=''
 WHERE id=? AND check_token=? RETURNING id,url,status,http_status,check_error,checked_at)
 INSERT INTO monitor_checks(monitor_id,url,status,http_status,check_error,checked_at)
 SELECT id,url,status,http_status,check_error,checked_at FROM completed`, result.Status, result.HTTPStatus, result.CheckError, result.CheckedAt, row.ID, row.CheckToken).Exec(ctx)
	return err
}

func (r *MonitorRepository) checkNext(ctx context.Context, client *http.Client) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	row, err := r.claimCheck(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	result := monitor.Check(ctx, client, row.URL)
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	return true, r.finishCheck(ctx, row, result)
}

func (r *MonitorRepository) RunChecks(ctx context.Context) {
	client := monitor.NewCheckClient()
	defer client.CloseIdleConnections()
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			timer := time.NewTimer(0)
			defer timer.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
				}
				worked, err := r.checkNext(ctx, client)
				if err != nil && ctx.Err() == nil {
					slog.Error("monitor check failed to persist")
				}
				delay := time.Second
				if worked && err == nil {
					delay = 0
				}
				timer.Reset(delay)
			}
		})
	}
	wg.Wait()
}
