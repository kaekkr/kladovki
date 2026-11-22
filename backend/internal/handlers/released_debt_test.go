package handlers

import (
	"net/http"
	"testing"
)

// QA: after «Расторгнуть и освободить» the debt must stay owed — the overdue
// time until the release plus the costs recorded at termination.
func TestDebtStaysAfterReleaseOverdue(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA) // 5000 ₸/month
	e.exec(`UPDATE rentals SET ends_at = now() - interval '75 days' + interval '1 hour' WHERE id=$1`, id)
	if r := e.do(http.MethodPatch, "/rentals/"+id+"/release-overdue", e.adminA,
		map[string]any{"reason": "Долг", "costs": 15000}); r.Code != http.StatusOK {
		t.Fatalf("release: %d %s", r.Code, r.Body)
	}
	// Time passes: the debt must not keep growing after the release.
	e.exec(`UPDATE terminations SET created_at = created_at - interval '10 days' WHERE rental_id=$1`, id)
	e.exec(`UPDATE rentals SET ends_at = ends_at - interval '10 days' WHERE id=$1`, id)

	_, rows := e.charges(e.adminA, e.jkA)
	var row chargeRow
	for _, c := range rows {
		if c.RentalID == id {
			row = c
		}
	}
	// 75 overdue days × 5000/30 = 12 500 + 15 000 costs
	if row.Debt != 27500 || row.OverdueDays != 75 || row.Status != "overdue" {
		t.Fatalf("released rental charge = %+v, want debt 27500, 75 days, overdue", row)
	}

	var stats struct {
		Debt           int64 `json:"debt"`
		DebtApartments int   `json:"debt_apartments"`
	}
	e.do(http.MethodGet, "/dashboard/stats", e.adminA, nil).json(t, &stats)
	if stats.Debt != 27500 || stats.DebtApartments != 1 {
		t.Fatalf("dashboard debt = %+v, want 27500 / 1", stats)
	}
}

// An early termination or an ОСИ cancel is not a debt.
func TestCancelledRentalHasNoDebt(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.do(http.MethodPatch, "/rentals/"+id+"/cancel", e.adminA, nil)
	_, rows := e.charges(e.adminA, e.jkA)
	for _, c := range rows {
		if c.RentalID == id && c.Debt != 0 {
			t.Fatalf("cancelled rental debt = %d", c.Debt)
		}
	}
}
