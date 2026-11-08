package handlers

import (
	"net/http"
	"testing"

	"github.com/kaekkr/kladovki/internal/config"
)

// Stage: with FakePayments on, «Оплатить» skips Kaspi but goes through the
// same activation as a real paid webhook (payment row, dates, occupied storage).
func TestFakePaymentsActivateWithoutKaspi(t *testing.T) {
	e := newTestEnvWith(t, func(c *config.Config) { c.FakePayments = true })
	id := e.lockAs(e.resident, e.storageA, 2)

	r := e.do(http.MethodPost, "/payments", e.resident, map[string]any{"rental_id": id, "accepted_terms_version": termsV})
	if r.Code != http.StatusOK {
		t.Fatalf("pay: %d %s", r.Code, r.Body)
	}
	s := e.rentalState(id)
	if s.Status != "active" || s.Storage != "occupied" || s.Total != 10000 {
		t.Fatalf("rental after fake payment = %+v", s)
	}
	if n := e.count(`SELECT count(*) FROM payments WHERE rental_id=$1 AND status='success' AND amount=10000`, id); n != 1 {
		t.Fatalf("successful payment rows = %d", n)
	}
	e.apipay.mu.Lock()
	sent := len(e.apipay.invoices)
	e.apipay.mu.Unlock()
	if sent != 0 {
		t.Fatalf("no Kaspi invoice may be sent in fake mode, got %d", sent)
	}
	if n := kinds(e.notifications(e.resident))["payment_success"]; n != 1 {
		t.Fatalf("payment_success notifications = %d", n)
	}

	// Renewal also completes immediately.
	before := e.rentalState(id).EndsAt
	if code, _, _ := e.renew(e.resident, id, 1); code != http.StatusCreated {
		t.Fatalf("renew: %d", code)
	}
	if after := e.rentalState(id).EndsAt; !after.Equal(before.UTC().AddDate(0, 1, 0)) {
		t.Fatalf("ends_at %v → %v, want +1 month", before, after)
	}
	// Consent is still required.
	id2 := e.lockAs(e.other, e.storage2, 1)
	if r := e.do(http.MethodPost, "/payments", e.other, map[string]any{"rental_id": id2}); r.Code != http.StatusBadRequest {
		t.Fatalf("no consent: %d", r.Code)
	}
}
