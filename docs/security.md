# Security

## Authentication

- Passwords are hashed with **bcrypt**. Login accepts ИИН, phone or email, looked up **by type only**: 12 digits → `iin`; contains `@` → `email`; anything else → `phone`. So nobody can register a phone equal to someone else's ИИН or email and take over (or lock out) their login.
- Phones are validated at registration and when the platform creates an ОСИ: Kazakhstan numbers `+7XXXXXXXXXX`, `8XXXXXXXXXX` or `7XXXXXXXXXX` (spaces, dashes, parentheses allowed) are stored as `+7XXXXXXXXXX`; anything else is a 400. Phone login matches the canonical form, and the raw input for rows created before this rule. The superadmin placeholder phone `superadmin:…` is never a login.
- The JWT (HS256, `JWT_SECRET`, TTL `JWT_TTL_HOURS`, default 24 h) holds `user_id`, `role`, `jk_id` and `tv` (the user's `token_version`). It is sent as an **HTTP-only** cookie (`Secure` in production), or as `Authorization: Bearer` from the Android app.
- **Sessions are checked against the DB on every request.** `middleware.Require` re-reads `id, role, jk_id, status, token_version, must_change_password` by primary key. The token is rejected (401) when the user is gone or `token_version` differs; role and ЖК for every downstream check come from the DB, not from the token (a demoted or moved ОСИ loses access at once). If the DB is down the answer is 503.
- `token_version` is bumped by: logout (`POST /auth/logout`, all devices of that user), password change (`PUT /me/password`; the caller immediately gets a new cookie, or a new `access_token` when it sends `X-Kladovki-Client: native`), the ОСИ resetting a resident's password, and the platform resetting an ОСИ password.
- **Temporary passwords are enforced by the server.** While `must_change_password` is set, only `GET /auth/me`, `PUT /me/password`, `GET /legal/status`, `POST /legal/accept` (plus public routes like `POST /auth/logout`, `GET /legal/:doc`) work; everything else behind login answers 403 `{"error":"Сначала смените временный пароль","code":"password_change_required"}`.
- `APP_ENV` unset means **production** (fail closed). The server refuses to start without `JWT_SECRET` unless `APP_ENV=development` exactly, and refuses any other `APP_ENV` value. The public fallback secret exists only for development.

## Rate limits

In-memory, per process (`internal/ratelimit`, fixed window, expired keys swept once per window). Defaults are in `config.DefaultRateLimits()`; the zero config (used by tests) disables them.

| What | Key | Default | Answer |
|---|---|---|---|
| `/auth/login`, `/auth/iin-status`, `/auth/register` (one shared budget) | client IP | 30 / min | 429 `{"error":"Слишком много запросов…","code":"rate_limited"}` + `Retry-After` |
| `POST /lead` | client IP | 10 / hour | same |
| `POST /rentals/lock` | client IP | 20 / 10 min | same |
| failed passwords on `/auth/login` | login identifier (canonical ИИН / email / phone) | 10 / 15 min | 429 «Слишком много попыток входа…», even with the right password, until the window ends; a successful login resets the counter |

- When the user is not found, login still runs one bcrypt compare against a dummy hash, so response time doesn't reveal which ИИН are registered (`/auth/iin-status` tells that by design; it is rate-limited).
- Trade-offs: counters reset on restart and are not shared between instances. The per-identifier limit lets someone who knows a victim's ИИН keep that account locked for 15 minutes at a time; the ОСИ password reset does not lift it (the counter is in memory). Per-IP limits only work when the client IP is real — see «Trusted proxies» in [operations.md](operations.md).

## Authorization: two layers

1. **Route level** (`router.go`): `Require()` means logged in. `RequireAdmin()` means role `admin`. `RequireSuperadmin()` means role `superadmin` (platform operator, `/platform/...`). A superadmin has no ЖК and does **not** pass `RequireAdmin`, so it can't read any ЖК's rentals, payments or residents.
2. **Ownership** (handlers):
   - `SameJK(c, jkID)`: the path's ЖК must equal the token's ЖК.
   - `storageInMyJK`: loads the storage room and checks its ЖК.
   - `canSeeRental`: the rental's owner, or an admin of its ЖК.
   - Resident lists use `/my` endpoints that filter by `user_id` from the token, never from the request.

Rule: **never trust IDs from the client to decide ownership.** Always derive them from the token, or load the record and compare.

Integration tests in `backend/internal/handlers/security_test.go` cover the rules above:

- a resident calling admin routes gets 403;
- an admin of ЖК A can't touch ЖК B;
- residents see only their own rentals and payments;
- storage creation is forced to the admin's ЖК.

## Who may rent

New users are `pending` until the ОСИ confirms they own property in the ЖК. Booking and renewal check `users.status = 'active'` on the server (`requireApproved`). The UI hiding the button is not the protection; the server check is.

## Webhooks

| Endpoint | Verified by | If the secret is missing |
|---|---|---|
| `/payments/webhook` | HMAC-SHA256 of the raw body, `X-Webhook-Signature: sha256=…`, constant-time compare | rejected (401). Exception: `APP_ENV=development` with no secret, for local tests |
| `/telegram/webhook` | `X-Telegram-Bot-Api-Secret-Token` must equal `TELEGRAM_WEBHOOK_SECRET` | rejected |

## Secrets

- **Never commit secrets.** `.gitignore` covers `.env` and `.env.*`, except `.env.example`, which holds names only.
- Put local values in `backend/.env.local`. On Render, use a **Secret File** named `.env.local` (mounted at `/etc/secrets/.env.local`) or individual env vars.
- The code has no fallback keys. If a key is missing, the feature turns off and the server logs a warning.
- **Incident 2026-11.** An ApiPay key and webhook secret had been committed to this public repo (commit `fdab4dd1`). Both were revoked and the code fallback was removed. Git history still contains the old values, which are now worthless. Rotate again if in doubt. Consider making the repo private.
- Database dumps (`*.dump`, `*.sql.gz`) contain personal data (ИИН, phones) and are git-ignored. Never share them.

## Personal data

ИИН and phone numbers are personal data under the Kazakhstan law on personal data. Keep them:

- **in the DB only**, not in logs, analytics, or notification texts beyond what is needed;
- **visible only to** the user themselves and the ОСИ of their ЖК.

## Checklist for new endpoints

- [ ] Is the route behind `auth` (and `admin` if needed)?
- [ ] Is every ID from the path or body checked against the token's ЖК or user?
- [ ] Does the server compute money amounts, rather than taking them from the client?
- [ ] Do error messages avoid leaking other users' data?
- [ ] Is there an integration test for the "wrong user / other ЖК" case?
