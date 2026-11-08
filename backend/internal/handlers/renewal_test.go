package handlers

import (
	"net/http"
	"testing"
	"time"
)

type rentalState struct {
	Status  string
	EndsAt  time.Time
	Total   int64
	Months  int
	Price   int64
	Storage string
}

func (e *testEnv) rentalState(id string) rentalState {
	e.t.Helper()
	var s rentalState
	if err := e.db.QueryRow(`
		SELECT r.status, r.ends_at, r.total_paid, r.months, r.price_per_month, s.status
		FROM rentals r JOIN storages s ON s.id = r.storage_id WHERE r.id=$1`, id).
		Scan(&s.Status, &s.EndsAt, &s.Total, &s.Months, &s.Price, &s.Storage); err != nil {
		e.t.Fatalf("rental state: %v", err)
	}
	return s
}

// renew asks for an extension invoice and returns the payment id.
func (e *testEnv) renew(userID, rentalID string, months int) (int, string, int64) {
	e.t.Helper()
	r := e.do(http.MethodPost, "/rentals/"+rentalID+"/renew", userID, map[string]any{"months": months, "phone": "+77001112233"})
	var p struct {
		ID     string `json:"id"`
		Amount int64  `json:"amount"`
	}
	if r.Code == http.StatusCreated {
		r.json(e.t, &p)
	}
	return r.Code, p.ID, p.Amount
}

func TestRenewActiveRental_ExtendsFromCurrentEnd(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	before := e.rentalState(id)

	code, payID, amount := e.renew(e.resident, id, 3)
	if code != http.StatusCreated {
		t.Fatalf("renew: %d", code)
	}
	if amount != 3*5000 { // area 5 × tariff 1000 × 3 months
		t.Fatalf("amount = %d, want 15000", amount)
	}
	if got := e.rentalState(id); !got.EndsAt.Equal(before.EndsAt) || got.Total != before.Total {
		t.Fatal("rental must not change before payment")
	}

	for i := 0; i < 2; i++ { // second webhook must be a no-op
		if r := e.webhook(paidEvent(payID), "valid"); r.Code != http.StatusOK {
			t.Fatalf("webhook: %d", r.Code)
		}
	}
	after := e.rentalState(id)
	if want := before.EndsAt.UTC().AddDate(0, 3, 0); !after.EndsAt.Equal(want) {
		t.Fatalf("ends_at = %v, want %v", after.EndsAt, want)
	}
	if after.Total != before.Total+15000 || after.Months != before.Months+3 || after.Status != "active" || after.Storage != "occupied" {
		t.Fatalf("after = %+v (before %+v)", after, before)
	}
}

func TestRenewOverdueRental_CoversDebtFirst(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.exec(`UPDATE rentals SET ends_at = now() - interval '10 days' WHERE id=$1`, id)
	before := e.rentalState(id)

	_, payID, _ := e.renew(e.resident, id, 1)
	e.webhook(paidEvent(payID), "valid")

	if after := e.rentalState(id); !after.EndsAt.Equal(before.EndsAt.UTC().AddDate(0, 1, 0)) {
		t.Fatalf("overdue renewal must extend from the old end: %v → %v", before.EndsAt, after.EndsAt)
	}
}

func TestRenewUsesCurrentTariff(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.exec(`UPDATE tariffs SET amount = 1200 WHERE jk_id=$1`, e.jkA)

	_, payID, amount := e.renew(e.resident, id, 1)
	if amount != 6000 {
		t.Fatalf("amount = %d, want 6000 (5 m² × 1200)", amount)
	}
	e.webhook(paidEvent(payID), "valid")
	if p := e.rentalState(id).Price; p != 6000 {
		t.Fatalf("price_per_month = %d, want 6000", p)
	}
}

func TestRenewRules(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)

	if code, _, _ := e.renew(e.other, id, 1); code != http.StatusNotFound {
		t.Fatalf("other user: %d, want 404", code)
	}
	if code, _, _ := e.renew(e.resident, id, 0); code != http.StatusBadRequest {
		t.Fatalf("0 months: %d, want 400", code)
	}
	locked := e.lockAs(e.resident, e.storage2, 1)
	if code, _, _ := e.renew(e.resident, locked, 1); code != http.StatusConflict {
		t.Fatalf("unpaid booking: %d, want 409", code)
	}
	e.exec(`UPDATE storages SET transfer_to_iin='900101300002', transfer_until=now() + interval '40 days' WHERE id=$1`, e.storageA)
	if code, _, _ := e.renew(e.resident, id, 1); code != http.StatusConflict {
		t.Fatalf("transfer pending: %d, want 409", code)
	}
}

func TestRenewPaidAfterRentalEnded_CreatesRefund(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	_, payID, _ := e.renew(e.resident, id, 2)
	e.exec(`UPDATE rentals SET status='cancelled' WHERE id=$1`, id) // ОСИ cancelled meanwhile

	e.webhook(paidEvent(payID), "valid")

	if s := e.rentalState(id); s.Status != "cancelled" {
		t.Fatalf("status = %s, must stay cancelled", s.Status)
	}
	list := e.adminRequests(e.adminA)
	if len(list) != 1 || list[0].Type != "payment_refund" || list[0].Amount != 10000 {
		t.Fatalf("requests = %+v", list)
	}
	if n := kinds(e.notifications(e.resident))["refund_pending"]; n != 1 {
		t.Fatalf("resident refund_pending notifications = %d", n)
	}
}

func TestLockDurationSetting(t *testing.T) {
	e := newTestEnv(t)
	path := "/jks/" + e.jkA + "/rental-settings"

	var s struct {
		LockMinutes int `json:"lock_duration_minutes"`
	}
	e.do(http.MethodGet, path, e.resident, nil).json(t, &s)
	if s.LockMinutes != 2 {
		t.Fatalf("default = %d, want 2", s.LockMinutes)
	}
	if r := e.do(http.MethodPut, path, e.resident, map[string]any{"lock_duration_minutes": 5}); r.Code != http.StatusForbidden {
		t.Fatalf("resident: %d", r.Code)
	}
	if r := e.do(http.MethodPut, path, e.adminB, map[string]any{"lock_duration_minutes": 5}); r.Code != http.StatusForbidden {
		t.Fatalf("other ЖК admin: %d", r.Code)
	}
	for _, bad := range []int{0, 61} {
		if r := e.do(http.MethodPut, path, e.adminA, map[string]any{"lock_duration_minutes": bad}); r.Code != http.StatusBadRequest {
			t.Fatalf("%d minutes: %d, want 400", bad, r.Code)
		}
	}
	if r := e.do(http.MethodPut, path, e.adminA, map[string]any{"lock_duration_minutes": 5}); r.Code != http.StatusOK {
		t.Fatalf("save: %d %s", r.Code, r.Body)
	}

	id := e.lockAs(e.resident, e.storageA, 1)
	hold := time.Until(e.rentalState(id).EndsAt)
	if hold < 4*time.Minute || hold > 5*time.Minute+5*time.Second {
		t.Fatalf("hold = %v, want ≈5m", hold)
	}
}
