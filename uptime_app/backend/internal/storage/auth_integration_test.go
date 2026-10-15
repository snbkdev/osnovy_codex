package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
	"uptime_app/backend/internal/auth"
	"uptime_app/backend/internal/config"
	"uptime_app/backend/migrations"
)

func testDB(t *testing.T) *bun.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a dedicated database ending in _test")
	}
	u, err := url.Parse(dsn)
	if err != nil || !strings.HasSuffix(u.Path, "_test") {
		t.Fatal("TEST_DATABASE_URL must name a dedicated database ending in _test")
	}
	if dsn == os.Getenv("DATABASE_URL") {
		t.Fatal("test database must differ from DATABASE_URL")
	}
	ctx := context.Background()
	db, err := Open(ctx, config.Config{DatabaseURL: dsn, MaxOpen: 10, MaxIdle: 5, ConnLifetime: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	// Only an explicitly named test database may be reset by this suite.
	if _, err := db.ExecContext(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatal(err)
	}
	collection, err := migrations.New()
	if err != nil {
		t.Fatal(err)
	}
	m := migrate.NewMigrator(db, collection)
	if err := m.Init(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Lock(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Unlock(ctx); err != nil {
		t.Fatal(err)
	}
	return db
}
func serviceFor(t *testing.T, db *bun.DB) *auth.Service {
	t.Helper()
	c, err := auth.NewCipher(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}
	s, err := auth.NewService(&AuthRepository{db}, auth.Signer{Key: bytes.Repeat([]byte{8}, 32), Issuer: "test", Audience: "test"}, c, "http://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func lastLetter(t *testing.T, db *bun.DB, s *auth.Service) (auth.Letter, string) {
	t.Helper()
	var row mailRow
	if err := db.NewRaw("SELECT * FROM mail_queue ORDER BY id DESC LIMIT 1").Scan(context.Background(), &row); err != nil {
		t.Fatal(err)
	}
	letter, err := s.Cipher.Decrypt(row.Payload)
	if err != nil {
		t.Fatal(err)
	}
	marker := "#token="
	idx := strings.Index(letter.Body, marker)
	if idx < 0 {
		t.Fatal("fragment token absent")
	}
	token := strings.Split(letter.Body[idx+len(marker):], "\n")[0]
	return letter, token
}
func requireError(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

const password = "  Correct password  "

func TestPostgresAuthLifecycle(t *testing.T) {
	db := testDB(t)
	s := serviceFor(t, db)
	ctx := context.Background()
	if err := s.Register(ctx, "  USER@Example.com ", password); err != nil {
		t.Fatal(err)
	}
	letter, first := lastLetter(t, db, s)
	if letter.To != "user@example.com" {
		t.Fatal("email not normalized")
	}
	if err := s.Register(ctx, "user@example.com", "another password"); err != nil {
		t.Fatal(err)
	}
	_, err := s.Login(ctx, "user@example.com", "another password")
	requireError(t, err, auth.ErrCredentials)
	_, err = s.Login(ctx, "user@example.com", password)
	requireError(t, err, auth.ErrUnverified)
	if err := s.SendToken(ctx, "user@example.com", "verify"); err != nil {
		t.Fatal(err)
	}
	_, token := lastLetter(t, db, s)
	requireError(t, s.Verify(ctx, first), auth.ErrInvalid)
	if err := s.Verify(ctx, token); err != nil {
		t.Fatal(err)
	}
	requireError(t, s.Verify(ctx, token), auth.ErrInvalid)
	_, err = s.Login(ctx, "missing@example.com", password)
	requireError(t, err, auth.ErrCredentials)
	_, err = s.Login(ctx, "user@example.com", strings.TrimSpace(password))
	requireError(t, err, auth.ErrCredentials)
	result, err := s.Login(ctx, "user@example.com", password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Me(ctx, result.AccessToken); err != nil {
		t.Fatal(err)
	}
	rotated, err := s.Refresh(ctx, result.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.RefreshToken == result.RefreshToken {
		t.Fatal("no rotation")
	}
	_, err = s.Refresh(ctx, result.RefreshToken)
	requireError(t, err, auth.ErrUnauthorized)
	_, err = s.Me(ctx, rotated.AccessToken)
	requireError(t, err, auth.ErrUnauthorized)
	_, err = s.Refresh(ctx, rotated.RefreshToken)
	requireError(t, err, auth.ErrUnauthorized)
	a, err := s.Login(ctx, "user@example.com", password)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Logout(ctx, a.RefreshToken); err != nil {
		t.Fatal(err)
	}
	_, err = s.Me(ctx, a.AccessToken)
	requireError(t, err, auth.ErrUnauthorized)
	a, _ = s.Login(ctx, "user@example.com", password)
	b, _ := s.Login(ctx, "user@example.com", password)
	if err := s.SendToken(ctx, "user@example.com", "reset"); err != nil {
		t.Fatal(err)
	}
	_, reset1 := lastLetter(t, db, s)
	if err := s.SendToken(ctx, "user@example.com", "reset"); err != nil {
		t.Fatal(err)
	}
	_, reset := lastLetter(t, db, s)
	requireError(t, s.Reset(ctx, reset1, "new long password"), auth.ErrInvalid)
	if err := s.Reset(ctx, reset, "new long password"); err != nil {
		t.Fatal(err)
	}
	requireError(t, s.Reset(ctx, reset, "another long password"), auth.ErrInvalid)
	for _, access := range []string{a.AccessToken, b.AccessToken} {
		_, err := s.Me(ctx, access)
		requireError(t, err, auth.ErrUnauthorized)
	}
	_, err = s.Login(ctx, "user@example.com", password)
	requireError(t, err, auth.ErrCredentials)
	if _, err := s.Login(ctx, "user@example.com", "new long password"); err != nil {
		t.Fatal(err)
	}
	for _, purpose := range []string{"verify", "reset"} {
		if err := s.SendToken(ctx, "absent@example.com", purpose); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	db.NewRaw("SELECT count(*) FROM users").Scan(ctx, &count)
	if count != 1 {
		t.Fatalf("users %d", count)
	}
	// One-time table holds hashes, not the raw email tokens.
	db.NewRaw("SELECT count(*) FROM one_time_tokens WHERE hash IN (?,?)", token, reset).Scan(ctx, &count)
	if count != 0 {
		t.Fatal("plaintext token stored")
	}
}
func TestPostgresExpiryAndAtomicity(t *testing.T) {
	db := testDB(t)
	s := serviceFor(t, db)
	ctx := context.Background()
	repo := s.Repo.(*AuthRepository)
	if err := s.Register(ctx, "expire@example.com", password); err != nil {
		t.Fatal(err)
	}
	_, verify := lastLetter(t, db, s)
	db.ExecContext(ctx, "UPDATE one_time_tokens SET expires_at=?", time.Now().Add(-time.Minute))
	requireError(t, s.Verify(ctx, verify), auth.ErrInvalid)
	s.SendToken(ctx, "expire@example.com", "verify")
	_, verify = lastLetter(t, db, s)
	if err := s.Verify(ctx, verify); err != nil {
		t.Fatal(err)
	}
	result, err := s.Login(ctx, "expire@example.com", password)
	if err != nil {
		t.Fatal(err)
	}
	var oldExpires time.Time
	db.NewRaw("SELECT expires_at FROM sessions LIMIT 1").Scan(ctx, &oldExpires)
	next, err := s.Refresh(ctx, result.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	var newExpires time.Time
	db.NewRaw("SELECT expires_at FROM sessions LIMIT 1").Scan(ctx, &newExpires)
	if !oldExpires.Equal(newExpires) {
		t.Fatal("absolute expiry extended")
	}
	_, _, err = repo.Rotate(ctx, auth.TokenHash(next.RefreshToken), auth.TokenHash(auth.RandomToken()), oldExpires)
	requireError(t, err, auth.ErrUnauthorized)
	s.SendToken(ctx, "expire@example.com", "reset")
	_, reset := lastLetter(t, db, s)
	db.ExecContext(ctx, "UPDATE one_time_tokens SET expires_at=?", time.Now().Add(-time.Minute))
	requireError(t, s.Reset(ctx, reset, "new password long"), auth.ErrInvalid)
	// An invalid queue payload makes the entire registration/token transaction fail.
	err = repo.Register(ctx, auth.User{ID: auth.RandomToken(), Email: "rollback@example.com", PasswordHash: "hash"}, auth.TokenHash("rollback"), time.Now().Add(time.Hour), nil)
	if err == nil {
		t.Fatal("expected queue constraint failure")
	}
	_, err = repo.FindUser(ctx, "rollback@example.com")
	requireError(t, err, auth.ErrCredentials)
	// A failed reissue must preserve the previously issued token.
	s.SendToken(ctx, "expire@example.com", "reset")
	_, reset = lastLetter(t, db, s)
	err = repo.Issue(ctx, "expire@example.com", "reset", auth.TokenHash("replacement"), time.Now().Add(time.Hour), nil)
	if err == nil {
		t.Fatal("expected failed issue")
	}
	if err := s.Reset(ctx, reset, "new password long"); err != nil {
		t.Fatal("old token lost on rollback", err)
	}
	// Foreign key, email uniqueness and purpose constraints.
	for _, q := range []string{"INSERT INTO sessions(id,user_id,expires_at) VALUES ('s','missing',now()+interval '1 day')", "INSERT INTO users(id,email,password_hash) VALUES ('u','expire@example.com','hash')", "INSERT INTO users(id,email,password_hash) VALUES ('u','UPPER@example.com','hash')", "INSERT INTO one_time_tokens(hash,user_id,purpose,expires_at) SELECT 'h',id,'wrong',now() FROM users LIMIT 1"} {
		if _, err := db.ExecContext(ctx, q); err == nil {
			t.Fatal("constraint missing", q)
		}
	}
}
func TestPostgresConcurrentTokens(t *testing.T) {
	db := testDB(t)
	s := serviceFor(t, db)
	ctx := context.Background()
	var wg sync.WaitGroup
	registrationErrors := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Go(func() { registrationErrors <- s.Register(ctx, "race@example.com", password) })
	}
	wg.Wait()
	close(registrationErrors)
	for err := range registrationErrors {
		if err != nil && !errors.Is(err, auth.ErrBusy) {
			t.Fatal(err)
		}
	}
	var count int
	db.NewRaw("SELECT count(*) FROM users").Scan(ctx, &count)
	if count != 1 {
		t.Fatal("duplicate users")
	}
	db.NewRaw("SELECT count(*) FROM mail_queue").Scan(ctx, &count)
	if count != 1 {
		t.Fatal("duplicate registration emails")
	}
	_, verify := lastLetter(t, db, s)
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Go(func() { results <- s.Verify(ctx, verify) })
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			requireError(t, err, auth.ErrInvalid)
		}
	}
	if successes != 1 {
		t.Fatal("token consumed twice")
	}
	result, err := s.Login(ctx, "race@example.com", password)
	if err != nil {
		t.Fatal(err)
	}
	refreshResults := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Go(func() { _, err := s.Refresh(ctx, result.RefreshToken); refreshResults <- err })
	}
	wg.Wait()
	close(refreshResults)
	successes = 0
	for err := range refreshResults {
		if err == nil {
			successes++
		} else {
			requireError(t, err, auth.ErrUnauthorized)
		}
	}
	if successes != 1 {
		t.Fatal("refresh rotated twice")
	}
	_, err = s.Me(ctx, result.AccessToken)
	requireError(t, err, auth.ErrUnauthorized)
	// Repeat reset vs refresh + login to exercise both lock orderings.
	for i := 0; i < 10; i++ {
		user, _ := s.Repo.FindUser(ctx, "race@example.com")
		currentPassword := password
		if i > 0 {
			currentPassword = fmt.Sprintf("new password %02d", i-1)
		}
		result, err = s.Login(ctx, user.Email, currentPassword)
		if err != nil {
			t.Fatal(err)
		}
		s.SendToken(ctx, user.Email, "reset")
		_, reset := lastLetter(t, db, s)
		start := make(chan struct{})
		errs := make(chan error, 3)
		var loginResult auth.LoginResult
		wg.Go(func() {
			<-start
			_, err := s.Refresh(ctx, result.RefreshToken)
			if errors.Is(err, auth.ErrUnauthorized) {
				err = nil
			}
			errs <- err
		})
		wg.Go(func() { <-start; errs <- s.Reset(ctx, reset, fmt.Sprintf("new password %02d", i)) })
		wg.Go(func() {
			<-start
			var err error
			loginResult, err = s.Login(ctx, user.Email, currentPassword)
			if errors.Is(err, auth.ErrCredentials) {
				err = nil
			}
			errs <- err
		})
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		db.NewRaw("SELECT count(*) FROM sessions WHERE revoked_at IS NULL AND user_id=?", user.ID).Scan(ctx, &count)
		if count != 0 {
			t.Fatalf("active session survived reset at iteration %d", i)
		}
		if loginResult.AccessToken != "" {
			_, err := s.Me(ctx, loginResult.AccessToken)
			requireError(t, err, auth.ErrUnauthorized)
		}
	}
}
func TestPostgresMigrations(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	collection, _ := migrations.New()
	m := migrate.NewMigrator(db, collection)
	if err := m.Lock(ctx); err != nil {
		t.Fatal(err)
	}
	defer m.Unlock(ctx)
	if err := migrate.NewMigrator(db, collection).Lock(ctx); err == nil {
		t.Fatal("migration lock not enforced")
	}
	group, err := m.Migrate(ctx)
	if err != nil || !group.IsZero() {
		t.Fatalf("repeat up: %+v %v", group, err)
	}
	list, err := m.MigrationsWithStatus(ctx)
	if err != nil || len(list) != 6 {
		t.Fatal("invalid status", err)
	}
	for _, migration := range list {
		if !migration.IsApplied() {
			t.Fatal("unapplied migration", migration.Name)
		}
	}
	if _, err := m.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	db.NewRaw("SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='users'").Scan(ctx, &count)
	if count != 0 {
		t.Fatal("down did not remove schema")
	}
	if _, err := m.Migrate(ctx); err != nil {
		t.Fatal("up after down", err)
	}
}

type testSender struct {
	Fail    bool
	Letters []auth.Letter
}

func (s *testSender) Send(_ context.Context, l auth.Letter) error {
	s.Letters = append(s.Letters, l)
	if s.Fail {
		return errors.New("SMTP failed")
	}
	return nil
}
func TestPostgresMailDelivery(t *testing.T) {
	db := testDB(t)
	s := serviceFor(t, db)
	ctx := context.Background()
	if err := s.Register(ctx, "mail@example.com", password); err != nil {
		t.Fatal(err)
	}
	sender := &testSender{Fail: true}
	worker := MailWorker{db, s.Cipher, sender}
	now := time.Now().Add(time.Second)
	for i := 1; i <= 5; i++ {
		found, err := worker.DeliverOne(ctx, now)
		if err != nil || !found {
			t.Fatalf("attempt %d %v %v", i, found, err)
		}
		var row mailRow
		db.NewRaw("SELECT * FROM mail_queue LIMIT 1").Scan(ctx, &row)
		if row.Attempts != i {
			t.Fatal("attempt count")
		}
		if i < 5 {
			if len(row.Payload) == 0 || !row.AvailableAt.After(now) {
				t.Fatal("retry not scheduled")
			}
			if found, err := worker.DeliverOne(ctx, now); err != nil || found {
				t.Fatal("retry ignored delay")
			}
			now = row.AvailableAt.Add(time.Second)
		} else if len(row.Payload) != 0 || row.FinishedAt == nil || row.Status != "failed" || row.ErrorCode == nil || *row.ErrorCode != "smtp_error" {
			t.Fatal("payload retained after terminal failure")
		}
	}
	if found, err := worker.DeliverOne(ctx, now.Add(time.Hour)); err != nil || found {
		t.Fatal("sixth attempt")
	}
	s.SendToken(ctx, "mail@example.com", "verify")
	sender.Fail = false
	if found, err := worker.DeliverOne(ctx, time.Now().Add(time.Second)); err != nil || !found {
		t.Fatal("successful send failed", err)
	}
	var count int
	db.NewRaw("SELECT count(*) FROM mail_queue WHERE payload IS NOT NULL OR status NOT IN ('failed','delivered')").Scan(ctx, &count)
	if count != 0 {
		t.Fatal("sent payload retained")
	}
	if len(sender.Letters) != 6 || sender.Letters[5].To != "mail@example.com" {
		t.Fatal("invalid sender input")
	}
}
