package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V6_4_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Per-list confirm-webhook targets for external lifecycle integrations
	// (e.g. notify another app when a subscriber confirms). Empty values
	// mean "no webhook for this list".
	// Idempotent: safe to re-run on an already migrated DB.
	if _, err := db.Exec(`
		ALTER TABLE lists
		ADD COLUMN IF NOT EXISTS webhook_url TEXT NOT NULL DEFAULT '',
		ADD COLUMN IF NOT EXISTS webhook_secret TEXT NOT NULL DEFAULT '';
	`); err != nil {
		return err
	}

	return nil
}
