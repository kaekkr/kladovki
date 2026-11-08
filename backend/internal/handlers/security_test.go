package handlers

import (
	"net/http"
	"testing"
	"time"
)

// Price in the seed: area 5 m² × tariff 1000 ₸ = 5000 ₸/month.

func TestResidentCannotUseAdminEndpoints(t *testing.T) {
	e := newTestEnv(t)
	cases := []struct {
		name, method, path string
		body               any
	}{
		{"set tariff", http.MethodPut, "/jks/" + e.jkA + "/tariff", map[string]any{"amount": 1}},
		{"tariff preview", http.MethodGet, "/jks/" + e.jkA + "/tariff/preview?amount=1", nil},
		{"create storage", http.MethodPost, "/storages", map[string]any{"number": "99", "area": 1}},
		{"update storage", http.MethodPut, "/storages/" + e.storageA, map[string]any{"number": "1", "area": 50}},
		{"delete storage", http.MethodDelete, "/storages/" + e.storage2, nil},
		{"all rentals of JK", http.MethodGet, "/rentals/jk/" + e.jkA, nil},
		{"all payments of JK", http.MethodGet, "/payments/jk/" + e.jkA, nil},
		{"dashboard", http.MethodGet, "/dashboard/stats", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := e.do(c.method, c.path, e.resident, c.body)
			if r.Code != http.StatusForbidden {
				t.Fatalf("want 403, got %d: %s", r.Code, r.Body)
			}
		})
	}
}

func TestAdminIsScopedToOwnJK(t *testing.T) {
	e := newTestEnv(t)
	cases := []struct {
		name, method, path string
		body               any
	}{
		{"list other JK storages", http.MethodGet, "/storages/jk/" + e.jkB, nil},
		{"list other JK rentals", http.MethodGet, "/rentals/jk/" + e.jkB, nil},
		{"list other JK payments", http.MethodGet, "/payments/jk/" + e.jkB, nil},
		{"set other JK tariff", http.MethodPut, "/jks/" + e.jkB + "/tariff", map[string]any{"amount": 1}},
		{"update other JK storage", http.MethodPut, "/storages/" + e.storageB, map[string]any{"number": "1", "area": 9}},
		{"delete other JK storage", http.MethodDelete, "/storages/" + e.storageB, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := e.do(c.method, c.path, e.adminA, c.body)
			if r.Code != http.StatusForbidden && r.Code != http.StatusNotFound {
				t.Fatalf("want 403/404, got %d: %s", r.Code, r.Body)
			}
		})
	}
	if r := e.do(http.MethodGet, "/storages/jk/"+e.jkA, e.adminA, nil); r.Code != http.StatusOK {
		t.Fatalf("own JK storages: %d %s", r.Code, r.Body)
	}
}

func TestResidentSeesOnlyOwnJKAndOwnRentals(t *testing.T) {
	e := newTestEnv(t)
	if r := e.do(http.MethodGet, "/storages/jk/"+e.jkB, e.resident, nil); r.Code != http.StatusForbidden {
		t.Fatalf("other JK storages: want 403, got %d", r.Code)
	}
	if r := e.do(http.MethodGet, "/storages/"+e.storageB, e.resident, nil); r.Code != http.StatusNotFound {
		t.Fatalf("other JK storage: want 404, got %d", r.Code)
	}
	if r := e.do(http.MethodPost, "/rentals/lock", e.resident, map[string]any{"storage_id": e.storageB, "months": 1}); r.Code != http.StatusNotFound {
		t.Fatalf("lock other JK storage: want 404, got %d", r.Code)
	}

	othersRental := e.lockAs(e.other, e.storage2, 1)
	if r := e.do(http.MethodGet, "/rentals/"+othersRental, e.resident, nil); r.Code != http.StatusNotFound {
		t.Fatalf("other resident's rental: want 404, got %d", r.Code)
	}

	mine := e.lockAs(e.resident, e.storageA, 1)
	r := e.do(http.MethodGet, "/rentals/my", e.resident, nil)
	if r.Code != http.StatusOK {
		t.Fatalf("my rentals: %d %s", r.Code, r.Body)
	}
	var list []struct {
		ID string `json:"id"`
	}
	r.json(t, &list)
	if len(list) != 1 || list[0].ID != mine {
		t.Fatalf("my rentals should contain only own rental, got %+v", list)
	}
	if r := e.do(http.MethodGet, "/payments/my", e.resident, nil); r.Code != http.StatusOK {
		t.Fatalf("my payments: %d %s", r.Code, r.Body)
	}
	// Admin of the JK can still open any rental of the JK.
	if r := e.do(http.MethodGet, "/rentals/"+othersRental, e.adminA, nil); r.Code != http.StatusOK {
		t.Fatalf("admin rental view: %d", r.Code)
	}
}

func TestConfirmPaymentShortcutDisabledInProduction(t *testing.T) {
	e := newTestEnv(t)
	id := e.lockAs(e.resident, e.storageA, 1)
	r := e.do(http.MethodPost, "/rentals/confirm-payment", e.resident, map[string]any{"rental_id": id})
	if r.Code == http.StatusOK {
		t.Fatalf("confirm-payment must not activate a rental without payment in production: %s", r.Body)
	}
	var status string
	_ = e.db.QueryRow(`SELECT status FROM rentals WHERE id=$1`, id).Scan(&status)
	if status != "locked" {
		t.Fatalf("rental status = %s, want locked", status)
	}
}

func TestPaymentAmountIsComputedOnServer(t *testing.T) {
	e := newTestEnv(t)
	id := e.lockAs(e.resident, e.storageA, 2)
	r := e.do(http.MethodPost, "/payments", e.resident, map[string]any{"rental_id": id, "amount": 1, "phone": "+77001112233"})
	if r.Code != http.StatusOK {
		t.Fatalf("create payment: %d %s", r.Code, r.Body)
	}
	inv := e.apipay.last()
	if inv == nil || inv["amount"] != float64(10000) {
		t.Fatalf("invoice amount = %v, want 10000 (5000 × 2 months)", inv["amount"])
	}
	var amount int64
	_ = e.db.QueryRow(`SELECT amount FROM payments WHERE rental_id=$1`, id).Scan(&amount)
	if amount != 10000 {
		t.Fatalf("stored payment amount = %d, want 10000", amount)
	}
}

func TestCannotPayForSomeoneElsesRental(t *testing.T) {
	e := newTestEnv(t)
	id := e.lockAs(e.other, e.storageA, 1)
	r := e.do(http.MethodPost, "/payments", e.resident, map[string]any{"rental_id": id, "phone": "+77001112233"})
	if r.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d %s", r.Code, r.Body)
	}
}

func paidEvent(paymentID string) map[string]any {
	return map[string]any{
		"event":      "invoice.status_changed",
		"is_sandbox": false,
		"invoice":    map[string]any{"id": 77, "external_order_id": paymentID, "status": "paid", "amount": "10000.00"},
	}
}

func TestWebhookRequiresValidSignature(t *testing.T) {
	e := newTestEnv(t)
	id := e.lockAs(e.resident, e.storageA, 2)
	e.do(http.MethodPost, "/payments", e.resident, map[string]any{"rental_id": id, "phone": "+77001112233"})
	pid := e.paymentIDFor(id)

	for _, sig := range []string{"", "sha256=deadbeef", sign([]byte("other"), webhookSecret)} {
		if r := e.webhook(paidEvent(pid), sig); r.Code != http.StatusUnauthorized {
			t.Fatalf("signature %q: want 401, got %d", sig, r.Code)
		}
	}
	var status string
	_ = e.db.QueryRow(`SELECT status FROM rentals WHERE id=$1`, id).Scan(&status)
	if status != "locked" {
		t.Fatalf("unsigned webhook changed rental to %s", status)
	}
}

func TestPaidWebhookActivatesRentalWithRealPeriodAndTotal(t *testing.T) {
	e := newTestEnv(t)
	id := e.lockAs(e.resident, e.storageA, 2)
	e.do(http.MethodPost, "/payments", e.resident, map[string]any{"rental_id": id, "phone": "+77001112233"})

	if r := e.webhook(paidEvent(e.paymentIDFor(id)), "valid"); r.Code != http.StatusOK {
		t.Fatalf("webhook: %d %s", r.Code, r.Body)
	}
	assertActiveTwoMonths(t, e, id)

	// Duplicate delivery is harmless.
	if r := e.webhook(paidEvent(e.paymentIDFor(id)), "valid"); r.Code != http.StatusOK {
		t.Fatalf("duplicate webhook: %d", r.Code)
	}
	assertActiveTwoMonths(t, e, id)
}

func TestPaymentArrivingAfterLockExpiryStillActivatesFreeStorage(t *testing.T) {
	e := newTestEnv(t)
	id := e.lockAs(e.resident, e.storageA, 2)
	e.do(http.MethodPost, "/payments", e.resident, map[string]any{"rental_id": id, "phone": "+77001112233"})
	// Simulate the lock running out and the worker freeing the storage.
	e.exec(`UPDATE rentals SET status='expired', ends_at=now() - interval '1 minute' WHERE id=$1`, id)
	e.exec(`UPDATE storages SET status='free' WHERE id=$1`, e.storageA)

	if r := e.webhook(paidEvent(e.paymentIDFor(id)), "valid"); r.Code != http.StatusOK {
		t.Fatalf("webhook: %d %s", r.Code, r.Body)
	}
	assertActiveTwoMonths(t, e, id)
}

func TestSandboxEventsIgnoredInLiveMode(t *testing.T) {
	e := newTestEnv(t)
	id := e.lockAs(e.resident, e.storageA, 2)
	e.do(http.MethodPost, "/payments", e.resident, map[string]any{"rental_id": id, "phone": "+77001112233"})
	ev := paidEvent(e.paymentIDFor(id))
	ev["is_sandbox"] = true
	if r := e.webhook(ev, "valid"); r.Code != http.StatusOK {
		t.Fatalf("webhook: %d", r.Code)
	}
	var status string
	_ = e.db.QueryRow(`SELECT status FROM rentals WHERE id=$1`, id).Scan(&status)
	if status != "locked" {
		t.Fatalf("sandbox event activated rental (%s)", status)
	}
}

func assertActiveTwoMonths(t *testing.T, e *testEnv, rentalID string) {
	t.Helper()
	var status, storageStatus string
	var total int64
	var starts, ends time.Time
	err := e.db.QueryRow(`
		SELECT r.status, r.total_paid, r.starts_at, r.ends_at, s.status
		FROM rentals r JOIN storages s ON s.id = r.storage_id WHERE r.id=$1`, rentalID).
		Scan(&status, &total, &starts, &ends, &storageStatus)
	if err != nil {
		t.Fatal(err)
	}
	if status != "active" || storageStatus != "occupied" {
		t.Fatalf("status=%s storage=%s, want active/occupied", status, storageStatus)
	}
	if total != 10000 {
		t.Fatalf("total_paid=%d, want 10000", total)
	}
	if want := starts.AddDate(0, 2, 0); !ends.Equal(want) {
		t.Fatalf("ends_at=%v, want starts_at+2 months=%v", ends, want)
	}
	var payments int
	_ = e.db.QueryRow(`SELECT count(*) FROM payments WHERE rental_id=$1 AND status='success'`, rentalID).Scan(&payments)
	if payments != 1 {
		t.Fatalf("successful payments=%d, want exactly 1", payments)
	}
}

// Dad's report (Sep 5): "Я смог завести кладовку с существующим номером".
func TestDuplicateStorageNumberRejected(t *testing.T) {
	e := newTestEnv(t)
	for _, number := range []string{"1", " 1 "} {
		r := e.do(http.MethodPost, "/storages", e.adminA, map[string]any{"number": number, "area": 3, "floor": 1, "entrance": 1})
		if r.Code != http.StatusConflict {
			t.Fatalf("number %q: want 409, got %d %s", number, r.Code, r.Body)
		}
	}
	r := e.do(http.MethodPut, "/storages/"+e.storage2, e.adminA, map[string]any{"number": "1", "area": 5, "floor": 1, "entrance": 1})
	if r.Code != http.StatusConflict {
		t.Fatalf("rename to existing number: want 409, got %d %s", r.Code, r.Body)
	}
	// Same number in another JK is fine.
	if r := e.do(http.MethodPost, "/storages", e.adminB, map[string]any{"number": "2", "area": 3}); r.Code != http.StatusCreated {
		t.Fatalf("other JK: want 201, got %d %s", r.Code, r.Body)
	}
}

// Agreement 4.2 / dad (Sep 21): an occupied storage's parameters can't change until the paid period ends.
func TestOccupiedStorageParametersAreFrozen(t *testing.T) {
	e := newTestEnv(t)
	id := e.lockAs(e.resident, e.storageA, 1)
	e.do(http.MethodPost, "/payments", e.resident, map[string]any{"rental_id": id, "phone": "+77001112233"})
	e.webhook(paidEvent(e.paymentIDFor(id)), "valid")

	r := e.do(http.MethodPut, "/storages/"+e.storageA, e.adminA, map[string]any{"number": "1", "area": 50, "floor": 1, "entrance": 1})
	if r.Code != http.StatusConflict {
		t.Fatalf("change area of occupied storage: want 409, got %d %s", r.Code, r.Body)
	}
	var area float64
	_ = e.db.QueryRow(`SELECT area FROM storages WHERE id=$1`, e.storageA).Scan(&area)
	if area != 5 {
		t.Fatalf("area changed to %v", area)
	}
}

func TestEditingStorageKeepsPendingTransfer(t *testing.T) {
	e := newTestEnv(t)
	e.exec(`UPDATE storages SET transfer_to_iin='900101300002', transfer_until=now()+interval '14 days' WHERE id=$1`, e.storage2)
	r := e.do(http.MethodPut, "/storages/"+e.storage2, e.adminA, map[string]any{"number": "2", "area": 6, "floor": 1, "entrance": 1})
	if r.Code != http.StatusOK {
		t.Fatalf("edit: %d %s", r.Code, r.Body)
	}
	var iin string
	_ = e.db.QueryRow(`SELECT COALESCE(transfer_to_iin,'') FROM storages WHERE id=$1`, e.storage2).Scan(&iin)
	if iin != "900101300002" {
		t.Fatalf("transfer was wiped by edit (iin=%q)", iin)
	}
}
