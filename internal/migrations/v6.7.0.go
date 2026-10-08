package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V6_7_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Dedicated template type for double opt-in confirmation e-mails, so they
	// can be previewed and rendered with the opt-in data context (.Lists,
	// .OptinURL, .UnsubURL) instead of the transactional {Subscriber, Tx} one.
	// Idempotent: safe to re-run on an already migrated DB.
	//
	// NOTE: ALTER TYPE ... ADD VALUE runs in its own implicit transaction; the
	// new value is only usable in a subsequent statement.
	if _, err := db.Exec(`
		ALTER TYPE template_type ADD VALUE IF NOT EXISTS 'optin';
	`); err != nil {
		return err
	}

	// Retype the templates that are actually wired as list opt-in e-mails.
	if _, err := db.Exec(`
		UPDATE templates SET type = 'optin'
		WHERE type = 'tx'
		AND id IN (SELECT DISTINCT optin_template_id FROM lists WHERE optin_template_id IS NOT NULL);
	`); err != nil {
		return err
	}

	return nil
}
