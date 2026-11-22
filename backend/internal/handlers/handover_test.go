package handlers

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// Paying the successor's booking during the rental closes the old one now and
// returns its unused part to the old tenant.
func TestHandoverDuringRentalRefundsOldTenant(t *testing.T) {
	e := newTestEnv(t)
	oldRental := e.setUpHandover()
	lock := e.lockAs(e.other, e.storageA, 1)
	e.pay(e.other, lock)
	e.webhook(paidEvent(e.paymentIDFor(lock)), "valid")

	old := e.rentalState(oldRental)
	// The DB clock (Docker VM) may run a few ms ahead of the test process.
	if old.Status != "expired" || old.EndsAt.After(time.Now().Add(time.Minute)) {
		t.Fatalf("old rental = %+v, want expired, ended now", old)
	}
	if s := e.rentalState(lock); s.Status != "active" {
		t.Fatalf("successor rental = %s, want active", s.Status)
	}
	var refund request
	for _, q := range e.myRequests(e.resident) {
		if q.Type == "payment_refund" {
			refund = q
		}
	}
	if refund.Status != "open" || !near(refund.Amount, 5000) {
		t.Fatalf("old tenant refund = %+v, want open ≈5000", refund)
	}
}

func TestTransferToYourselfIsRejected(t *testing.T) {
	e := newTestEnv(t)
	e.activeRental(e.resident, e.storageA)
	for _, who := range []string{e.resident, e.adminA} {
		r := e.do(http.MethodPatch, "/storages/"+e.storageA+"/transfer", who, map[string]any{"iin": "900101300001"})
		if r.Code != http.StatusBadRequest || !strings.Contains(string(r.Body), "текущему арендатору") {
			t.Fatalf("tenant's own IIN: %d %s, want 400", r.Code, r.Body)
		}
	}
}

// QA: an overdue tenant must not write the debt off by handing the storage over.
func TestHandoverRefusedWhileDebtUnpaid(t *testing.T) {
	e := newTestEnv(t)
	oldRental := e.setUpHandover()
	e.exec(`UPDATE rentals SET ends_at = now() - interval '20 days' WHERE id=$1`, oldRental)

	r := e.do(http.MethodPost, "/rentals/lock", e.other, map[string]any{"storage_id": e.storageA, "months": 1})
	if r.Code != http.StatusConflict || !strings.Contains(string(r.Body), "задолженност") {
		t.Fatalf("successor lock with debt: %d %s, want 409", r.Code, r.Body)
	}
	if s := e.rentalState(oldRental); s.Status != "active" {
		t.Fatalf("old rental = %s, must stay active (debt kept)", s.Status)
	}
	// The overdue tenant can still pay the debt by renewing.
	if code, _, _ := e.renew(e.resident, oldRental, 1); code != http.StatusCreated {
		t.Fatalf("overdue renew with transfer pending: %d, want 201", code)
	}
}

// A very late booking payment must not close another tenant's (overdue) rental.
func TestLatePaymentDoesNotEvictOverdueTenant(t *testing.T) {
	e := newTestEnv(t)
	late := e.lockAs(e.other, e.storageA, 1)
	e.pay(e.other, late)
	e.exec(`UPDATE rentals SET status='expired', ends_at = now() - interval '1 hour' WHERE id=$1`, late)
	e.exec(`UPDATE storages SET status='free' WHERE id=$1`, e.storageA)
	tenant := e.activeRental(e.resident, e.storageA)
	e.exec(`UPDATE rentals SET ends_at = now() - interval '3 days' WHERE id=$1`, tenant) // overdue

	e.webhook(paidEvent(e.paymentIDFor(late)), "valid")
	if s := e.rentalState(tenant); s.Status != "active" {
		t.Fatalf("overdue tenant's rental = %s, must stay active", s.Status)
	}
	if s := e.rentalState(late); s.Status == "active" {
		t.Fatal("late payment must not activate over another tenant")
	}
	found := false
	for _, q := range e.myRequests(e.other) {
		found = found || (q.Type == "payment_refund" && q.Status == "open")
	}
	if !found {
		t.Fatal("late payer must get a payment_refund")
	}
}
