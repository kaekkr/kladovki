package handlers

import (
	"net/http"
	"testing"
)

type chargeRow struct {
	RentalID      string `json:"rental_id"`
	StorageNumber string `json:"storage_number"`
	Paid          int64  `json:"paid"`
	Debt          int64  `json:"debt"`
	Accrued       int64  `json:"accrued"`
	OverdueDays   int    `json:"overdue_days"`
	Status        string `json:"status"`
}

func (e *testEnv) charges(adminID, jkID string) (int, []chargeRow) {
	e.t.Helper()
	r := e.do(http.MethodGet, "/charges/jk/"+jkID, adminID, nil)
	var rows []chargeRow
	if r.Code == http.StatusOK {
		r.json(e.t, &rows)
	}
	return r.Code, rows
}

// Debt = time used after the paid period, not price × months − paid; unpaid bookings are not debts.
func TestChargesComputeDebtFromOverdueTime(t *testing.T) {
	e := newTestEnv(t)
	overdue := e.activeRental(e.resident, e.storageA) // 5 000 ₸/month, 10 000 paid
	e.exec(`UPDATE rentals SET ends_at = now() - interval '30 days' + interval '1 hour' WHERE id=$1`, overdue)
	current := e.activeRental(e.other, e.storage2) // paid, 30 days left
	// An abandoned booking (never paid) must not appear anywhere.
	e.exec(`INSERT INTO rentals (id, storage_id, user_id, jk_id, months, price_per_month, total_paid, starts_at, ends_at, status)
		VALUES ('abandoned', $1, $2, $3, 3, 5000, 0, now() - interval '1 day', now() - interval '1 day', 'expired')`, e.storageA, e.other, e.jkA)

	code, rows := e.charges(e.adminA, e.jkA)
	if code != http.StatusOK {
		t.Fatalf("charges: %d", code)
	}
	byID := map[string]chargeRow{}
	for _, r := range rows {
		byID[r.RentalID] = r
	}
	if _, ok := byID["abandoned"]; ok || len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	if o := byID[overdue]; o.Status != "overdue" || o.OverdueDays != 30 || o.Debt != 5000 || o.Paid != 10000 || o.Accrued != 15000 {
		t.Fatalf("overdue row = %+v", o)
	}
	if c := byID[current]; c.Status != "paid" || c.Debt != 0 {
		t.Fatalf("current row = %+v", c)
	}

	if code, _ := e.charges(e.adminB, e.jkA); code != http.StatusForbidden {
		t.Fatalf("other ЖК admin: %d", code)
	}
	if code, _ := e.charges(e.resident, e.jkA); code != http.StatusForbidden {
		t.Fatalf("resident: %d", code)
	}

	var stats struct {
		Debt           int64 `json:"debt"`
		DebtApartments int   `json:"debt_apartments"`
	}
	e.do(http.MethodGet, "/dashboard/stats", e.adminA, nil).json(t, &stats)
	if stats.Debt != 5000 || stats.DebtApartments != 1 {
		t.Fatalf("dashboard stats = %+v", stats)
	}

	var attention []struct {
		Type        string `json:"type"`
		RentalID    string `json:"rental_id"`
		DaysOverdue int    `json:"days_overdue"`
		Debt        int64  `json:"debt"`
	}
	e.do(http.MethodGet, "/dashboard/attention", e.adminA, nil).json(t, &attention)
	if len(attention) == 0 || attention[0].Type != "rental_overdue" || attention[0].RentalID != overdue ||
		attention[0].DaysOverdue != 30 || attention[0].Debt != 5000 {
		t.Fatalf("attention = %+v (overdue rentals must come first)", attention)
	}
}
