# API reference

Base path: **`/api/v1`** (configurable with `API_PREFIX`). All bodies are JSON. Errors look like `{"error": "<message in Russian>"}`.

**Auth.** After login the server sets an HTTP-only JWT cookie (`access_token`). It also accepts `Authorization: Bearer <token>`, which the Android app uses.

**Legend**

| Mark | Meaning |
|---|---|
| — | public |
| 🔑 | any logged-in user |
| 🛡 | ОСИ admin only |
| ЖК | the resource must belong to the caller's own ЖК (otherwise 403) |

Source of truth: [`backend/internal/handlers/router.go`](../backend/internal/handlers/router.go).

## Auth

| Method | Path | Who | Body / notes |
|---|---|---|---|
| POST | `/auth/iin-status` | — | `{iin}` → `{registered: bool}`. Step 1 of the IIN login |
| POST | `/auth/login` | — | `{login, password}`. `login` = ИИН, phone or email. 401 means bad credentials; 503 means the DB is unavailable |
| POST | `/auth/register` | — | `{iin, full_name, phone, password, jk_id?}`. First login of a resident. 409 if the ИИН or phone is already taken |
| POST | `/auth/logout` | — | Clears the cookie |
| GET | `/auth/me` | 🔑 | The current user |

## ЖК & tariff

| Method | Path | Who | Body / notes |
|---|---|---|---|
| GET | `/jks/:id` | 🔑 ЖК | Name and address |
| GET | `/jks/:id/tariff` | 🔑 ЖК | `{amount}` ₸ per m² per month |
| PUT | `/jks/:id/tariff` | 🛡 ЖК | `{amount}` → `{recalc_requests: n}`. A decrease creates tariff_recalc requests |
| GET | `/jks/:id/tariff/preview?amount=` | 🛡 ЖК | `{affected_count, total_difference}` for a planned decrease |
| GET | `/jks/:id/rental-settings` | 🔑 ЖК | `{lock_duration_minutes}` |
| PUT | `/jks/:id/rental-settings` | 🛡 ЖК | `{lock_duration_minutes: 1–60}`; 400 outside the range |

## Storage rooms

| Method | Path | Who | Body / notes |
|---|---|---|---|
| GET | `/storages/jk/:jk_id` | 🔑 ЖК | All storage rooms of the ЖК with status |
| GET | `/storages/:id` | 🔑 ЖК | One storage room |
| POST | `/storages` | 🛡 | `{number, area, floor, entrance}`. The ЖК is taken from the token. 409 if the number is taken |
| PUT | `/storages/:id` | 🛡 ЖК | `{number, area, floor, entrance, status}`. 409 when changing the details of a room that isn't free |
| DELETE | `/storages/:id` | 🛡 ЖК | |
| PATCH | `/storages/:id/transfer` | 🔑 ЖК | `{iin}`. Name a successor. Only the current tenant or the ОСИ |

## Rentals

| Method | Path | Who | Body / notes |
|---|---|---|---|
| GET | `/rentals/my` | 🔑 | The caller's rentals |
| GET | `/rentals/:id` | 🔑 | Owner or the ОСИ of that ЖК |
| POST | `/rentals/lock` | 🔑 | `{storage_id, months}`. Hold for the ЖК's lock duration (default 2 min); 409 if not free |
| POST | `/rentals/:id/renew` | 🔑 owner | `{months (1–120), phone?}` → 201 payment `{id, amount, …}`. Kaspi invoice at the **current** tariff; when paid, `ends_at += months` from the current end. 409 if the rental isn't active or a transfer is pending. In development it is applied immediately |
| POST | `/rentals/:id/terminate` | 🔑 owner | `{comment?}` → an early_termination request |
| GET | `/rentals/jk/:jk_id` | 🛡 ЖК | All rentals in the ЖК |
| GET | `/rentals/storage/:storage_id` | 🛡 ЖК | The active rental of a storage room |
| PATCH | `/rentals/:id/cancel` | 🛡 ЖК | Cancel a rental; the room becomes free |
| PATCH | `/rentals/:id/force-release` | 🛡 ЖК | Clear a stuck hold |
| PATCH | `/rentals/:id/release-overdue` | 🛡 ЖК | End an overdue rental; the room becomes free |
| POST | `/rentals/confirm-payment` | 🔑 | **Development only** (`APP_ENV=development`): activate without paying. Not registered in production |

## Payments

| Method | Path | Who | Body / notes |
|---|---|---|---|
| POST | `/payments` | 🔑 owner | `{rental_id, phone?}`. Creates the Kaspi invoice. The server computes the amount. Hold → 10 min |
| GET | `/payments/my` | 🔑 | The caller's payments |
| GET | `/payments/jk/:jk_id` | 🛡 ЖК | All payments in the ЖК |
| POST | `/payments/webhook` | ApiPay | Requires `X-Webhook-Signature: sha256=<hmac>`. See [payments.md](payments.md) |

## Requests (заявки)

| Method | Path | Who | Body / notes |
|---|---|---|---|
| GET | `/requests/my` | 🔑 | The caller's requests |
| POST | `/requests/:id/choice` | 🔑 owner | `{choice: "extend" \| "refund"}` for a tariff_recalc request |
| GET | `/requests/jk/:jk_id` | 🛡 ЖК | All requests of the ЖК |
| GET | `/requests/open-count` | 🛡 | `{open}` for the nav badge |
| PATCH | `/requests/:id/resolve` | 🛡 ЖК | `{action: "approve" \| "reject", comment?}` |

## Notifications

| Method | Path | Who | Body / notes |
|---|---|---|---|
| GET | `/notifications` | 🔑 | The latest notifications of the caller |
| GET | `/notifications/unread-count` | 🔑 | `{unread}` |
| POST | `/notifications/read-all` | 🔑 | |
| POST | `/notifications/:id/read` | 🔑 | |
| GET | `/me/channels` | 🔑 | Which channels reach me: `{email, telegram, sms, whatsapp, telegram_available}` |
| POST | `/me/telegram/link` | 🔑 | `{url}`, a `t.me/<bot>?start=<token>` deep link (valid 1 day). 503 if the bot isn't configured |
| DELETE | `/me/telegram` | 🔑 | Unlink Telegram |
| POST | `/telegram/webhook` | Telegram | Requires `X-Telegram-Bot-Api-Secret-Token`. Handles `/start <token>` |

## ОСИ dashboard

All are 🛡 and scoped to the admin's ЖК.

| Method | Path | Returns |
|---|---|---|
| GET | `/dashboard/stats` | received (+ change), debt, debt_apartments, occupied/total storage rooms, active/new rentals |
| GET | `/dashboard/revenue` | Revenue by month |
| GET | `/dashboard/attention` | Rentals needing action (ending soon, overdue) with tenant contacts |
| GET | `/dashboard/activity` | Recent events |
| GET | `/dashboard/occupancy` | Free / occupied / locked breakdown |

## Misc

| Method | Path | Who | Notes |
|---|---|---|---|
| POST | `/lead` | — | `{full_name, phone, email?}` from the landing page |

## Status codes

| Code | When |
|---|---|
| 400 | Invalid body / validation (e.g. weak password, bad ИИН) |
| 401 | Not logged in / bad credentials / bad webhook signature |
| 403 | Logged in but not allowed (wrong role, other ЖК, not the owner) |
| 404 | Not found (also used for "not in your ЖК" on some reads) |
| 409 | Conflict: storage room not free, duplicate number, ИИН taken, etc. |
| 502 | ApiPay failed when creating an invoice |
| 503 | Database unavailable |
