# Patra

Patra is a self-hosted newsletter, mailing list, and multi-channel posting manager (e-mail + web-push), packed into a single binary with a PostgreSQL database as its data store.

## Run with Docker (recommended)

The image builds itself from this repo — no local database needed. Postgres runs managed (eg: Supabase) via `PATRA_DATABASE_URL`.

```shell
cp .env.sample .env   # fill in PATRA_DATABASE_URL and mail/push keys. Never commit .env.
docker compose up --build -d
docker compose logs -f app
```

Visit `http://localhost:9000` (or your domain) and create the admin user on first visit. To create it up front instead: `PATRA_ADMIN_USER=... PATRA_ADMIN_PASSWORD=... docker compose up --build -d`.

Use the **direct** Postgres connection (port 5432), not a transaction pooler: `postgresql://postgres:PASSWORD@db.<ref>.supabase.co:5432/postgres?sslmode=require`.

## Binary

- `go build -o patra ./cmd` (Go 1.26+), plus `make build-frontend` and `make pack-bin` to bundle the admin UI.
- `./patra --new-config` to generate config.toml. Edit it.
- `./patra --install` to setup the Postgres DB (or `--upgrade` for an existing DB; upgrades are idempotent).
- Run `./patra` and visit `http://localhost:9000`.

## Channels

- **E-mail** — set `PATRA_SMTP_*` in `.env` (Hostinger preset by default) or configure servers in Settings → SMTP.
- **Web-push (FCM)** — set `FIREBASE_VAPID_KEY`, `FIREBASE_WEB_*`, and `FIREBASE_SERVICE_ACCOUNT_KEY` in `.env`, then open the Push Notifications page in the admin to register browsers and send tests.

## Developers

The backend is Go, the frontend is Vue 2 + Buefy. `make run` (backend) and `make run-frontend` for dev mode.

## License

Licensed under the AGPL v3 license. See LICENSE.
