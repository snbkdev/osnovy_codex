package storage

import (
	"context"
	"time"
	"uptime_app/backend/internal/auth"

	"github.com/uptrace/bun"
	"uptime_app/backend/internal/monitor"
)

type MonitorRepository struct{ DB *bun.DB }

var _ monitor.Repository = (*MonitorRepository)(nil)

type monitorRow struct {
	Status          string
	HTTPStatus      int
	CheckError      string
	CheckedAt       *time.Time
	NextCheckAt     time.Time
	CheckToken      string
	ID              string
	UserID          string
	URL             string
	IntervalSeconds int
	CreatedAt       time.Time
}

func publicMonitor(row monitorRow) monitor.Monitor {
	return monitor.Monitor{Status: row.Status, HTTPStatus: row.HTTPStatus, CheckError: row.CheckError, CheckedAt: row.CheckedAt, ID: row.ID, UserID: row.UserID, URL: row.URL, IntervalSeconds: row.IntervalSeconds, CreatedAt: row.CreatedAt}
}

func (r *MonitorRepository) Create(ctx context.Context, value monitor.Monitor) (monitor.Monitor, error) {
	var row monitorRow
	err := r.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := lockUser(ctx, tx, value.UserID); err != nil {
			return missing(err, auth.ErrUnauthorized)
		}
		var count int
		if err := tx.NewRaw("SELECT count(*) FROM monitors WHERE user_id=?", value.UserID).Scan(ctx, &count); err != nil {
			return err
		}
		if count >= monitor.MaxMonitors {
			return monitor.ErrQuota
		}
		return tx.NewRaw("INSERT INTO monitors (id,user_id,url,interval_seconds,created_at) VALUES (?,?,?,?,?) RETURNING *", value.ID, value.UserID, value.URL, value.IntervalSeconds, value.CreatedAt).Scan(ctx, &row)
	})
	return publicMonitor(row), err
}

func (r *MonitorRepository) List(ctx context.Context, userID string, cursor ...string) ([]monitor.Monitor, error) {
	var rows []monitorRow
	var err error
	if len(cursor) == 0 || cursor[0] == "" {
		err = r.DB.NewRaw("SELECT * FROM monitors WHERE user_id=? ORDER BY created_at DESC,id DESC LIMIT ?", userID, monitor.PageSize).Scan(ctx, &rows)
	} else {
		err = r.DB.NewRaw("SELECT * FROM monitors WHERE user_id=? AND (created_at,id) < (SELECT created_at,id FROM monitors WHERE user_id=? AND id=?) ORDER BY created_at DESC,id DESC LIMIT ?", userID, userID, cursor[0], monitor.PageSize).Scan(ctx, &rows)
	}
	if err != nil {
		return nil, err
	}
	result := make([]monitor.Monitor, 0, len(rows))
	for _, row := range rows {
		result = append(result, publicMonitor(row))
	}
	return result, nil
}

func (r *MonitorRepository) Update(ctx context.Context, userID, id, url string, interval int) (monitor.Monitor, error) {
	var row monitorRow
	err := r.DB.NewRaw("UPDATE monitors SET url=?,interval_seconds=?,status='pending',http_status=0,check_error='',checked_at=NULL,next_check_at=now(),check_token='' WHERE user_id=? AND id=? RETURNING *", url, interval, userID, id).Scan(ctx, &row)
	return publicMonitor(row), missing(err, monitor.ErrNotFound)
}

func (r *MonitorRepository) Delete(ctx context.Context, userID, id string) error {
	var deletedID string
	err := r.DB.NewRaw("DELETE FROM monitors WHERE user_id=? AND id=? RETURNING id", userID, id).Scan(ctx, &deletedID)
	return missing(err, monitor.ErrNotFound)
}
