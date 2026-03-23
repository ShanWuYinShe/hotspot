package main

import (
	"context"
	"log"
	"time"

	"hotspot/backend-go/internal/platform/config"
	platformmigrate "hotspot/backend-go/internal/platform/migrate"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/migrations"
)

func main() {
	cfg := config.Load()
	if cfg.PostgresDSN == "" {
		log.Fatal("POSTGRES_DSN is required")
	}

	db := persistence.NewPostgresDB(cfg.PostgresDSN)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer db.Close()

	if err := platformmigrate.EnsureSchema(ctx, db, migrations.Files); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}
}
