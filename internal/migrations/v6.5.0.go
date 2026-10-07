package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V6_5_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Per-list custom double opt-in confirmation e-mail template. An empty
	// value means the built-in system template (subscriber-optin) is used.
	// Idempotent: safe to re-run on an already migrated DB.
	if _, err := db.Exec(`
		ALTER TABLE lists
		ADD COLUMN IF NOT EXISTS optin_template TEXT NOT NULL DEFAULT '';
	`); err != nil {
		return err
	}

	return nil
}
