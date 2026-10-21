package storage

import (
	"context"
	"time"
	"uptime_app/backend/internal/monitor"
)

func (r *MonitorRepository) History(ctx context.Context, userID, id string, from, to time.Time) (monitor.History, error) {
	var row monitorRow
	if err := r.DB.NewRaw("SELECT * FROM monitors WHERE user_id=? AND id=?", userID, id).Scan(ctx, &row); err != nil {
		return monitor.History{}, missing(err, monitor.ErrNotFound)
	}
	buckets := make([]monitor.HistoryBucket, 0)
	err := r.DB.NewRaw(`SELECT date_bin(interval '5 minutes',c.checked_at,timestamptz '2000-01-01') AS start,
 count(*) FILTER (WHERE c.status='success') AS success,
 count(*) FILTER (WHERE c.status='error') AS failure
 FROM monitor_checks c JOIN monitors m ON m.id=c.monitor_id
 WHERE m.user_id=? AND m.id=? AND c.url=? AND c.checked_at>=? AND c.checked_at<=?
 GROUP BY start ORDER BY start`, userID, id, row.URL, from, to).Scan(ctx, &buckets)
	return monitor.History{Monitor: publicMonitor(row), From: from, To: to, BucketSeconds: 300, Buckets: buckets}, err
}
