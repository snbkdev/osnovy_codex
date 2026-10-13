package main

import (
	"context"
	"fmt"
	"github.com/uptrace/bun/migrate"
	"os"
	"time"
	"uptime_app/backend/internal/config"
	"uptime_app/backend/internal/storage"
	"uptime_app/backend/migrations"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "down" && os.Args[1] != "status") {
		return fmt.Errorf("usage: migrate up|status|down")
	}
	c, err := config.LoadDatabase()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := storage.Open(ctx, c)
	if err != nil {
		return err
	}
	defer db.Close()
	collection, err := migrations.New()
	if err != nil {
		return err
	}
	m := migrate.NewMigrator(db, collection)
	if err := m.Init(ctx); err != nil {
		return err
	}
	if err := m.Lock(ctx); err != nil {
		return err
	}
	defer func() {
		unlockCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := m.Unlock(unlockCtx); err != nil {
			fmt.Fprintln(os.Stderr, "migration unlock failed")
		}
	}()
	switch os.Args[1] {
	case "up":
		group, err := m.Migrate(ctx)
		if err != nil {
			return err
		}
		fmt.Println(group)
	case "down":
		group, err := m.Rollback(ctx)
		if err != nil {
			return err
		}
		fmt.Println(group)
	case "status":
		list, err := m.MigrationsWithStatus(ctx)
		if err != nil {
			return err
		}
		for _, item := range list {
			fmt.Printf("%s applied=%t\n", item.Name, item.IsApplied())
		}
	}
	return nil
}
