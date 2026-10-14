package storage

import (
	"context"
	"github.com/uptrace/bun"
	"time"
	"uptime_app/backend/internal/auth"
)

func (r *AuthRepository) UpdateProfile(ctx context.Context, uid, sid, name string, birthday *time.Time, now time.Time) (auth.User, error) {
	var user userRow
	err := r.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := lockUser(ctx, tx, uid); err != nil {
			return missing(err, auth.ErrUnauthorized)
		}
		var session sessionRow
		if err := tx.NewRaw("SELECT * FROM sessions WHERE id=? AND user_id=? AND revoked_at IS NULL AND expires_at>? FOR UPDATE", sid, uid, now).Scan(ctx, &session); err != nil {
			return missing(err, auth.ErrUnauthorized)
		}
		err := tx.NewRaw("UPDATE users SET full_name=?,date_of_birth=? WHERE id=? AND email_verified=true RETURNING *", name, birthday, uid).Scan(ctx, &user)
		return missing(err, auth.ErrUnauthorized)
	})
	return publicUser(user), err
}
