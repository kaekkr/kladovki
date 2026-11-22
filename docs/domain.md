# Domain & business rules

## Glossary

| Term | Meaning |
|---|---|
| **ЖК** (`jks`) | Residential complex. Everything (storage rooms, tariff, users, requests) belongs to exactly one ЖК |
| **ОСИ** | Building owners' association, the **admin** role. One ОСИ admin manages one ЖК |
| **Оператор платформы** | The **superadmin** role (us). Creates ЖК and their ОСИ accounts in `/platform`; has no ЖК and can't use ОСИ pages |
| **Кадастровый номер** | ЖК: `21:319:021:848:8`. Premises: the ЖК number + `:unit`, e.g. `21:319:021:848:8:ВП-2`. Used to find a resident's ЖК |
| **Житель** | Resident, the **resident** role. Rents storage rooms in their own ЖК |
| **Кладовка** (`storages`) | A storage room: number, area (м²), floor, entrance |
| **Аренда** (`rentals`) | A rental of one storage room by one resident for N months at a fixed monthly price |
| **Тариф** (`tariffs`) | Monthly price in ₸ **per m²** in this ЖК. A room's monthly price = `round(area × tariff)`, fixed into the rental when it is booked or renewed |
| **Заявка** (`requests`) | Something the ОСИ must act on, or a resident must answer (refund, early termination, tariff choice) |
| **Долг** | Money owed for an overdue rental: every started day after `ends_at` costs `price_per_month / 30`. For a rental released for debt: frozen at the release + termination costs |
| **ИИН** | 12-digit Kazakh personal ID number, used as the resident's login |

## Roles

| | Resident (`resident`) | ОСИ (`admin`) |
|---|---|---|
| Sees | storage rooms of their ЖК, **their own** rentals, payments, requests, notifications | everything in **their** ЖК |
| Can | book + pay, choose extend/refund, ask for early termination, name a successor (transfer) | create, edit and delete storage rooms; set the tariff; cancel or release rentals; resolve requests; see dashboards |
| Logs in with | ИИН (also phone / email) | phone / email / ИИН |

Users of one ЖК can never see or change anything in another ЖК. See [security.md](security.md).

The **platform operator** (`superadmin`) belongs to no ЖК. In the super-admin panel (`/platform`) they list all ЖК with counts, create a ЖК together with its first ОСИ account, edit a ЖК's name, address and cadastral number, add more ОСИ accounts and reset an ОСИ's password. They see no rentals, payments or residents' data. The first superadmin comes from env (see [operations.md](operations.md)).

## User agreement (consent)

- The product owner wrote two agreements in Russian and Kazakh: **«Пользовательское соглашение и правила аренды кладовых»** for residents, and **«Регламент использования сервиса для ОСИ»**. They live in `backend/internal/legal/texts/`, can be read at `/terms/resident_terms` and `/terms/osi_rules`, and are linked from the profile and from Настройки.
- **Residents** tick «Я принимаю…» at **registration** and at **every payment and renewal**. Per the product owner, that tick at payment *is* the rental contract. The checkbox opens the full text with a Русский / Қазақша switch.
- **ОСИ** see a full-screen prompt in the admin panel until they accept the current version of the regulation.
- Each acceptance is stored in `agreement_acceptances` with user, document, version, context (`registration`/`payment`/`renewal`/`prompt`), rental, IP, browser and time, as evidence.
- **Changing a text:** edit the file and bump its version in `legal.go`. Residents accept the new version at their next payment; ОСИ are prompted again.

## Who may rent

Per the user agreement (п. 2.1), only **owners of a flat or commercial premises** in the ЖК may rent. This covers individuals and companies; one owner may rent several rooms. Flat tenants and outsiders may not.

- At registration the user states **what they own**: a flat or commercial premises, and its number. They also say whether they act as a person or **represent a company**. A company representative adds the company's **БИН** (12 digits) and name; the ИИН stays the representative's own. An ИП registers as a person, since its БИН is the ИИН.
- The account starts as **pending**: they can log in and browse, but booking, paying a hold and renewal are refused (403). The message depends on the status: pending «Аккаунт ожидает подтверждения ОСИ…», rejected «ОСИ отклонил подтверждение аккаунта…», blocked «Аккаунт заблокирован ОСИ…». A hold made before the ОСИ rejected/blocked the user can't be paid either.
- The ОСИ checks them on **Жители** (`/admin/residents`):
  - **Подтвердить**: the user becomes `active`;
  - **Отклонить**: a reason is required, and the user sees it, fixes the data, and resubmits;
  - **Заблокировать**: existing rentals stay, but no new bookings or renewals.
- Users who existed before this feature are `active`.
- **Later:** an automatic ownership check via egov.kz (п. 2.2), replacing the manual check.

## Registration & login (IIN flow)

1. The resident types their ИИН (an ОСИ or the platform operator types an email). The app calls `POST /auth/iin-status`.
   - **Known ИИН**: ask for the password.
   - **Unknown ИИН**: go to `/auth/register?iin=…`, the "Вы впервые входите в систему. Придумайте пароль." screen. It asks for name, phone, owner type, the **cadastral number of the premises**, and the password twice.
2. **The ЖК is found from the cadastral number** (since 2026-11-22; the QR code was removed from Настройки). For a premises `21:319:021:848:8:ВП-2`, the ЖК is the one whose `jks.cadastral_number` is `21:319:021:848:8`. The last facet is the unit: digits mean a flat (`154`), anything else a non-residential premises (`ВП-2`, `НП-14`). The rules live in `backend/internal/cadastral`.
   - A wrong format, a ЖК that isn't connected, or an old QR link (`jk_id`) pointing to another ЖК → 400 with a Russian message.
   - Older clients may still send `jk_id` + `premises_number` without a cadastral number. Then the QR / single-ЖК rule applies.
3. Ownership is **still confirmed by the ОСИ** (see "Who may rent"). The egov check via Smart Bridge will later replace the manual step, using the same cadastral numbers.
4. Validation: the ИИН must be exactly 12 digits (the checksum is not verified yet), the password at least 6 characters, and the phone and ИИН unique. Errors are in Russian.
5. **Android app**: opens on the login screen (or the user's home if a session is stored). The marketing landing is website-only.

Password reset: the ОСИ sets a temporary password on «Жители», and the resident must change it at the next login.

## Booking & payment

![booking](diagrams/booking-payment-flow.svg)

| Step | Rule |
|---|---|
| Lock | Only a `free` storage room can be locked. The hold lasts the ЖК's **lock duration**: set by the ОСИ in Настройки (1–60 min, default 2; `jks.lock_minutes`). A second resident gets 409 "ожидается подтверждение оплаты". **One unpaid hold per resident**: a second live hold → 409 «У вас уже есть неоплаченная бронь…» (checked inside the lock transaction under a row lock on the user, so parallel taps can't get around it) |
| Invoice | `POST /payments` with `rental_id` (+ optional Kaspi phone). The amount is `price_per_month × months`, **computed on the server**. The hold is extended to **10 minutes** (`service.PaymentWindow`) |
| Paid | The webhook sets the rental to `active`, `starts_at = now`, `ends_at = now + months`, `total_paid = amount`, and the storage room to `occupied`. Repeated webhooks are no-ops |
| Paid late | If the hold expired but the storage room is still free, the rental is activated anyway |
| Paid, room taken | Someone else rents the room (an **overdue** tenant included — a late webhook never evicts anyone) or holds a live booking. The rental is not activated. A **payment_refund** request is opened for the ОСИ. Exception: a handover to the payer by ИИН while the current paid period is still running |
| Not paid | Invoice `cancelled` / `expired` / `error` → payment `failed`. The hold runs out → storage room `free` |

## Rental lifecycle

![lifecycle](diagrams/rental-lifecycle.svg)

- **Overdue** is not a separate status. It is an `active` rental whose `ends_at` has passed. The storage room stays `occupied` (the resident's things are inside). Reminders and debt warnings go out, and the ОСИ decides what to do.
- ОСИ actions:
  - **cancel** (any active rental);
  - **release-overdue** (rental → `expired`, room → `free`, with a termination record). The debt does **not** disappear: it is frozen at the release moment (`price_per_month / 30` × started days from `ends_at` to the release) plus the recorded `costs`, and keeps showing in «Начисления», «Задолженности» and the dashboard total. There is no «settled» mark yet: the ОСИ collects it outside the app;
  - **force-release** (clear a stuck hold).
- The UI shows an unpaid expired hold as "Бронь истекла". The cost of a hold is shown as `price × months`; once paid, it's `total_paid`.

## Renewal (продление)

- Any **active** rental can be renewed by its tenant, **overdue included**: «Оплатить продление» on the rental page → choose 1/3/6/12 months → Kaspi invoice (`POST /rentals/:id/renew`).
- The price is the **current** tariff: `round(area × tariff) × months`. After payment the rental's `price_per_month` becomes that price.
- When paid: `ends_at += months` **from the current end date**, not from today. An overdue tenant therefore pays for the overdue time first. `total_paid += amount`, `months += months`.
- Not allowed while a transfer to a successor is pending (409).
- Not allowed while the tenant's **early termination** request is open (409 «Подана заявка на досрочное расторжение…»).
- If the rental was ended (cancelled or released) before the money arrived, it is not revived. A `payment_refund` request is opened instead.

## Remaining value

Several rules need "how much of what was paid is still unused" (early termination, ОСИ cancel, the refund option of a tariff decrease and its preview). Each rental keeps an **anchor**: the unused time was worth `V` ₸ at moment `A`, and the value runs down evenly to 0 at `ends_at`:

```
remaining value = V × (ends_at − now) / (ends_at − A)      clamped to [0, V]
```

| Event | Anchor after it |
|---|---|
| Paid booking (fresh rental) | none stored: `V = total_paid`, `A = starts_at`. So an N-month rental at price P is worth exactly **P × N** at the start and **0** at the end |
| Renewal paid | `V = remaining value now + renewal amount`, `A = now` (or the old `ends_at` if the rental was overdue: that overdue time is already used up) |
| Tariff decrease, **extend** | `V = remaining value now` (unchanged money, spread over the longer period), `A = now` |
| Tariff decrease, **refund** | `V = remaining value now − refund`, `A = now` |

Stored in `rentals.value_left` / `rentals.value_at` (NULL = no anchor yet). Rentals extended or refunded before this rule (2026-11-22) keep their old anchor and may still be over-valued; the ОСИ checks such an amount before approving.

Worked examples (QA, 2026-11-22):

- 15 000 ₸ for 6 months, 3 months used, tariff halved, **extend**: remaining value was 7 500 ₸ at the choice, so the anchor is `V = 7 500`, `A = now`, `ends_at` = now + 6 months. Early termination right away returns **7 500 ₸** (the old formula returned 10 000).
- 24 000 ₸ for 6 months, 5 months left (≈ 19 912 ₸), tariff halved, **refund** (≈ 9 956 ₸ owed): anchor `V ≈ 9 956`, `A = now`. A later early termination returns ≈ 9 956 ₸ × (time left / time left at the choice), not 11 652.
- 10 000 ₸ for 2 months, half used (5 000 left), then a 1-month renewal at 6 000 ₸: remaining value right after it = **11 000 ₸**.

## Tariff change

![requests](diagrams/requests-flow.svg)

- **Increase**: existing rentals keep their price until they end. Residents get a `tariff_changed` notification.
- **Decrease**: before saving, the ОСИ sees a preview: how many rentals are affected and the total difference (`GET /jks/:id/tariff/preview?amount=`). When saved, each active rental with time left gets a **tariff_recalc** request in status `awaiting_choice`. Older unanswered ones become `superseded`. The resident picks one of two options:
  - **extend**: the remaining time is stretched to the new price, `ends_at += remaining_time × (old/new − 1)`. Applied immediately, request `done`.
  - **refund**: the difference `remaining value now × (1 − new/old)` is computed **when the tenant answers** and stored as the request amount; `price_per_month = new` and the remaining value is re-anchored without it. The request becomes `open` and the ОСИ admins get a «Новая заявка: возврат разницы по тарифу» notification. The ОСИ returns the money and presses «Деньги возвращены» (approve); only then `total_paid −= amount`. Until then the rental shows `total_paid` unchanged and `refund_pending` = the amount, so the UI can show «Оплачено 10 000 ₸ · к возврату 2 500 ₸». The ОСИ **cannot reject** it (409): the tenant already gave up the extension. The refund stays open (owed) even if the rental ends before it is paid.
- **How the ОСИ sees it:** «Заявки» shows unanswered requests (`awaiting_choice`) in a separate block, «Ждут ответа арендатора». Once the tenant answers, the item leaves that block and appears as the tenant's заявка: «Возврат разницы после снижения тарифа» (to act on) or «Продление срока после снижения тарифа» (already applied, closed). It is the same DB row, shown differently.

## Early termination

- The resident clicks «Досрочное расторжение» (with an optional comment, up to 1000 characters). This creates an **early_termination** request (`open`) with `amount` = remaining value. Only one open request per rental is allowed.
- Refused (409 «Счёт на продление ещё ожидает оплаты…») while a renewal invoice for the rental, sent less than **30 minutes** ago (`repository.RenewalPendingWindow`), is still pending.
- **A renewal paid after the request was filed** (an older Kaspi invoice paid late) does not extend the rental: the money goes to a `payment_refund` request, as when the rental has ended. So the amount the ОСИ approves stays correct, and the renewal is returned whether the ОСИ approves or rejects. (Chosen over recomputing the amount at approval: one rule, no hidden change of an amount the tenant already saw.)
- The ОСИ can **approve**: the rental becomes `cancelled`, `ends_at = now`, the storage room becomes `free`, and the ОСИ refunds the amount. Or **reject**, with a comment.

## Transfer to another resident (by ИИН)

- The current tenant (or the ОСИ) sets `transfer_to_iin` on an occupied storage room (`PATCH /storages/:id/transfer`), **at any time during the rental** (or after it ended). `transfer_until` = rental `ends_at` + **14 days**. The tenant's own ИИН → 400 «Нельзя передать кладовку текущему арендатору…».
- Only the resident with that ИИН can book this storage room while the window is open.
- **Debt first** (owner, 2026-11-22): while the current rental is overdue (`active` and `ends_at` passed, i.e. it has a debt), the successor's booking is refused: 409 «Предыдущий арендатор не оплатил задолженность…». The overdue tenant can still renew (pay the debt) despite the pending transfer; a tenant who is not overdue can't renew while the transfer is pending (409).
- When the successor pays, the previous rental ends **now** (`expired`, `ends_at = now`); its unused part (remaining value) is owed back to the previous tenant as a `payment_refund` request («Кладовка передана новому арендатору…»), and its pending requests are closed.
- If the successor's booking is paid after the previous rental became overdue (the 10-minute payment window crossed `ends_at`), the room counts as taken and the successor's money goes to a `payment_refund`.

## Storage editing rules

- Room numbers are unique per ЖК (case- and space-insensitive) → 409.
- Number, area, floor and entrance of a room that is not `free` cannot be changed → 409.
- Creating a storage room always uses the admin's own ЖК, whatever the body says.

## Known gaps

- The eGov/GBD ownership check (via Smart Bridge) isn't built yet. Until then the ОСИ confirms owners and blocks ex-owners by hand.
- Refunds are tracked as requests. The money itself is returned manually by the ОСИ; ApiPay refund isn't automated.
