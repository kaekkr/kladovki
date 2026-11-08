package handlers

import (
	"math"
	"net/http"
	"testing"
	"time"
)

// activeRental creates a paid 2-month rental (5000 ₸/month, 10 000 ₸ total)
// that is exactly half used: started 30 days ago, ends in 30 days.
func (e *testEnv) activeRental(userID, storageID string) string {
	e.t.Helper()
	id := e.lockAs(userID, storageID, 2)
	e.do(http.MethodPost, "/payments", userID, map[string]any{"rental_id": id, "phone": "+77001112233"})
	if r := e.webhook(paidEvent(e.paymentIDFor(id)), "valid"); r.Code != http.StatusOK {
		e.t.Fatalf("activate: %d", r.Code)
	}
	e.exec(`UPDATE rentals SET starts_at = now() - interval '30 days', ends_at = now() + interval '30 days' WHERE id=$1`, id)
	return id
}

type request struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Amount   int64  `json:"amount"`
	RentalID string `json:"rental_id"`
	Choice   string `json:"choice"`
}

func (e *testEnv) myRequests(userID string) []request {
	e.t.Helper()
	r := e.do(http.MethodGet, "/requests/my", userID, nil)
	if r.Code != http.StatusOK {
		e.t.Fatalf("my requests: %d %s", r.Code, r.Body)
	}
	var out []request
	r.json(e.t, &out)
	return out
}

func near(a, b int64) bool { return math.Abs(float64(a-b)) <= 2 }

func TestTariffDecreasePreviewCountsRemainingValue(t *testing.T) {
	e := newTestEnv(t)
	e.activeRental(e.resident, e.storageA)

	r := e.do(http.MethodGet, "/jks/"+e.jkA+"/tariff/preview?amount=800", e.adminA, nil)
	if r.Code != http.StatusOK {
		t.Fatalf("preview: %d %s", r.Code, r.Body)
	}
	var p struct {
		AffectedCount   int64 `json:"affected_count"`
		TotalDifference int64 `json:"total_difference"`
	}
	r.json(t, &p)
	// (5000 − 4000) ₸/month × 2 months × ½ remaining = 1000 ₸
	if p.AffectedCount != 1 || !near(p.TotalDifference, 1000) {
		t.Fatalf("preview = %+v, want 1 rental / ~1000 ₸", p)
	}
}

func TestTariffIncreaseCreatesNoRequests(t *testing.T) {
	e := newTestEnv(t)
	e.activeRental(e.resident, e.storageA)
	if r := e.do(http.MethodPut, "/jks/"+e.jkA+"/tariff", e.adminA, map[string]any{"amount": 1500}); r.Code != http.StatusOK {
		t.Fatalf("set tariff: %d %s", r.Code, r.Body)
	}
	if got := e.myRequests(e.resident); len(got) != 0 {
		t.Fatalf("increase must not create requests, got %+v", got)
	}
}

func TestTariffDecreaseTenantChoosesExtension(t *testing.T) {
	e := newTestEnv(t)
	rentalID := e.activeRental(e.resident, e.storageA)
	var endsBefore time.Time
	_ = e.db.QueryRow(`SELECT ends_at FROM rentals WHERE id=$1`, rentalID).Scan(&endsBefore)

	if r := e.do(http.MethodPut, "/jks/"+e.jkA+"/tariff", e.adminA, map[string]any{"amount": 800}); r.Code != http.StatusOK {
		t.Fatalf("set tariff: %d %s", r.Code, r.Body)
	}
	reqs := e.myRequests(e.resident)
	if len(reqs) != 1 || reqs[0].Type != "tariff_recalc" || reqs[0].Status != "awaiting_choice" || !near(reqs[0].Amount, 1000) {
		t.Fatalf("requests = %+v", reqs)
	}

	r := e.do(http.MethodPost, "/requests/"+reqs[0].ID+"/choice", e.resident, map[string]any{"choice": "extend"})
	if r.Code != http.StatusOK {
		t.Fatalf("choice: %d %s", r.Code, r.Body)
	}

	var endsAfter time.Time
	var ppm int64
	_ = e.db.QueryRow(`SELECT ends_at, price_per_month FROM rentals WHERE id=$1`, rentalID).Scan(&endsAfter, &ppm)
	// 30 days left at 5000 buy 30 × 5000/4000 = 37.5 days at 4000 → +7.5 days.
	if got := endsAfter.Sub(endsBefore); got < 7*24*time.Hour || got > 8*24*time.Hour {
		t.Fatalf("extended by %v, want ~7.5 days", got)
	}
	if ppm != 4000 {
		t.Fatalf("price_per_month = %d, want 4000", ppm)
	}
	if reqs := e.myRequests(e.resident); reqs[0].Status != "done" || reqs[0].Choice != "extend" {
		t.Fatalf("after extend: %+v", reqs[0])
	}
	// Choosing twice is rejected.
	if r := e.do(http.MethodPost, "/requests/"+reqs[0].ID+"/choice", e.resident, map[string]any{"choice": "refund"}); r.Code != http.StatusConflict {
		t.Fatalf("second choice: want 409, got %d", r.Code)
	}
}

func TestTariffDecreaseRefundGoesToOSI(t *testing.T) {
	e := newTestEnv(t)
	e.activeRental(e.resident, e.storageA)
	e.do(http.MethodPut, "/jks/"+e.jkA+"/tariff", e.adminA, map[string]any{"amount": 800})
	req := e.myRequests(e.resident)[0]

	// Another resident cannot answer it.
	if r := e.do(http.MethodPost, "/requests/"+req.ID+"/choice", e.other, map[string]any{"choice": "refund"}); r.Code != http.StatusNotFound {
		t.Fatalf("foreign choice: want 404, got %d", r.Code)
	}
	if r := e.do(http.MethodPost, "/requests/"+req.ID+"/choice", e.resident, map[string]any{"choice": "refund"}); r.Code != http.StatusOK {
		t.Fatalf("choice: %d %s", r.Code, r.Body)
	}

	list := e.adminRequests(e.adminA)
	if len(list) != 1 || list[0].Status != "open" || list[0].Choice != "refund" || !near(list[0].Amount, 1000) {
		t.Fatalf("admin list = %+v", list)
	}
	if r := e.do(http.MethodGet, "/requests/jk/"+e.jkA, e.resident, nil); r.Code != http.StatusForbidden {
		t.Fatalf("resident admin list: want 403, got %d", r.Code)
	}
	if r := e.do(http.MethodPatch, "/requests/"+req.ID+"/resolve", e.adminB, map[string]any{"action": "approve"}); r.Code != http.StatusNotFound {
		t.Fatalf("other JK admin resolve: want 404, got %d", r.Code)
	}
	if r := e.do(http.MethodPatch, "/requests/"+req.ID+"/resolve", e.adminA, map[string]any{"action": "approve", "comment": "Вернули на Kaspi"}); r.Code != http.StatusOK {
		t.Fatalf("resolve: %d %s", r.Code, r.Body)
	}
	if got := e.myRequests(e.resident)[0]; got.Status != "done" {
		t.Fatalf("after resolve: %+v", got)
	}
}

func TestEarlyTerminationRequest(t *testing.T) {
	e := newTestEnv(t)
	rentalID := e.activeRental(e.resident, e.storageA)

	if r := e.do(http.MethodPost, "/rentals/"+rentalID+"/terminate", e.other, nil); r.Code != http.StatusNotFound {
		t.Fatalf("foreign terminate: want 404, got %d", r.Code)
	}
	r := e.do(http.MethodPost, "/rentals/"+rentalID+"/terminate", e.resident, map[string]any{"comment": "Переезжаю"})
	if r.Code != http.StatusCreated {
		t.Fatalf("terminate: %d %s", r.Code, r.Body)
	}
	if r := e.do(http.MethodPost, "/rentals/"+rentalID+"/terminate", e.resident, nil); r.Code != http.StatusConflict {
		t.Fatalf("duplicate terminate: want 409, got %d", r.Code)
	}

	list := e.adminRequests(e.adminA)
	// 10 000 ₸ paid, half the period unused → 5000 ₸
	if len(list) != 1 || list[0].Type != "early_termination" || !near(list[0].Amount, 5000) {
		t.Fatalf("admin list = %+v", list)
	}
	if r := e.do(http.MethodPatch, "/requests/"+list[0].ID+"/resolve", e.adminA, map[string]any{"action": "approve"}); r.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", r.Code, r.Body)
	}
	var rentalStatus, storageStatus string
	_ = e.db.QueryRow(`SELECT r.status, s.status FROM rentals r JOIN storages s ON s.id=r.storage_id WHERE r.id=$1`, rentalID).
		Scan(&rentalStatus, &storageStatus)
	if rentalStatus != "cancelled" || storageStatus != "free" {
		t.Fatalf("after approve: rental=%s storage=%s, want cancelled/free", rentalStatus, storageStatus)
	}
}

func TestEarlyTerminationRejectKeepsRental(t *testing.T) {
	e := newTestEnv(t)
	rentalID := e.activeRental(e.resident, e.storageA)
	e.do(http.MethodPost, "/rentals/"+rentalID+"/terminate", e.resident, nil)
	req := e.adminRequests(e.adminA)[0]
	if r := e.do(http.MethodPatch, "/requests/"+req.ID+"/resolve", e.adminA, map[string]any{"action": "reject", "comment": "Нет"}); r.Code != http.StatusOK {
		t.Fatalf("reject: %d", r.Code)
	}
	var status string
	_ = e.db.QueryRow(`SELECT status FROM rentals WHERE id=$1`, rentalID).Scan(&status)
	if status != "active" {
		t.Fatalf("rental status %s, want active", status)
	}
}

func TestPaidButTakenCreatesRefundRequest(t *testing.T) {
	e := newTestEnv(t)
	late := e.lockAs(e.resident, e.storageA, 2)
	e.do(http.MethodPost, "/payments", e.resident, map[string]any{"rental_id": late, "phone": "+77001112233"})
	// Lock expires; someone else books and pays.
	e.exec(`UPDATE rentals SET status='expired', ends_at=now() - interval '1 minute' WHERE id=$1`, late)
	e.exec(`UPDATE storages SET status='free' WHERE id=$1`, e.storageA)
	e.activeRental(e.other, e.storageA)

	if r := e.webhook(paidEvent(e.paymentIDFor(late)), "valid"); r.Code != http.StatusOK {
		t.Fatalf("webhook: %d", r.Code)
	}
	list := e.adminRequests(e.adminA)
	if len(list) != 1 || list[0].Type != "payment_refund" || list[0].Amount != 10000 || list[0].Status != "open" {
		t.Fatalf("admin list = %+v", list)
	}
}

func (e *testEnv) adminRequests(adminID string) []request {
	e.t.Helper()
	var jk string
	_ = e.db.QueryRow(`SELECT jk_id FROM users WHERE id=$1`, adminID).Scan(&jk)
	r := e.do(http.MethodGet, "/requests/jk/"+jk, adminID, nil)
	if r.Code != http.StatusOK {
		e.t.Fatalf("admin requests: %d %s", r.Code, r.Body)
	}
	var out []request
	r.json(e.t, &out)
	return out
}
