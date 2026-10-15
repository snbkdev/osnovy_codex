package storage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/uptrace/bun"
	"log/slog"
	"sync/atomic"
	"time"
	"uptime_app/backend/internal/auth"
	"uptime_app/backend/internal/mail"
)

var mailDelivered atomic.Uint64
var mailFailed atomic.Uint64
var mailRetries atomic.Uint64

// DeliveryStats returns process-local counters without message contents.
func DeliveryStats() (delivered, failed, retries uint64) {
	return mailDelivered.Load(), mailFailed.Load(), mailRetries.Load()
}

type MailWorker struct {
	DB     *bun.DB
	Cipher *auth.Cipher
	Sender mail.Sender
}

// The transaction holds the selected row while sending. SMTP is at-least-once:
// a crash after SMTP acceptance and before commit can cause a duplicate letter.
func (w *MailWorker) DeliverOne(ctx context.Context, now time.Time) (bool, error) {
	found := false
	var status, code string
	var id int64
	var attempt int
	err := w.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var row mailRow
		err := tx.NewRaw("SELECT * FROM mail_queue WHERE finished_at IS NULL AND available_at<=? ORDER BY available_at,id LIMIT 1 FOR UPDATE SKIP LOCKED", now).Scan(ctx, &row)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		letter, deliveryErr := w.Cipher.Decrypt(row.Payload)
		code = ""
		if deliveryErr != nil {
			code = "decrypt_error"
		} else {
			deliveryErr = w.Sender.Send(ctx, letter)
			if deliveryErr != nil {
				code = "smtp_error"
			}
		}
		attempts := row.Attempts + 1
		id, attempt = row.ID, attempts
		if deliveryErr == nil || attempts >= 5 {
			status = "delivered"
			if deliveryErr != nil {
				status = "failed"
			}
			_, err = tx.ExecContext(ctx, "UPDATE mail_queue SET payload=NULL,finished_at=?,attempts=?,status=?,error_code=NULLIF(?,'') WHERE id=?", now, attempts, status, code, row.ID)
		} else {
			status = "pending"
			retry := now.Add(time.Duration(1<<uint(attempts-1)) * time.Minute)
			_, err = tx.ExecContext(ctx, "UPDATE mail_queue SET attempts=?,available_at=?,error_code=? WHERE id=?", attempts, retry, code, row.ID)
		}
		return err
	})
	if err == nil && found {
		switch status {
		case "delivered":
			mailDelivered.Add(1)
		case "failed":
			mailFailed.Add(1)
			slog.Error("mail delivery failed permanently", "mail_id", id, "attempt", attempt, "code", code)
		case "pending":
			mailRetries.Add(1)
			slog.Warn("mail delivery retry scheduled", "mail_id", id, "attempt", attempt, "code", code)
		}
	}
	return found, err
}
func (w *MailWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for i := 0; i < 20; i++ {
				deliveryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
				found, err := w.DeliverOne(deliveryCtx, time.Now())
				cancel()
				if err != nil {
					if ctx.Err() == nil {
						slog.Error("mail queue processing failed")
					}
					break
				}
				if !found {
					break
				}
			}
		}
	}
}
