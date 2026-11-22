package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// Code review: a tiny room at a tiny tariff rounds to 0 ₸/month, and the
// extend formula divides by the new price.
func TestTariffDecreaseToZeroPriceAsksNothing(t *testing.T) {
	e := newTestEnv(t)
	e.exec(`UPDATE storages SET area = 0.4 WHERE id=$1`, e.storageA) // 400 ₸/month at 1000
	id := e.activeRental(e.resident, e.storageA)

	if r := e.do(http.MethodPut, "/jks/"+e.jkA+"/tariff", e.adminA, map[string]any{"amount": 1}); r.Code != http.StatusOK {
		t.Fatalf("set tariff: %d %s", r.Code, r.Body) // round(0.4 × 1) = 0
	}
	if q := e.awaitingTariffRequest(e.resident, id); q != "" {
		t.Fatal("a rental whose new price rounds to 0 must not get a tariff_recalc request")
	}
}

func TestTariffChoiceWithZeroNewPriceIsNot500(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	reqID := uuid.NewString()
	e.exec(`INSERT INTO requests (id, jk_id, user_id, rental_id, type, status, amount, old_price, new_price)
		VALUES ($1,$2,$3,$4,'tariff_recalc','awaiting_choice',5000,5000,0)`, reqID, e.jkA, e.resident, id)
	for _, choice := range []string{"extend", "refund"} {
		if r := e.do(http.MethodPost, "/requests/"+reqID+"/choice", e.resident, map[string]any{"choice": choice}); r.Code >= 500 {
			t.Fatalf("%s with new price 0: %d %s", choice, r.Code, r.Body)
		}
	}
	if s := e.rentalState(id); s.Price != 5000 {
		t.Fatalf("price_per_month = %d, must stay 5000", s.Price)
	}
}
