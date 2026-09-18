package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V6_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Web-push (FCM) device tokens for browser notifications.
	// Idempotent: safe to re-run on an already migrated DB.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS push_tokens (
			id         SERIAL PRIMARY KEY,
			uuid       UUID NOT NULL UNIQUE,
			token      TEXT NOT NULL UNIQUE,
			label      TEXT NOT NULL DEFAULT '',
			is_active  BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_push_tokens_active ON push_tokens(is_active);
	`); err != nil {
		return err
	}

	return nil
}
