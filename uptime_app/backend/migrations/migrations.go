package migrations

import (
	"embed"
	"github.com/uptrace/bun/migrate"
)

//go:embed *.sql
var files embed.FS

func New() (*migrate.Migrations, error) {
	m := migrate.NewMigrations()
	if err := m.Discover(files); err != nil {
		return nil, err
	}
	return m, nil
}
