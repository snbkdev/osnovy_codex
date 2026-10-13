package storage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/uptrace/bun"
	"log/slog"
	"time"
	"uptime_app/backend/internal/auth"
	"uptime_app/backend/internal/mail"
)

type MailWorker struct {
	DB     *bun.DB
	Cipher *auth.Cipher
	Sender mail.Sender
}

// The transaction holds the selected row while sending. SMTP is at-least-once:
// a crash after SMTP acceptance and before commit can cause a duplicate letter.
func (w *MailWorker) DeliverOne(ctx context.Context, now time.Time) (bool, error) {
	found := false
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
		if deliveryErr == nil {
			deliveryErr = w.Sender.Send(ctx, letter)
		}
		attempts := row.Attempts + 1
		if deliveryErr == nil || attempts >= 5 {
			_, err = tx.ExecContext(ctx, "UPDATE mail_queue SET payload=NULL,finished_at=?,attempts=? WHERE id=?", now, attempts, row.ID)
		} else {
			retry := now.Add(time.Duration(1<<uint(attempts-1)) * time.Minute)
			_, err = tx.ExecContext(ctx, "UPDATE mail_queue SET attempts=?,available_at=? WHERE id=?", attempts, retry, row.ID)
		}
		return err
	})
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
