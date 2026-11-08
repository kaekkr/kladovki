# Operations: run, test, deploy

## Run locally

```bash
# 1. Database (Postgres 16 + Adminer on :8081)
docker compose up -d postgres adminer

# 2. Backend (:8080). Migrations run automatically on start.
cd backend
cp .env.example .env.local      # once; fill in only what you need
air                              # hot reload; or: go run ./cmd/server

# 3. Frontend (:4200, talks to http://localhost:8080/api/v1)
cd frontend
npm ci
npm start
```

With `APP_ENV=development` you can activate a rental without Kaspi by calling `POST /api/v1/rentals/confirm-payment`.

## Environment variables

Env files are loaded in this order: `.env.local` → `/etc/secrets/.env.local` → `.env` → `/etc/secrets/.env`. The first non-empty value wins, and real environment variables override all files. Empty placeholders (`KEY=`) are skipped, so they never hide a value from a later file.

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `APP_ENV` | prod: `production` | `development` | Enables secure cookies and hides the dev-only routes |
| `PORT` | | `8080` | HTTP port (Render sets it) |
| `DB_PATH` | **yes** | local docker DSN | Postgres DSN (Supabase connection string) |
| `JWT_SECRET` | **prod** | dev value | Signs login tokens. Server won't start in production without it |
| `JWT_TTL_HOURS` | | `24` | Session length |
| `COOKIE_SECURE`, `COOKIE_NAME` | | auto / `access_token` | |
| `API_PREFIX` | | `/api/v1` | |
| `APP_BASE_URL` | for links + Telegram | | e.g. `https://kladovki.onrender.com` |
| `APIPAY_KEY` | for payments | | ApiPay API key |
| `APIPAY_WEBHOOK_SECRET` | for payments | | Webhook signature secret |
| `APIPAY_BASE_URL` | | `https://api.apipay.kz/api` | |
| `APIPAY_LIVE` | | empty | `true` = ignore sandbox events (real money) |
| `TELEGRAM_BOT_TOKEN`, `TELEGRAM_BOT_USERNAME`, `TELEGRAM_WEBHOOK_SECRET` | optional | | Telegram channel |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM` | optional | port `587` | Email channel |
| `MOBIZON_API_KEY` (`MOBIZON_API_BASE`) | optional | | SMS channel |
| `GREENAPI_INSTANCE_ID`, `GREENAPI_TOKEN` (`GREENAPI_BASE`) | optional | | WhatsApp channel |
| `TEST_DATABASE_URL` | tests only | | Postgres used by integration tests. **The tests wipe it** |

## Tests

```bash
# Backend: integration tests against a throwaway Postgres
docker run -d --name kladovki_test_db -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=kladovki_test -p 5442:5432 postgres:16-alpine
cd backend
TEST_DATABASE_URL='postgres://test:test@localhost:5442/kladovki_test?sslmode=disable' go test -p 1 ./...

# Frontend
cd frontend && npx ng test --watch=false
```

- Backend tests live in `internal/handlers/*_test.go`. They go through the real router and database, with fake ApiPay and Telegram servers. Covered areas: auth, IIN login, security, payments, requests, notifications. Channel unit tests are in `internal/notify`, and env loading is in `internal/config`.
- `-p 1` matters: the packages share one database.
- Without `TEST_DATABASE_URL` the DB tests are **skipped**, not failed.
- The project uses **TDD**: write the failing test first, then the code.

## CI

`.github/workflows/ci.yml` runs on every PR and every push to `main`:

- **backend**: Postgres service, then `go vet`, then `go test -p 1 ./...`;
- **frontend**: `npm ci`, then `ng test`, then `ng build`.

Merge only when CI is green.

## Deploy

- **Render** web service auto-deploys from `main`. The service runs from the repo root and serves `frontend/dist/frontend/browser` as the SPA. Build and start commands are set in the Render dashboard; there is no `render.yaml`.
- **Secrets on Render**: Dashboard → service → **Environment**. Either add the variables one by one, or add a **Secret File** named `.env.local` with the same content as your local one.
- **Database**: Supabase Postgres. Set the connection string as `DB_PATH`. Schema migrations run on every boot and are idempotent.
- **ApiPay webhook URL**: `https://kladovki.onrender.com/api/v1/payments/webhook`.
- **Android**: `cd frontend && npx ng build && npx cap sync android`, then build the APK in Android Studio. The app calls the production API URL from `src/environments/environment.ts`.

## Runbook

| Symptom | Likely cause | Fix |
|---|---|---|
| Login says "сервер недоступен" (503) | DB unreachable (paused Supabase, wrong `DB_PATH`) | Check Supabase status and connection limits; check Render logs for `failed to open database` |
| "Не удалось загрузить кладовки" | SQL error in the storages query | Render logs; usually a NULL in an unexpected column or a schema mismatch |
| Paid in Kaspi but the rental is still "Ожидает оплаты" | Webhook rejected (401: secret mismatch) or not delivered | ApiPay dashboard → webhook log; check `APIPAY_WEBHOOK_SECRET`; check `APIPAY_LIVE` isn't `true` while using sandbox |
| Invoice not created (502) | Wrong or missing `APIPAY_KEY`, or ApiPay down | Render logs: `apipay request failed` |
| Telegram not delivering | Bot token wrong, webhook not registered, user blocked the bot | Startup log `telegram webhook` line; `SELECT last_error FROM notification_deliveries WHERE channel='telegram' ORDER BY created_at DESC LIMIT 5` |
| Storage room stuck in "locked" | A hold is still alive (≤ 10 min) | Wait, or use admin **force-release** |

## Database backups

Supabase takes daily backups on paid plans. For a manual dump:

```bash
pg_dump "$DB_PATH" -Fc -f kladovki-$(date +%F).dump   # git-ignored; contains personal data
```
