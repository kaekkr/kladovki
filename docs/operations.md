# Operations: run, test, deploy

## Run locally

```bash
# 1. Database (Postgres 16 + Adminer on :8081)
docker compose up -d postgres adminer

# 2. Backend (:8080). Migrations run automatically on start.
cd backend
cp .env.example .env.local      # once; fill in only what you need
echo 'APP_ENV=development' >> .env.local   # required locally: unset means production
air                              # hot reload; or: go run ./cmd/server

# 3. Frontend (:4200, talks to http://localhost:8080/api/v1)
cd frontend
npm ci
npm start
```

**Local development must set `APP_ENV=development`.** When `APP_ENV` (and `ENV`) are unset the server runs as **production**: it needs a real `JWT_SECRET`, sets `Secure` cookies and rejects unsigned webhooks. Any value other than exactly `development` or `production` stops the server at startup.

With `APP_ENV=development` you can activate a rental without Kaspi by calling `POST /api/v1/rentals/confirm-payment`.

## Environment variables

Env files are loaded in this order: `.env.local` → `/etc/secrets/.env.local` → `.env` → `/etc/secrets/.env`. The first non-empty value wins, and real environment variables override all files. Empty placeholders (`KEY=`) are skipped, so they never hide a value from a later file.

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `APP_ENV` (or `ENV`) | local: `development` | `production` | Exactly `development` or `production`; anything else is a startup error. `development` enables the dev-only routes, the dev JWT secret and non-Secure cookies |
| `PORT` | | `8080` | HTTP port (Render sets it) |
| `DB_PATH` | **yes** | local docker DSN | Postgres DSN (Supabase connection string) |
| `JWT_SECRET` | **yes** (except development) | dev value only in development | Signs login tokens. Server won't start without it unless `APP_ENV=development` |
| `JWT_TTL_HOURS` | | `24` | Session length |
| `COOKIE_SECURE`, `COOKIE_NAME` | | auto / `access_token` | |
| `API_PREFIX` | | `/api/v1` | |
| `APP_BASE_URL` | **prod** | | e.g. `https://kladovki.onrender.com`. Links, Telegram webhook, and the CORS allow-list (its origin) |
| `CORS_ORIGINS` | optional | | Extra exact origins allowed by CORS, comma-separated (e.g. a custom domain). No wildcards |
| `APIPAY_KEY` | for payments | | ApiPay API key |
| `APIPAY_WEBHOOK_SECRET` | for payments | | Webhook signature secret |
| `APIPAY_BASE_URL` | | `https://api.apipay.kz/api` | |
| `APIPAY_LIVE` | | empty | `true` = ignore sandbox events (real money) |
| `TELEGRAM_BOT_TOKEN`, `TELEGRAM_BOT_USERNAME`, `TELEGRAM_WEBHOOK_SECRET` | optional | | Telegram channel |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM` | optional | port `587` | Email channel |
| `MOBIZON_API_KEY` (`MOBIZON_API_BASE`) | optional | | SMS channel |
| `GREENAPI_INSTANCE_ID`, `GREENAPI_TOKEN` (`GREENAPI_BASE`) | optional | | WhatsApp channel |
| `SUPERADMIN_EMAIL`, `SUPERADMIN_PASSWORD` | optional | | Bootstrap the first platform operator at startup (see below). Password ≥ 10 characters |
| `TRUST_RENDER_PROXY` | **prod on Render**: `true` | empty | Trust `X-Forwarded-For` from private-network peers (10/8, 172.16/12, 192.168/16, loopback, fc00::/7). See «Trusted proxies» |
| `TRUSTED_PROXIES` | optional | empty | Extra proxy CIDRs/IPs whose `X-Forwarded-For` is trusted, comma-separated. An invalid entry stops the server |
| `TEST_DATABASE_URL` | tests only | | Postgres used by integration tests. **The tests wipe it** |

Rate limits have no env vars; the defaults live in `config.DefaultRateLimits()` (see [security.md](security.md#rate-limits)).

## Trusted proxies (client IP)

`c.ClientIP()` feeds the per-IP rate limits and the IP stored with every agreement acceptance. `gin.Default()` used to trust **every** proxy, so any client could choose its own IP by sending `X-Forwarded-For`. Now (`middleware.ConfigureClientIP`):

- **Default: no proxy is trusted.** The client IP is the TCP peer. Nothing can be spoofed, but behind Render's load balancer the peer is the balancer, so many or all users share one "IP": per-IP limits turn into near-global limits (30 auth calls / min for everyone), and the stored consent IP is the balancer's. The server logs a warning in production.
- **`TRUST_RENDER_PROXY=true` (recommended on Render).** `X-Forwarded-For` is believed only when the TCP peer is in a private range. gin reads the header from the right and returns the first address that isn't a trusted proxy, which is the address Render's proxy saw. A client-supplied left part (`X-Forwarded-For: 6.6.6.6, …`) is ignored, and `X-Real-IP` is never read. This is safe because a Render web service is reachable only through Render's proxy, so nobody else connects from a private address.
- **`TRUSTED_PROXIES=<cidr>,…`** adds other proxies (e.g. a CDN in front of Render).
- Limits of the approach: if another proxy with a public IP sits in front of Render (e.g. Cloudflare), the "client" is that proxy's edge IP until you add its ranges to `TRUSTED_PROXIES`. Render's internal addresses aren't documented, so this could not be checked offline. After deploying, check that `SELECT ip, count(*) FROM agreement_acceptances WHERE accepted_at > now() - interval '1 day' GROUP BY ip` shows varied public addresses, not one private one.

## First superadmin (platform operator)

The super-admin panel (`/platform`) creates ЖК and their ОСИ accounts. Its first account is created from env at startup:

1. Set `SUPERADMIN_EMAIL` and `SUPERADMIN_PASSWORD` (≥ 10 characters) on Render (Environment or the `.env.local` Secret File) and redeploy.
2. On boot the server makes sure a user with that email exists with role `superadmin` (no ЖК; the phone is a placeholder `superadmin:<email>`). The log says `superadmin bootstrap: created …` or `… already exists`. The password is never logged.
3. Log in at `/login` with that email and password; you land on `/platform`.
4. Optionally remove `SUPERADMIN_PASSWORD` afterwards. Later boots never change an existing account's password, and an email that belongs to an ОСИ or resident is **never** promoted (a warning is logged instead).

Creating a ЖК or an ОСИ account (or resetting an ОСИ password) shows a temporary password **once**. Hand it over in person; the ОСИ must choose a new password at first login. A new ЖК has no tariff: the ОСИ sets it in Настройки before residents can book.

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

## Test data

`backend/db/seed/seed_test_data.sql` creates a separate test ЖК «ЖК Тестовый (seed)». It contains 12 storage rooms, an ОСИ admin and residents in every state:
- active, ending soon, overdue, more than 2 months in debt;
- a company tenant;
- an early-termination request;
- a rental released for debt, with its termination record;
- pending, rejected and blocked users.

Every user's password is `test1234`; the logins are listed at the top of the file.

- Run it in the Supabase SQL editor after the backend has created the schema at least once. It is safe to run again.
- It touches nothing outside the test ЖК, and every row id starts with `seed-`.
- `backend/db/seed/cleanup_test_data.sql` removes the test ЖК and everything in it, including users who later registered into it.

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
