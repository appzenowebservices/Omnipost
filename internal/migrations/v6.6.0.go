package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V6_6_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Per-list opt-in e-mails now reference a reusable template from the
	// Templates UI (Campaigns -> Templates) instead of storing raw HTML inline.
	// Idempotent: safe to re-run on an already migrated DB.
	if _, err := db.Exec(`
		ALTER TABLE lists
		ADD COLUMN IF NOT EXISTS optin_template_id INTEGER REFERENCES templates(id) ON DELETE SET NULL;
	`); err != nil {
		return err
	}

	// Migrate any existing inline opt-in bodies into the templates table and
	// point the list at the new template, so no custom HTML is lost.
	if _, err := db.Exec(`
		DO $$
		DECLARE
			r RECORD;
			new_id INT;
		BEGIN
			FOR r IN
				SELECT id, name, optin_template FROM lists
				WHERE optin_template <> '' AND optin_template_id IS NULL
			LOOP
				INSERT INTO templates (name, type, subject, body)
				VALUES ('Opt-in: ' || r.name, 'tx', 'Confirm subscription', r.optin_template)
				RETURNING id INTO new_id;

				UPDATE lists
				SET optin_template_id = new_id, optin_template = '', updated_at = NOW()
				WHERE id = r.id;
			END LOOP;
		END $$;
	`); err != nil {
		return err
	}

	return nil
}
