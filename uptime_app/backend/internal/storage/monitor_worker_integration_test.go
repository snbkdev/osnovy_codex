package storage

import (
	"github.com/uptrace/bun/migrate"
	"testing"
	"time"
	"uptime_app/backend/internal/monitor"
	"uptime_app/backend/migrations"
)

func TestPostgresMonitorChecks(t *testing.T) {
	db := testDB(t)
	ctx := t.Context()
	auth := serviceFor(t, db)
	if err := auth.Register(ctx, "checks@example.com", password); err != nil {
		t.Fatal(err)
	}
	user, err := auth.Repo.FindUser(ctx, "checks@example.com")
	if err != nil {
		t.Fatal(err)
	}
	repo := &MonitorRepository{DB: db}
	service := &monitor.Service{Repo: repo}
	created, err := service.Create(ctx, user.ID, "https://example.com", 5)
	if err != nil {
		t.Fatal(err)
	}
	row, err := repo.claimCheck(ctx)
	if err != nil || row.ID != created.ID {
		t.Fatal(row, err)
	}
	if _, err = repo.claimCheck(ctx); err == nil {
		t.Fatal("monitor claimed twice during lease")
	}
	result := monitor.CheckResult{Status: "success", HTTPStatus: 200, CheckedAt: time.Now().UTC()}
	if err = repo.finishCheck(ctx, row, result); err != nil {
		t.Fatal(err)
	}
	values, err := repo.List(ctx, user.ID)
	if err != nil || values[0].Status != "success" || values[0].HTTPStatus != 200 || values[0].CheckedAt == nil {
		t.Fatal(values, err)
	}
	if _, err = repo.claimCheck(ctx); err == nil {
		t.Fatal("interval ignored")
	}
	if _, err = repo.Update(ctx, user.ID, created.ID, "https://changed.example/", 5); err != nil {
		t.Fatal(err)
	}
	if err = repo.finishCheck(ctx, row, result); err != nil {
		t.Fatal(err)
	}
	values, err = repo.List(ctx, user.ID)
	if err != nil || values[0].Status != "pending" || values[0].CheckedAt != nil {
		t.Fatal("stale check overwrote edit", values, err)
	}
	row, err = repo.claimCheck(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.NewRaw("UPDATE monitors SET next_check_at=now()-interval '1 second' WHERE id=?", row.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	next, err := repo.claimCheck(ctx)
	if err != nil || next.CheckToken == row.CheckToken {
		t.Fatal("lease recovery failed", err)
	}
	if err = repo.finishCheck(ctx, row, result); err != nil {
		t.Fatal(err)
	}
	result.Status = "error"
	result.HTTPStatus = 503
	if err = repo.finishCheck(ctx, next, result); err != nil {
		t.Fatal(err)
	}
	values, err = repo.List(ctx, user.ID)
	if err != nil || values[0].HTTPStatus != 503 {
		t.Fatal(values, err)
	}
}

func TestPostgresMonitorCheckMigrationUpgrade(t *testing.T) {
	db := testDB(t)
	ctx := t.Context()
	collection, err := migrations.New()
	if err != nil {
		t.Fatal(err)
	}
	all := migrate.NewMigrator(db, collection)
	if _, err = all.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	previous := migrate.NewMigrations()
	for _, entry := range collection.Sorted() {
		if entry.Name < "20261021000007" {
			previous.Add(entry)
		}
	}
	if _, err = migrate.NewMigrator(db, previous).Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	auth := serviceFor(t, db)
	if err = auth.Register(ctx, "upgrade@example.com", password); err != nil {
		t.Fatal(err)
	}
	user, err := auth.Repo.FindUser(ctx, "upgrade@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.NewRaw("INSERT INTO monitors(id,user_id,url,interval_seconds) VALUES ('before-upgrade',?,'https://example.com/',5)", user.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = all.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	repo := &MonitorRepository{DB: db}
	values, err := repo.List(ctx, user.ID)
	if err != nil || len(values) != 1 || values[0].Status != "pending" {
		t.Fatal(values, err)
	}
	if _, err = repo.claimCheck(ctx); err != nil {
		t.Fatal("existing monitor not scheduled", err)
	}
	if _, err = all.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.NewRaw("SELECT count(*) FROM monitors WHERE id='before-upgrade'").Scan(ctx, &count); err != nil || count != 1 {
		t.Fatal("rollback lost monitor", err)
	}
	if err = db.NewRaw("SELECT count(*) FROM information_schema.columns WHERE table_name='monitors' AND column_name='checked_at'").Scan(ctx, &count); err != nil || count != 0 {
		t.Fatal("rollback retained column", err)
	}
	if _, err = all.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	values, err = repo.List(ctx, user.ID)
	if err != nil || values[0].Status != "pending" {
		t.Fatal(values, err)
	}
}

func TestPostgresMonitorHistory(t *testing.T) {
	db := testDB(t)
	ctx := t.Context()
	auth := serviceFor(t, db)
	if err := auth.Register(ctx, "history@example.com", password); err != nil {
		t.Fatal(err)
	}
	user, err := auth.Repo.FindUser(ctx, "history@example.com")
	if err != nil {
		t.Fatal(err)
	}
	repo := &MonitorRepository{DB: db}
	service := &monitor.Service{Repo: repo}
	created, err := service.Create(ctx, user.ID, "https://example.com", 5)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := service.History(ctx, user.ID, created.ID)
	if err != nil || empty.Buckets == nil || len(empty.Buckets) != 0 {
		t.Fatal(empty, err)
	}
	if _, err = service.History(ctx, "other", created.ID); err != monitor.ErrNotFound {
		t.Fatal("foreign monitor exposed", err)
	}
	if _, err = service.History(ctx, user.ID, "missing"); err != monitor.ErrNotFound {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	row, err := repo.claimCheck(ctx)
	if err != nil {
		t.Fatal(err)
	}
	result := monitor.CheckResult{Status: "success", HTTPStatus: 200, CheckedAt: now.Add(-time.Minute)}
	if err = repo.finishCheck(ctx, row, result); err != nil {
		t.Fatal(err)
	}
	// Replayed/stale completion must not duplicate history.
	if err = repo.finishCheck(ctx, row, result); err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct {
		when   time.Time
		status string
		code   int
	}{{now.Add(-time.Minute), "error", 503}, {from.Add(-time.Second), "success", 200}, {now.Add(time.Hour), "error", 500}} {
		if _, err = db.NewRaw("INSERT INTO monitor_checks(monitor_id,url,checked_at,status,http_status) VALUES (?,?,?,?,?)", created.ID, created.URL, sample.when, sample.status, sample.code).Exec(ctx); err != nil {
			t.Fatal(err)
		}
	}
	history, err := repo.History(ctx, user.ID, created.ID, from, now)
	if err != nil || len(history.Buckets) != 1 || history.Buckets[0].Success != 1 || history.Buckets[0].Failure != 1 {
		t.Fatal(history, err)
	}
	if _, err = repo.Update(ctx, user.ID, created.ID, "https://changed.example/", 5); err != nil {
		t.Fatal(err)
	}
	history, err = service.History(ctx, user.ID, created.ID)
	if err != nil || len(history.Buckets) != 0 {
		t.Fatal("different URL history mixed", history, err)
	}
	if err = repo.finishCheck(ctx, row, result); err != nil {
		t.Fatal(err)
	}
	if err = repo.Delete(ctx, user.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.NewRaw("SELECT count(*) FROM monitor_checks WHERE monitor_id=?", created.ID).Scan(ctx, &count); err != nil || count != 0 {
		t.Fatal("history not deleted", count, err)
	}
}

func TestPostgresHistoryMigrationUpgrade(t *testing.T) {
	db := testDB(t)
	ctx := t.Context()
	collection, err := migrations.New()
	if err != nil {
		t.Fatal(err)
	}
	all := migrate.NewMigrator(db, collection)
	if _, err = all.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	previous := migrate.NewMigrations()
	for _, entry := range collection.Sorted() {
		if entry.Name < "20261021000008" {
			previous.Add(entry)
		}
	}
	if _, err = migrate.NewMigrator(db, previous).Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	auth := serviceFor(t, db)
	if err = auth.Register(ctx, "history-upgrade@example.com", password); err != nil {
		t.Fatal(err)
	}
	user, err := auth.Repo.FindUser(ctx, "history-upgrade@example.com")
	if err != nil {
		t.Fatal(err)
	}
	repo := &MonitorRepository{DB: db}
	service := &monitor.Service{Repo: repo}
	created, err := service.Create(ctx, user.ID, "https://example.com", 5)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.NewRaw("UPDATE monitors SET status='success',http_status=200,checked_at=now() WHERE id=?", created.ID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = all.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	history, err := service.History(ctx, user.ID, created.ID)
	if err != nil || len(history.Buckets) != 1 || history.Buckets[0].Success != 1 {
		t.Fatal("last known sample not preserved", history, err)
	}
	if _, err = all.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.NewRaw("SELECT count(*) FROM monitors WHERE id=?", created.ID).Scan(ctx, &count); err != nil || count != 1 {
		t.Fatal("rollback lost monitor", err)
	}
	if err = db.NewRaw("SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='monitor_checks'").Scan(ctx, &count); err != nil || count != 0 {
		t.Fatal("rollback did not drop history", err)
	}
	if _, err = all.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	history, err = service.History(ctx, user.ID, created.ID)
	if err != nil || len(history.Buckets) != 1 {
		t.Fatal(history, err)
	}
}
