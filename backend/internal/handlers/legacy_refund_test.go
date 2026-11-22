package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// Before 2026-11-22 choosing «refund» lowered total_paid at once. Such a
// request, still open at deploy, must not be deducted again on approval.
func TestLegacyTariffRefundDeductedOnce(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA) // total_paid 10000
	e.exec(`UPDATE rentals SET total_paid = 7500, price_per_month = 2500 WHERE id=$1`, id)
	reqID := uuid.NewString()
	e.exec(`INSERT INTO requests (id, jk_id, user_id, rental_id, type, status, amount, choice, old_price, new_price, refund_deducted)
		VALUES ($1,$2,$3,$4,'tariff_recalc','open',2500,'refund',5000,2500,true)`, reqID, e.jkA, e.resident, id)

	// Shown like a new refund: paid 10 000, 2 500 of it to be returned.
	if v := e.rentalView(e.resident, id); v.TotalPaid != 10000 || v.RefundPending != 2500 {
		t.Fatalf("before return: %+v, want total_paid 10000 + refund_pending 2500", v)
	}
	if r := e.do(http.MethodPatch, "/requests/"+reqID+"/resolve", e.adminA, map[string]any{"action": "approve"}); r.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", r.Code, r.Body)
	}
	if s := e.rentalState(id); s.Total != 7500 {
		t.Fatalf("total_paid after approval = %d, want 7500 (deducted once)", s.Total)
	}
	if v := e.rentalView(e.resident, id); v.TotalPaid != 7500 || v.RefundPending != 0 {
		t.Fatalf("after return: %+v, want 7500 / 0", v)
	}
}

// An unanswered request from before the deploy (marked by the migration) is
// answered with the new rule: deducted on approval, exactly once.
func TestLegacyAwaitingChoiceRefundDeductedOnApproval(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	reqID := uuid.NewString()
	e.exec(`INSERT INTO requests (id, jk_id, user_id, rental_id, type, status, amount, old_price, new_price, refund_deducted)
		VALUES ($1,$2,$3,$4,'tariff_recalc','awaiting_choice',2500,5000,2500,true)`, reqID, e.jkA, e.resident, id)
	if r := e.do(http.MethodPost, "/requests/"+reqID+"/choice", e.resident, map[string]any{"choice": "refund"}); r.Code != http.StatusOK {
		t.Fatalf("choice: %d %s", r.Code, r.Body)
	}
	if r := e.do(http.MethodPatch, "/requests/"+reqID+"/resolve", e.adminA, map[string]any{"action": "approve"}); r.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", r.Code, r.Body)
	}
	if s := e.rentalState(id); !near(s.Total, 7500) {
		t.Fatalf("total_paid = %d, want 7500", s.Total)
	}
}
