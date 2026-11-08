# Security

## Authentication

- Passwords are hashed with **bcrypt**. Login accepts ИИН, phone or email.
- The JWT (HS256, `JWT_SECRET`, TTL `JWT_TTL_HOURS`, default 24 h) holds `user_id`, `role` and `jk_id`. It is sent as an **HTTP-only** cookie (`Secure` in production), or as `Authorization: Bearer` from the Android app.
- In production the server refuses to start without `JWT_SECRET`.

## Authorization: two layers

1. **Route level** (`router.go`): `Require()` means logged in. `RequireAdmin()` means role `admin`.
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
