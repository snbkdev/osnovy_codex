package storage

import (
	"context"
	"sync"
	"testing"
	"time"
	"uptime_app/backend/internal/auth"
)

func TestPostgresRefreshLostResponse(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	service := serviceFor(t, db)
	if err := service.Register(ctx, "retry@example.com", password); err != nil {
		t.Fatal(err)
	}
	_, verify := lastLetter(t, db, service)
	if err := service.Verify(ctx, verify); err != nil {
		t.Fatal(err)
	}
	login, err := service.Login(ctx, "retry@example.com", password)
	if err != nil {
		t.Fatal(err)
	}
	attempt := auth.RandomToken()
	// Rotation commits, but the client retains the original cookie after losing the response.
	var first auth.LoginResult
	var wg sync.WaitGroup
	type refreshResult struct {
		value auth.LoginResult
		err   error
	}
	results := make(chan refreshResult, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := service.Refresh(ctx, login.RefreshToken, attempt)
			results <- refreshResult{value, err}
		}()
	}
	wg.Wait()
	close(results)
	for result := range results {
		if result.err != nil {
			t.Fatal("concurrent retry rejected", result.err)
		}
		if first.RefreshToken != "" && result.value.RefreshToken != first.RefreshToken {
			t.Fatal("concurrent retry changed successor")
		}
		first = result.value
	}
	// Retry through a separate service instance, as with multiple API replicas.
	other := serviceFor(t, db)
	replay, err := other.Refresh(ctx, login.RefreshToken, attempt)
	if err != nil || replay.RefreshToken != first.RefreshToken {
		t.Fatal("retry did not restore the same successor", err)
	}
	var count int
	if err := db.NewRaw("SELECT count(*) FROM refresh_tokens").Scan(ctx, &count); err != nil || count != 2 {
		t.Fatal("retry created another token", count, err)
	}
	if _, err := service.Me(ctx, replay.AccessToken); err != nil {
		t.Fatal("recovered session revoked", err)
	}
	// Once the successor is consumed, replaying its parent must revoke the family.
	next, err := service.Refresh(ctx, replay.RefreshToken, auth.RandomToken())
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Refresh(ctx, login.RefreshToken, attempt)
	requireError(t, err, auth.ErrUnauthorized)
	_, err = service.Refresh(ctx, next.RefreshToken, auth.RandomToken())
	requireError(t, err, auth.ErrUnauthorized)
}

func TestPostgresRefreshRetryBoundaries(t *testing.T) {
	for _, mode := range []string{"different-key", "expired-window", "logout", "reset"} {
		t.Run(mode, func(t *testing.T) {
			db := testDB(t)
			ctx := context.Background()
			service := serviceFor(t, db)
			if err := service.Register(ctx, "boundary@example.com", password); err != nil {
				t.Fatal(err)
			}
			_, verify := lastLetter(t, db, service)
			if err := service.Verify(ctx, verify); err != nil {
				t.Fatal(err)
			}
			login, err := service.Login(ctx, "boundary@example.com", password)
			if err != nil {
				t.Fatal(err)
			}
			attempt := auth.RandomToken()
			result, err := service.Refresh(ctx, login.RefreshToken, attempt)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "different-key":
				attempt = auth.RandomToken()
			case "expired-window":
				if _, err := db.ExecContext(ctx, "UPDATE refresh_tokens SET used_at=? WHERE hash=?", time.Now().Add(-auth.RefreshRetryWindow-time.Second), auth.TokenHash(login.RefreshToken)); err != nil {
					t.Fatal(err)
				}
			case "logout":
				if err := service.Logout(ctx, result.RefreshToken); err != nil {
					t.Fatal(err)
				}
			case "reset":
				if err := service.SendToken(ctx, "boundary@example.com", "reset"); err != nil {
					t.Fatal(err)
				}
				_, token := lastLetter(t, db, service)
				if err := service.Reset(ctx, token, "new long password"); err != nil {
					t.Fatal(err)
				}
			}
			_, err = service.Refresh(ctx, login.RefreshToken, attempt)
			requireError(t, err, auth.ErrUnauthorized)
			_, err = service.Me(ctx, result.AccessToken)
			requireError(t, err, auth.ErrUnauthorized)
		})
	}
}
