package handlers

import (
	"net/http"
	"strings"
	"testing"
)

// Code review: a double tap sent two renewal invoices, and both extended the rental when paid.
func TestSecondRenewalInvoiceIsRefusedWhileOneIsPending(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	code, first, _ := e.renew(e.resident, id, 1)
	if code != http.StatusCreated {
		t.Fatalf("first renewal: %d", code)
	}
	r := e.do(http.MethodPost, "/rentals/"+id+"/renew", e.resident,
		map[string]any{"months": 1, "phone": "+77001112233", "accepted_terms_version": termsV})
	if r.Code != http.StatusConflict || !strings.Contains(string(r.Body), "Счёт на продление уже отправлен") {
		t.Fatalf("second renewal: %d %s, want 409", r.Code, r.Body)
	}
	if n := e.count(`SELECT count(*) FROM payments WHERE rental_id=$1 AND kind='renewal'`, id); n != 1 {
		t.Fatalf("renewal payments = %d, want 1", n)
	}

	// A stale invoice (the Kaspi push was ignored) no longer blocks.
	e.exec(`UPDATE payments SET created_at = now() - interval '31 minutes' WHERE id=$1`, first)
	code, second, _ := e.renew(e.resident, id, 1)
	if code != http.StatusCreated {
		t.Fatalf("renewal after a stale invoice: %d", code)
	}
	// Neither does a failed one.
	e.exec(`UPDATE payments SET status = 'failed' WHERE id=$1`, second)
	if code, _, _ := e.renew(e.resident, id, 1); code != http.StatusCreated {
		t.Fatalf("renewal after a failed invoice: %d", code)
	}
}
