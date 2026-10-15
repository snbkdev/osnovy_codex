package storage

import (
	"context"
	"time"

	"github.com/uptrace/bun"
	"uptime_app/backend/internal/monitor"
)

type MonitorRepository struct{ DB *bun.DB }

var _ monitor.Repository = (*MonitorRepository)(nil)

type monitorRow struct {
	ID              string
	UserID          string
	URL             string
	IntervalSeconds int
	CreatedAt       time.Time
}

func publicMonitor(row monitorRow) monitor.Monitor {
	return monitor.Monitor{ID: row.ID, UserID: row.UserID, URL: row.URL, IntervalSeconds: row.IntervalSeconds, CreatedAt: row.CreatedAt}
}

func (r *MonitorRepository) Create(ctx context.Context, value monitor.Monitor) (monitor.Monitor, error) {
	var row monitorRow
	err := r.DB.NewRaw("INSERT INTO monitors (id,user_id,url,interval_seconds,created_at) VALUES (?,?,?,?,?) RETURNING *", value.ID, value.UserID, value.URL, value.IntervalSeconds, value.CreatedAt).Scan(ctx, &row)
	return publicMonitor(row), err
}

func (r *MonitorRepository) List(ctx context.Context, userID string) ([]monitor.Monitor, error) {
	var rows []monitorRow
	if err := r.DB.NewRaw("SELECT * FROM monitors WHERE user_id=? ORDER BY created_at DESC,id DESC", userID).Scan(ctx, &rows); err != nil {
		return nil, err
	}
	result := make([]monitor.Monitor, 0, len(rows))
	for _, row := range rows {
		result = append(result, publicMonitor(row))
	}
	return result, nil
}
