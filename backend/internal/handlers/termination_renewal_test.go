package handlers

import (
	"net/http"
	"strings"
	"testing"
)

// QA: a renewal paid while an early termination is open must not be lost.
func TestRenewRefusedWhileEarlyTerminationOpen(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.earlyTerminationAmount(e.resident, id)

	r := e.do(http.MethodPost, "/rentals/"+id+"/renew", e.resident, map[string]any{"months": 1, "phone": "+77001112233", "accepted_terms_version": termsV})
	if r.Code != http.StatusConflict || !strings.Contains(string(r.Body), "расторжение") {
		t.Fatalf("renew with open termination: %d %s, want 409 with a Russian message", r.Code, r.Body)
	}
	if n := e.count(`SELECT count(*) FROM payments WHERE rental_id=$1 AND kind='renewal'`, id); n != 0 {
		t.Fatalf("renewal payments = %d, want 0", n)
	}
}

func TestEarlyTerminationRefusedWhileRenewalPending(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	if code, _, _ := e.renew(e.resident, id, 1); code != http.StatusCreated {
		t.Fatalf("renew: %d", code)
	}
	r := e.do(http.MethodPost, "/rentals/"+id+"/terminate", e.resident, nil)
	if r.Code != http.StatusConflict || !strings.Contains(string(r.Body), "продлени") {
		t.Fatalf("terminate with pending renewal: %d %s, want 409", r.Code, r.Body)
	}
}

// An old unpaid renewal invoice (Kaspi push ignored) must not block termination
// forever; if it is paid after the request was filed, the money is refunded and
// the approved amount stays right.
func TestLateRenewalAfterTerminationRequestIsRefunded(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	_, payID, _ := e.renew(e.resident, id, 1)
	e.exec(`UPDATE payments SET created_at = now() - interval '2 hours' WHERE id=$1`, payID)
	before := e.rentalState(id)

	amount := e.earlyTerminationAmount(e.resident, id)
	if !near(amount, 5000) {
		t.Fatalf("termination amount = %d, want 5000", amount)
	}
	e.webhook(paidEvent(payID), "valid")

	if s := e.rentalState(id); !s.EndsAt.Equal(before.EndsAt) || s.Total != before.Total {
		t.Fatalf("late renewal must not extend a rental being terminated: %+v → %+v", before, s)
	}
	var refund, termination request
	for _, q := range e.adminRequests(e.adminA) {
		switch q.Type {
		case "payment_refund":
			refund = q
		case "early_termination":
			termination = q
		}
	}
	if refund.Status != "open" || refund.Amount != 5000 {
		t.Fatalf("renewal money must go to a payment_refund: %+v", refund)
	}
	if r := e.do(http.MethodPatch, "/requests/"+termination.ID+"/resolve", e.adminA, map[string]any{"action": "approve"}); r.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", r.Code, r.Body)
	}
	if s := e.rentalState(id); s.Status != "cancelled" {
		t.Fatalf("rental = %s, want cancelled", s.Status)
	}
	for _, q := range e.adminRequests(e.adminA) {
		if q.Type == "payment_refund" && q.Status != "open" {
			t.Fatalf("refund of the late renewal must stay open after approval: %+v", q)
		}
		if q.Type == "early_termination" && !near(q.Amount, 5000) {
			t.Fatalf("termination amount changed to %d", q.Amount)
		}
	}
}
