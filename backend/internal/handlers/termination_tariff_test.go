package handlers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Code review: early termination (amount = whole remaining value) and a
// tariff refund must never both pay out the same unused money.

func (e *testEnv) awaitingTariffRequest(userID, rentalID string) string {
	e.t.Helper()
	for _, q := range e.myRequests(userID) {
		if q.Type == "tariff_recalc" && q.Status == "awaiting_choice" && q.RentalID == rentalID {
			return q.ID
		}
	}
	return ""
}

// Termination first: a later tariff decrease asks nothing of this rental.
func TestTerminationThenTariffDecreaseCreatesNoRecalc(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	if amount := e.earlyTerminationAmount(e.resident, id); !near(amount, 5000) {
		t.Fatalf("termination amount = %d, want 5000", amount)
	}

	r := e.do(http.MethodGet, "/jks/"+e.jkA+"/tariff/preview?amount=500", e.adminA, nil)
	var p struct {
		AffectedCount int64 `json:"affected_count"`
	}
	r.json(t, &p)
	if p.AffectedCount != 0 {
		t.Fatalf("preview counts a rental being terminated: %+v", p)
	}
	if r := e.do(http.MethodPut, "/jks/"+e.jkA+"/tariff", e.adminA, map[string]any{"amount": 500}); r.Code != http.StatusOK {
		t.Fatalf("set tariff: %d %s", r.Code, r.Body)
	}
	if q := e.awaitingTariffRequest(e.resident, id); q != "" {
		t.Fatal("a rental with an open early termination must not get a tariff_recalc request")
	}
}

// Termination first, then a (legacy) unanswered tariff choice: refund and
// extend are refused, so the ОСИ never pays V plus a tariff difference.
func TestTariffChoiceRefusedWhileTerminationOpen(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.earlyTerminationAmount(e.resident, id)
	reqID := uuid.NewString()
	e.exec(`INSERT INTO requests (id, jk_id, user_id, rental_id, type, status, amount, old_price, new_price)
		VALUES ($1,$2,$3,$4,'tariff_recalc','awaiting_choice',2500,5000,2500)`, reqID, e.jkA, e.resident, id)

	for _, choice := range []string{"refund", "extend"} {
		r := e.do(http.MethodPost, "/requests/"+reqID+"/choice", e.resident, map[string]any{"choice": choice})
		if r.Code != http.StatusConflict || !strings.Contains(string(r.Body), "досрочное расторжение") {
			t.Fatalf("%s with open termination: %d %s, want 409", choice, r.Code, r.Body)
		}
	}
	var total int64
	_ = e.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM requests WHERE rental_id=$1 AND status='open'`, id).Scan(&total)
	if !near(total, 5000) {
		t.Fatalf("open money for the rental = %d, want only the 5000 termination", total)
	}
}

// Tariff choice first: termination waits for the answer; after a refund
// choice, termination returns only what is left, so the sum stays ≤ V.
func TestTerminationWaitsForTariffChoice(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA) // 5000 unused
	if r := e.do(http.MethodPut, "/jks/"+e.jkA+"/tariff", e.adminA, map[string]any{"amount": 500}); r.Code != http.StatusOK {
		t.Fatalf("set tariff: %d %s", r.Code, r.Body)
	}
	reqID := e.awaitingTariffRequest(e.resident, id)
	if reqID == "" {
		t.Fatal("no tariff_recalc request")
	}

	r := e.do(http.MethodPost, "/rentals/"+id+"/terminate", e.resident, nil)
	if r.Code != http.StatusConflict || !strings.Contains(string(r.Body), "Сначала выберите вариант перерасчёта") {
		t.Fatalf("terminate while choice is open: %d %s, want 409", r.Code, r.Body)
	}

	if r := e.do(http.MethodPost, "/requests/"+reqID+"/choice", e.resident, map[string]any{"choice": "refund"}); r.Code != http.StatusOK {
		t.Fatalf("choice: %d %s", r.Code, r.Body)
	}
	termination := e.earlyTerminationAmount(e.resident, id)
	var refund int64
	_ = e.db.QueryRow(`SELECT amount FROM requests WHERE id=$1`, reqID).Scan(&refund)
	if !near(refund, 2500) || !near(termination, 2500) || refund+termination > 5000 {
		t.Fatalf("refund %d + termination %d, want 2500 + 2500 ≤ 5000", refund, termination)
	}

	var termID string
	_ = e.db.QueryRow(`SELECT id FROM requests WHERE rental_id=$1 AND type='early_termination'`, id).Scan(&termID)
	if r := e.do(http.MethodPatch, "/requests/"+termID+"/resolve", e.adminA, map[string]any{"action": "approve"}); r.Code != http.StatusOK {
		t.Fatalf("approve termination: %d %s", r.Code, r.Body)
	}
	var status string
	_ = e.db.QueryRow(`SELECT status FROM requests WHERE id=$1`, reqID).Scan(&status)
	if status != "open" {
		t.Fatalf("tariff refund after termination = %s, must stay open (still owed)", status)
	}
}
