package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"hotspot/backend-go/internal/platform/config"
	platformmigrate "hotspot/backend-go/internal/platform/migrate"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/migrations"
)

// cmd/migrate 是迁移的唯一显式入口（与启动期 EnsureSchema 同实现）：
//
//	go run ./cmd/migrate                                   # 应用 pending 迁移（幂等）
//	go run ./cmd/migrate -verify-only                      # 只读校验：checksum + pending + schema manifest
//	go run ./cmd/migrate -write-manifest=migrations/schema_manifest.json  # 在全新库上生成期望 schema 快照
func main() {
	var (
		verifyOnly    = flag.Bool("verify-only", false, "only verify applied-migration checksums and the schema manifest (read-only)")
		writeManifest = flag.String("write-manifest", "", "write the schema manifest of the current database to this path (run against a freshly migrated database)")
	)
	flag.Parse()

	cfg := config.Load()
	if cfg.PostgresDSN == "" {
		log.Fatal("POSTGRES_DSN is required")
	}

	db := persistence.NewPostgresDB(cfg.PostgresDSN)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	defer db.Close()

	switch {
	case *writeManifest != "":
		if err := platformmigrate.EnsureSchemaForManifest(ctx, db, migrations.Files); err != nil {
			log.Fatalf("ensure schema: %v", err)
		}
		manifest, err := platformmigrate.BuildSchemaManifest(ctx, db)
		if err != nil {
			log.Fatalf("build schema manifest: %v", err)
		}
		blob, err := manifest.Encode()
		if err != nil {
			log.Fatalf("encode schema manifest: %v", err)
		}
		if err := os.WriteFile(*writeManifest, blob, 0o644); err != nil {
			log.Fatalf("write schema manifest: %v", err)
		}
		log.Printf("schema manifest written to %s (tables=%d columns=%d indexes=%d)",
			*writeManifest, len(manifest.Tables), len(manifest.Columns), len(manifest.Indexes))
	case *verifyOnly:
		if err := platformmigrate.VerifyOnly(ctx, db, migrations.Files); err != nil {
			log.Fatalf("verify schema: %v", err)
		}
		log.Printf("schema verification passed")
	default:
		if err := platformmigrate.EnsureSchema(ctx, db, migrations.Files); err != nil {
			log.Fatalf("ensure schema: %v", err)
		}
	}
}
