package storage

import (
	"context"
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
	if _, err := repo.Create(ctx, monitor.Monitor{ID: "invalid-interval", UserID: owner.ID, URL: "https://example.com", IntervalSeconds: 4, CreatedAt: first.CreatedAt}); err == nil {
		t.Fatal("database accepted invalid interval")
	}
	if _, err := repo.Create(ctx, monitor.Monitor{ID: "invalid-owner", UserID: "missing-user", URL: "https://example.com", IntervalSeconds: 5, CreatedAt: first.CreatedAt}); err == nil {
		t.Fatal("database accepted missing owner")
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
