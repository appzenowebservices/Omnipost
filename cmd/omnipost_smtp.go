package main

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
)

// OmniPost default e-mail provider preset (Hostinger).
//
// The app's SMTP settings live in the DB (settings key 'smtp', seeded from
// schema.sql). For SaaS-style deploys, setting OMNIPOST_SMTP_* env vars
// overrides the DB-backed SMTP config at startup with a single enabled
// Hostinger server, so fresh deploys send mail without manual UI setup.
//
// Supported vars (OMNIPOST_ takes precedence, LISTMONK_ accepted as alias):
//
//	OMNIPOST_SMTP_HOST      default smtp.hostinger.com (when any SMTP env is set)
//	OMNIPOST_SMTP_PORT      default 465
//	OMNIPOST_SMTP_USERNAME  e.g. contact@appzenowebservices.com
//	OMNIPOST_SMTP_PASSWORD  SMTP password / API secret (never commit)
//	OMNIPOST_SMTP_AUTH      default login (login|plain|cram|none)
//	OMNIPOST_SMTP_TLS       default TLS (TLS|STARTTLS|none)
//	OMNIPOST_SMTP_NAME      default email-omnipost
//	OMNIPOST_FROM_EMAIL     e.g. OmniPost <contact@appzenowebservices.com>
//
// If none of HOST/PASSWORD/FROM_EMAIL are set, this is a no-op and the DB
// (schema.sql defaults + Settings UI) remains the source of truth.
func applyOmniPostSMTPEnv(ko *koanf.Koanf, db *sqlx.DB) {
	host := firstEnv("OMNIPOST_SMTP_HOST", "LISTMONK_SMTP_HOST")
	portStr := firstEnv("OMNIPOST_SMTP_PORT", "LISTMONK_SMTP_PORT")
	username := firstEnv("OMNIPOST_SMTP_USERNAME", "LISTMONK_SMTP_USERNAME")
	password := firstEnv("OMNIPOST_SMTP_PASSWORD", "LISTMONK_SMTP_PASSWORD")
	auth := firstEnv("OMNIPOST_SMTP_AUTH", "LISTMONK_SMTP_AUTH")
	tls := firstEnv("OMNIPOST_SMTP_TLS", "LISTMONK_SMTP_TLS")
	name := firstEnv("OMNIPOST_SMTP_NAME", "LISTMONK_SMTP_NAME")
	fromEmail := firstEnv("OMNIPOST_FROM_EMAIL", "LISTMONK_FROM_EMAIL", "OMNIPOST_SMTP_FROM")

	if host == "" && password == "" && username == "" && fromEmail == "" {
		return
	}

	// Zero-config SaaS default: if credentials are given without an
	// explicit host, assume the Hostinger preset.
	if host == "" {
		host = "smtp.hostinger.com"
	}
	port := 465
	if p, err := strconv.Atoi(strings.TrimSpace(portStr)); err == nil && p > 0 && p < 65536 {
		port = p
	}
	if auth == "" {
		auth = "login"
	}
	if tls == "" {
		tls = "TLS"
	}
	if name == "" {
		name = "email-omnipost"
	}

	// Reuse the existing SMTP UUID (if any) so password-UUID matching in
	// Settings updates and UI state stay stable across restarts.
	uuidStr := ""
	if slices := ko.Slices("smtp"); len(slices) > 0 {
		uuidStr = slices[0].String("uuid")
	}
	if uuidStr == "" {
		uuidStr = uuid.Must(uuid.NewV4()).String()
	}

	server := map[string]any{
		"name":            name,
		"uuid":            uuidStr,
		"enabled":         true,
		"host":            strings.TrimSpace(host),
		"hello_hostname":  "",
		"port":            port,
		"auth_protocol":   auth,
		"username":        username,
		"password":        password,
		"email_headers":   []any{},
		"max_conns":       10,
		"max_msg_retries": 2,
		"msg_retry_delay": "10ms",
		"idle_timeout":    "15s",
		"wait_timeout":    "5s",
		"tls_type":        tls,
		"tls_skip_verify": false,
		"from_addresses":  []any{},
	}

	ko.Set("smtp", []map[string]any{server})
	lo.Printf("OmniPost SMTP preset active: %s:%d (user %s)", host, port, username)

	if fromEmail != "" {
		ko.Set("app.from_email", fromEmail)
	}

	// Persist back to the DB so the Settings UI reflects the env-driven
	// config instead of showing stale values.
	if db == nil {
		return
	}
	if b, err := json.Marshal([]map[string]any{server}); err == nil {
		if _, err := db.Exec(`UPDATE settings SET value = $1::JSONB, updated_at = NOW() WHERE key = 'smtp'`, string(b)); err != nil {
			lo.Printf("warning: could not persist OMNIPOST_SMTP_* to settings table: %v", err)
		}
	}
	if fromEmail != "" {
		if b, err := json.Marshal(fromEmail); err == nil {
			if _, err := db.Exec(`UPDATE settings SET value = $1::JSONB, updated_at = NOW() WHERE key = 'app.from_email'`, string(b)); err != nil {
				lo.Printf("warning: could not persist OMNIPOST_FROM_EMAIL to settings table: %v", err)
			}
		}
	}
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}
