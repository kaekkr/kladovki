package handlers

import (
	"testing"

	"github.com/google/uuid"
)

// Code review: with a NULL payer ИИН, `transfer_to_iin = payer.iin` was NULL
// and the occupied storage counted as free — a late payment evicted the tenant.
func TestLatePaymentByUserWithoutIINDoesNotTakeOccupiedStorage(t *testing.T) {
	e := newTestEnv(t)
	tenant := e.activeRental(e.other, e.storageA) // paid period still running
	e.exec(`UPDATE storages SET transfer_to_iin = '900101300099', transfer_until = now() + interval '40 days' WHERE id=$1`, e.storageA)
	e.exec(`UPDATE users SET iin = NULL WHERE id=$1`, e.resident)

	late, payID := uuid.NewString(), uuid.NewString()
	e.exec(`INSERT INTO rentals (id, storage_id, user_id, jk_id, months, price_per_month, total_paid, starts_at, ends_at, status)
		VALUES ($1,$2,$3,$4,1,5000,0,now() - interval '1 hour', now() - interval '50 minutes','locked')`,
		late, e.storageA, e.resident, e.jkA)
	e.exec(`INSERT INTO payments (id, rental_id, user_id, amount, provider, status, created_at)
		VALUES ($1,$2,$3,5000,'apipay','pending',now() - interval '1 hour')`, payID, late, e.resident)

	if r := e.webhook(paidEvent(payID), "valid"); r.Code != 200 {
		t.Fatalf("webhook: %d %s", r.Code, r.Body)
	}
	if s := e.rentalState(tenant); s.Status != "active" {
		t.Fatalf("current tenant's rental = %s, must stay active", s.Status)
	}
	if s := e.rentalState(late); s.Status == "active" {
		t.Fatal("late payer without ИИН took an occupied storage")
	}
	if n := e.count(`SELECT count(*) FROM requests WHERE type='payment_refund' AND rental_id=$1`, late); n != 1 {
		t.Fatalf("payment_refund requests = %d, want 1", n)
	}
}
