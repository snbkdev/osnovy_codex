package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"
	"uptime_app/backend/internal/auth"
)

type AuthRepository struct{ DB *bun.DB }

var _ auth.Repository = (*AuthRepository)(nil)

func publicUser(u userRow) auth.User {
	return auth.User{ID: u.ID, Email: u.Email, PasswordHash: u.PasswordHash, EmailVerified: u.EmailVerified, FullName: u.FullName, AvatarURL: u.AvatarURL, DateOfBirth: u.DateOfBirth}
}
func missing(err, replacement error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return replacement
	}
	return err
}
func lockUser(ctx context.Context, tx bun.Tx, id string) (userRow, error) {
	var u userRow
	err := tx.NewRaw("SELECT * FROM users WHERE id = ? FOR UPDATE", id).Scan(ctx, &u)
	return u, err
}
func enqueue(ctx context.Context, tx bun.Tx, uid, purpose, hash string, expires time.Time, payload []byte) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM one_time_tokens WHERE user_id = ? AND purpose = ?", uid, purpose); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO one_time_tokens (hash,user_id,purpose,expires_at) VALUES (?,?,?,?)", hash, uid, purpose, expires); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO mail_queue (payload) VALUES (?)", payload)
	return err
}
func (r *AuthRepository) Register(ctx context.Context, user auth.User, hash string, expires time.Time, payload []byte) error {
	return r.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		result, err := tx.ExecContext(ctx, "INSERT INTO users (id,email,password_hash) VALUES (?,?,?) ON CONFLICT (email) DO NOTHING", user.ID, user.Email, user.PasswordHash)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil || n == 0 {
			return err
		}
		return enqueue(ctx, tx, user.ID, "verify", hash, expires, payload)
	})
}
func (r *AuthRepository) Issue(ctx context.Context, email, purpose, hash string, expires time.Time, payload []byte) error {
	return r.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var u userRow
		err := tx.NewRaw("SELECT * FROM users WHERE email = ? FOR UPDATE", email).Scan(ctx, &u)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if (purpose == "verify" && u.EmailVerified) || (purpose == "reset" && !u.EmailVerified) {
			return nil
		}
		return enqueue(ctx, tx, u.ID, purpose, hash, expires, payload)
	})
}
func (r *AuthRepository) FindUser(ctx context.Context, email string) (auth.User, error) {
	var u userRow
	err := r.DB.NewRaw("SELECT * FROM users WHERE email = ?", email).Scan(ctx, &u)
	return publicUser(u), missing(err, auth.ErrCredentials)
}

// All session/token mutations lock the user first, then the session/token.
// A password reset serializes with both login and refresh on that user row.
func (r *AuthRepository) CreateSession(ctx context.Context, user auth.User, s auth.Session, hash string, now time.Time) error {
	return r.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		u, err := lockUser(ctx, tx, user.ID)
		if err != nil {
			return err
		}
		if u.PasswordHash != user.PasswordHash || !u.EmailVerified {
			return auth.ErrCredentials
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO sessions (id,user_id,expires_at,created_at) VALUES (?,?,?,?)", s.ID, s.UserID, s.ExpiresAt, now); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO refresh_tokens (hash,session_id) VALUES (?,?)", hash, s.ID)
		return err
	})
}
func (r *AuthRepository) Rotate(ctx context.Context, oldHash, nextHash string, now time.Time) (auth.User, auth.Session, error) {
	var u userRow
	var s sessionRow
	denied := false
	err := r.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var found sessionRow
		if err := tx.NewRaw("SELECT s.* FROM sessions s JOIN refresh_tokens t ON t.session_id=s.id WHERE t.hash=?", oldHash).Scan(ctx, &found); err != nil {
			return missing(err, auth.ErrUnauthorized)
		}
		var err error
		u, err = lockUser(ctx, tx, found.UserID)
		if err != nil {
			return err
		}
		if err := tx.NewRaw("SELECT * FROM sessions WHERE id=? FOR UPDATE", found.ID).Scan(ctx, &s); err != nil {
			return err
		}
		if s.RevokedAt != nil || !s.ExpiresAt.After(now) {
			return auth.ErrUnauthorized
		}
		var token refreshRow
		if err := tx.NewRaw("SELECT * FROM refresh_tokens WHERE hash=? FOR UPDATE", oldHash).Scan(ctx, &token); err != nil {
			return missing(err, auth.ErrUnauthorized)
		}
		if token.UsedAt != nil {
			denied = true
			_, err := tx.ExecContext(ctx, "UPDATE sessions SET revoked_at=? WHERE id=?", now, s.ID)
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE refresh_tokens SET used_at=? WHERE hash=?", now, oldHash); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO refresh_tokens (hash,session_id) VALUES (?,?)", nextHash, s.ID)
		return err
	})
	if err == nil && denied {
		err = auth.ErrUnauthorized
	}
	return publicUser(u), auth.Session{ID: s.ID, UserID: s.UserID, ExpiresAt: s.ExpiresAt}, err
}
func (r *AuthRepository) Revoke(ctx context.Context, hash string, now time.Time) error {
	_, err := r.DB.ExecContext(ctx, "UPDATE sessions SET revoked_at=? WHERE id=(SELECT session_id FROM refresh_tokens WHERE hash=?) AND revoked_at IS NULL", now, hash)
	return err
}
func (r *AuthRepository) consume(ctx context.Context, hash, purpose, password string, now time.Time) error {
	return r.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var initial tokenRow
		if err := tx.NewRaw("SELECT * FROM one_time_tokens WHERE hash=? AND purpose=?", hash, purpose).Scan(ctx, &initial); err != nil {
			return missing(err, auth.ErrInvalid)
		}
		if _, err := lockUser(ctx, tx, initial.UserID); err != nil {
			return err
		}
		var token tokenRow
		if err := tx.NewRaw("SELECT * FROM one_time_tokens WHERE hash=? AND purpose=? FOR UPDATE", hash, purpose).Scan(ctx, &token); err != nil {
			return missing(err, auth.ErrInvalid)
		}
		if !token.ExpiresAt.After(now) {
			return auth.ErrInvalid
		}
		if purpose == "verify" {
			if _, err := tx.ExecContext(ctx, "UPDATE users SET email_verified=true WHERE id=?", token.UserID); err != nil {
				return err
			}
		} else {
			if _, err := tx.ExecContext(ctx, "UPDATE users SET password_hash=? WHERE id=?", password, token.UserID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE sessions SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL", now, token.UserID); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, "DELETE FROM one_time_tokens WHERE hash=?", hash)
		return err
	})
}
func (r *AuthRepository) Verify(ctx context.Context, hash string, now time.Time) error {
	return r.consume(ctx, hash, "verify", "", now)
}
func (r *AuthRepository) Reset(ctx context.Context, hash, password string, now time.Time) error {
	return r.consume(ctx, hash, "reset", password, now)
}
func (r *AuthRepository) Current(ctx context.Context, uid, sid string, now time.Time) (auth.User, error) {
	var u userRow
	err := r.DB.NewRaw("SELECT u.* FROM users u JOIN sessions s ON s.user_id=u.id WHERE u.id=? AND s.id=? AND s.revoked_at IS NULL AND s.expires_at>? AND u.email_verified=true", uid, sid, now).Scan(ctx, &u)
	return publicUser(u), missing(err, auth.ErrUnauthorized)
}
