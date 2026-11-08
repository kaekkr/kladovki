package handlers

import (
	"net/http"
	"testing"
	"time"
)

func (e *testEnv) count(q string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.db.QueryRow(q, args...).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

func (e *testEnv) pay(userID, rentalID string) int {
	e.t.Helper()
	return e.do(http.MethodPost, "/payments", userID, map[string]any{
		"rental_id": rentalID, "phone": "+77001112233", "accepted_terms_version": termsV,
	}).Code
}

// A double tap on «Оплатить» must not send a second Kaspi invoice or extend the hold again.
func TestSecondPayRequestWhileInvoicePending(t *testing.T) {
	e := newTestEnv(t)
	id := e.lockAs(e.resident, e.storageA, 1)
	if code := e.pay(e.resident, id); code != http.StatusOK {
		t.Fatalf("first pay: %d", code)
	}
	hold := e.rentalState(id).EndsAt
	time.Sleep(20 * time.Millisecond)

	if code := e.pay(e.resident, id); code != http.StatusConflict {
		t.Fatalf("second pay: %d, want 409", code)
	}
	if n := e.count(`SELECT count(*) FROM payments WHERE rental_id=$1`, id); n != 1 {
		t.Fatalf("payments = %d, want 1", n)
	}
	e.apipay.mu.Lock()
	invoices := len(e.apipay.invoices)
	e.apipay.mu.Unlock()
	if invoices != 1 {
		t.Fatalf("Kaspi invoices sent = %d, want 1", invoices)
	}
	if !e.rentalState(id).EndsAt.Equal(hold) {
		t.Fatal("the hold must not be extended again")
	}
	// After the invoice is cancelled in Kaspi the resident can try again.
	e.webhook(map[string]any{"event": "invoice.status_changed", "invoice": map[string]any{"external_order_id": e.paymentIDFor(id), "status": "cancelled"}}, "valid")
	if code := e.pay(e.resident, id); code != http.StatusOK {
		t.Fatalf("pay after cancelled invoice: %d", code)
	}
}

// If two invoices for one booking were both paid, the second must become a refund, not revenue.
func TestBookingPaidTwiceOpensRefund(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	before := e.rentalState(id)
	e.exec(`INSERT INTO payments (id, rental_id, user_id, amount, provider, status, kind)
		VALUES ('dup-pay', $1, $2, 10000, 'apipay', 'pending', 'booking')`, id, e.resident)

	if r := e.webhook(paidEvent("dup-pay"), "valid"); r.Code != http.StatusOK {
		t.Fatalf("webhook: %d", r.Code)
	}
	if after := e.rentalState(id); after.Total != before.Total || !after.EndsAt.Equal(before.EndsAt) {
		t.Fatalf("rental changed: %+v → %+v", before, after)
	}
	list := e.adminRequests(e.adminA)
	if len(list) != 1 || list[0].Type != "payment_refund" || list[0].Amount != 10000 || list[0].Status != "open" {
		t.Fatalf("requests = %+v", list)
	}
	if n := kinds(e.notifications(e.resident))["refund_pending"]; n != 1 {
		t.Fatalf("refund_pending = %d", n)
	}
}

// ОСИ cancelling a paid rental owes the tenant the unused part (agreement 4.2: refunds go via a request).
func TestOSICancelPaidRentalOpensRefundAndNotifies(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA) // 10 000 ₸ paid; 30 days used, 30 left
	e.do(http.MethodPost, "/rentals/"+id+"/terminate", e.resident, map[string]any{"comment": "x"})

	if r := e.do(http.MethodPatch, "/rentals/"+id+"/cancel", e.adminA, nil); r.Code != http.StatusOK {
		t.Fatalf("cancel: %d %s", r.Code, r.Body)
	}
	if s := e.rentalState(id); s.Status != "cancelled" || s.Storage != "free" {
		t.Fatalf("state = %+v", s)
	}
	var refund *request
	for _, q := range e.adminRequests(e.adminA) {
		q := q
		if q.Type == "payment_refund" {
			refund = &q
		}
		if q.Type == "early_termination" {
			t.Fatalf("the tenant's open termination request must be closed, got %+v", q)
		}
	}
	if refund == nil || refund.Status != "open" || !near(refund.Amount, 5000) {
		t.Fatalf("refund = %+v", refund)
	}
	if n := e.count(`SELECT count(*) FROM requests WHERE rental_id=$1 AND type='early_termination' AND status='superseded'`, id); n != 1 {
		t.Fatalf("superseded termination requests = %d, want 1", n)
	}
	if n := kinds(e.notifications(e.resident))["rental_cancelled"]; n != 1 {
		t.Fatalf("tenant not notified: %d", n)
	}
}

func TestOSICancelOverdueRentalHasNothingToRefund(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.exec(`UPDATE rentals SET ends_at = now() - interval '3 days' WHERE id=$1`, id)
	if r := e.do(http.MethodPatch, "/rentals/"+id+"/cancel", e.adminA, nil); r.Code != http.StatusOK {
		t.Fatalf("cancel: %d", r.Code)
	}
	if n := e.count(`SELECT count(*) FROM requests WHERE type='payment_refund'`); n != 0 {
		t.Fatalf("refund requests = %d, want 0", n)
	}
	// Cancelling again / an expired rental is a conflict, not a server error.
	e.exec(`UPDATE rentals SET status='expired' WHERE id=$1`, id)
	if r := e.do(http.MethodPatch, "/rentals/"+id+"/cancel", e.adminA, nil); r.Code != http.StatusConflict {
		t.Fatalf("cancel expired: %d, want 409", r.Code)
	}
}

// Status is derived from rentals; the edit form must not be able to free an occupied room.
func TestStorageStatusCannotBeEditedByHand(t *testing.T) {
	e := newTestEnv(t)
	e.activeRental(e.resident, e.storageA)
	for _, status := range []string{"free", "foo"} {
		r := e.do(http.MethodPut, "/storages/"+e.storageA, e.adminA, map[string]any{
			"number": "1", "area": 5, "floor": 1, "entrance": 1, "status": status,
		})
		if r.Code != http.StatusOK {
			t.Fatalf("update with status %q: %d %s", status, r.Code, r.Body)
		}
		var st string
		_ = e.db.QueryRow(`SELECT status FROM storages WHERE id=$1`, e.storageA).Scan(&st)
		if st != "occupied" {
			t.Fatalf("status = %s after sending %q", st, status)
		}
	}
}
