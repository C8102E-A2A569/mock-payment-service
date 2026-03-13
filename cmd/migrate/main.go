// Утилита для применения миграций без запуска сервера.
// Использование: go run ./cmd/migrate [ -down ]
package main

import (
	"log"
	"os"
	"path/filepath"

	"new-project/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "configs/config.yaml"
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = ""
	}
	cfg, err := config.Load(path)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	dsn := cfg.DB.DSN()

	dir := filepath.Join("internal", "repository", "postgres", "migrations")
	sourceURL := "file://" + filepath.ToSlash(dir)

	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		log.Fatalf("create migrator: %v", err)
	}
	defer m.Close()

	if len(os.Args) > 1 && os.Args[1] == "-down" {
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("migrate down: %v", err)
		}
		log.Println("migrations down applied")
		return
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migrate up: %v", err)
	}
	log.Println("migrations up applied")
}
