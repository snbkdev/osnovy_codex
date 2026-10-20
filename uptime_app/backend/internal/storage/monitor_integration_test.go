package storage

import (
	"context"
	"errors"
	"testing"

	"uptime_app/backend/internal/monitor"
)

func TestPostgresMonitors(t *testing.T) {
	db := testDB(t)
	authService := serviceFor(t, db)
	ctx := context.Background()
	for _, email := range []string{"monitor-owner@example.com", "monitor-other@example.com"} {
		if err := authService.Register(ctx, email, password); err != nil {
			t.Fatal(err)
		}
	}
	owner, err := authService.Repo.FindUser(ctx, "monitor-owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	other, err := authService.Repo.FindUser(ctx, "monitor-other@example.com")
	if err != nil {
		t.Fatal(err)
	}
	repo := &MonitorRepository{DB: db}
	service := &monitor.Service{Repo: repo}
	empty, err := service.List(ctx, owner.ID)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatal("invalid empty list", empty, err)
	}
	first, err := service.Create(ctx, owner.ID, "https://example.com", 5)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(ctx, owner.ID, "https://example.com/status", 420)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, other.ID, "https://other.example.com", 3600); err != nil {
		t.Fatal(err)
	}
	values, err := service.List(ctx, owner.ID)
	if err != nil || len(values) != 2 || values[0].ID != second.ID || values[1].ID != first.ID || values[0].IntervalSeconds != 420 || values[1].URL != "https://example.com/" {
		t.Fatal("persistence/order failed", values, err)
	}
	others, err := service.List(ctx, other.ID)
	if err != nil || len(others) != 1 || others[0].UserID != other.ID {
		t.Fatal("ownership isolation failed", others, err)
	}
	if _, err := repo.Update(ctx, other.ID, first.ID, "https://unwanted.example/", 60); !errors.Is(err, monitor.ErrNotFound) {
		t.Fatal("another user could update monitor", err)
	}
	if err := repo.Delete(ctx, other.ID, first.ID); !errors.Is(err, monitor.ErrNotFound) {
		t.Fatal("another user could delete monitor", err)
	}
	updated, err := repo.Update(ctx, owner.ID, first.ID, "https://updated.example/status", 3600)
	if err != nil || updated.ID != first.ID || updated.UserID != owner.ID || !updated.CreatedAt.Equal(first.CreatedAt) || updated.URL != "https://updated.example/status" || updated.IntervalSeconds != 3600 {
		t.Fatal("update failed or changed immutable fields", updated, err)
	}
	values, err = service.List(ctx, owner.ID)
	if err != nil || len(values) != 2 || values[1] != updated {
		t.Fatal("update was not persisted or changed ordering", values, err)
	}
	if _, err := repo.Update(ctx, owner.ID, "missing-monitor", "https://updated.example/", 60); !errors.Is(err, monitor.ErrNotFound) {
		t.Fatal("updating missing monitor did not return not found", err)
	}
	if err := repo.Delete(ctx, owner.ID, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, owner.ID, first.ID); !errors.Is(err, monitor.ErrNotFound) {
		t.Fatal("repeated deletion did not return not found", err)
	}
	values, err = service.List(ctx, owner.ID)
	if err != nil || len(values) != 1 || values[0].ID != second.ID {
		t.Fatal("deletion was not persisted", values, err)
	}
	if _, err := repo.Create(ctx, monitor.Monitor{ID: "invalid-interval", UserID: owner.ID, URL: "https://example.com", IntervalSeconds: 4, CreatedAt: first.CreatedAt}); err == nil {
		t.Fatal("database accepted invalid interval")
	}
	if _, err := repo.Create(ctx, monitor.Monitor{ID: "invalid-owner", UserID: "missing-user", URL: "https://example.com", IntervalSeconds: 5, CreatedAt: first.CreatedAt}); err == nil {
		t.Fatal("database accepted missing owner")
	}
	for i := 1; i < monitor.MaxMonitors; i++ {
		if _, err := service.Create(ctx, owner.ID, "https://quota.example", 60); err != nil {
			t.Fatal("could not fill monitor quota", err)
		}
	}
	if _, err := service.Create(ctx, owner.ID, "https://quota.example", 60); !errors.Is(err, monitor.ErrQuota) {
		t.Fatal("monitor quota was not enforced", err)
	}
	if err := repo.Delete(ctx, owner.ID, second.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, owner.ID, "https://replacement.example", 60); err != nil {
		t.Fatal("deletion did not free quota", err)
	}
	if _, err := db.NewRaw("DELETE FROM users WHERE id=?", owner.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	values, err = service.List(ctx, owner.ID)
	if err != nil || len(values) != 0 {
		t.Fatal("owner deletion did not cascade", values, err)
	}
	others, err = service.List(ctx, other.ID)
	if err != nil || len(others) != 1 {
		t.Fatal("owner deletion affected another user", others, err)
	}
}
