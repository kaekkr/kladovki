package handlers

import (
	"net/http"
	"testing"
)

// earlyTerminationAmount files an early termination and returns the amount
// offered back (the rental's remaining value at that moment).
func (e *testEnv) earlyTerminationAmount(userID, rentalID string) int64 {
	e.t.Helper()
	r := e.do(http.MethodPost, "/rentals/"+rentalID+"/terminate", userID, nil)
	if r.Code != http.StatusCreated {
		e.t.Fatalf("terminate: %d %s", r.Code, r.Body)
	}
	var q request
	r.json(e.t, &q)
	return q.Amount
}

// chooseTariff lowers the ЖК tariff and answers the tenant's tariff_recalc request.
func (e *testEnv) chooseTariff(userID string, tariff int64, choice string) request {
	e.t.Helper()
	if r := e.do(http.MethodPut, "/jks/"+e.jkA+"/tariff", e.adminA, map[string]any{"amount": tariff}); r.Code != http.StatusOK {
		e.t.Fatalf("set tariff: %d %s", r.Code, r.Body)
	}
	var req request
	for _, q := range e.myRequests(userID) {
		if q.Type == "tariff_recalc" && q.Status == "awaiting_choice" {
			req = q
		}
	}
	if req.ID == "" {
		e.t.Fatal("no tariff_recalc request")
	}
	if r := e.do(http.MethodPost, "/requests/"+req.ID+"/choice", userID, map[string]any{"choice": choice}); r.Code != http.StatusOK {
		e.t.Fatalf("choice: %d %s", r.Code, r.Body)
	}
	return req
}

// Remaining value must be what the unused time is worth: P×N for a fresh
// N-month rental at price P, 0 at the end, and fair after renewals at a new
// price and after a tariff decrease (extend or refund).
func TestRemainingValue(t *testing.T) {
	cases := []struct {
		name  string
		setup func(e *testEnv) string // returns the rental id
		want  int64
	}{
		{"fresh rental at start = price × months", func(e *testEnv) string {
			id := e.lockAs(e.resident, e.storageA, 2)
			e.pay(e.resident, id)
			e.webhook(paidEvent(e.paymentIDFor(id)), "valid")
			return id
		}, 10000},
		{"half used", func(e *testEnv) string { return e.activeRental(e.resident, e.storageA) }, 5000},
		{"at the end", func(e *testEnv) string {
			id := e.activeRental(e.resident, e.storageA)
			e.exec(`UPDATE rentals SET starts_at = now() - interval '60 days', ends_at = now() + interval '1 minute' WHERE id=$1`, id)
			return id
		}, 0},
		{"after renewal at a new price: unused old part + whole renewal", func(e *testEnv) string {
			id := e.activeRental(e.resident, e.storageA) // 5000 left
			e.exec(`UPDATE tariffs SET amount = 1200 WHERE jk_id=$1`, e.jkA)
			_, payID, _ := e.renew(e.resident, id, 1) // 6000
			e.webhook(paidEvent(payID), "valid")
			return id
		}, 11000},
		{"after an overdue renewal: overdue time is already used", func(e *testEnv) string {
			id := e.activeRental(e.resident, e.storageA)
			e.exec(`UPDATE rentals SET ends_at = now() - interval '10 days' WHERE id=$1`, id)
			_, payID, _ := e.renew(e.resident, id, 2) // 10000 for [old end, old end + 2 months]
			e.webhook(paidEvent(payID), "valid")
			return id
		}, -1}, // ~10 of ~61 days of the renewal are used → ≈ 10000 × 51/61
		{"after extend (QA: 15000 for 6 months, 3 used, tariff halved) = 7500", func(e *testEnv) string {
			e.exec(`UPDATE tariffs SET amount = 500 WHERE jk_id=$1`, e.jkA) // 2500 ₸/month
			id := e.lockAs(e.resident, e.storageA, 6)
			e.pay(e.resident, id)
			e.webhook(paidEvent(e.paymentIDFor(id)), "valid")
			e.exec(`UPDATE rentals SET starts_at = now() - interval '90 days', ends_at = now() + interval '90 days' WHERE id=$1`, id)
			e.chooseTariff(e.resident, 250, "extend")
			return id
		}, 7500},
		{"after refund = remaining value minus the refund", func(e *testEnv) string {
			id := e.activeRental(e.resident, e.storageA) // 5000 left at 5000/month
			e.chooseTariff(e.resident, 500, "refund")    // refund 2500
			return id
		}, 2500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEnv(t)
			id := tc.setup(e)
			got := e.earlyTerminationAmount(e.resident, id)
			want := tc.want
			if want < 0 { // overdue renewal case: computed in setup
				if got < 8300 || got > 8400 {
					t.Fatalf("remaining value = %d, want ≈ 10000 × 51/61 ≈ 8360", got)
				}
				return
			}
			if !near(got, want) {
				t.Fatalf("remaining value = %d, want %d", got, want)
			}
		})
	}
}

// The preview total must use the same remaining value as the requests it announces.
func TestTariffPreviewAfterExtend(t *testing.T) {
	e := newTestEnv(t)
	e.activeRental(e.resident, e.storageA)    // 5000 left
	e.chooseTariff(e.resident, 500, "extend") // still worth 5000, now at 2500/month

	r := e.do(http.MethodGet, "/jks/"+e.jkA+"/tariff/preview?amount=250", e.adminA, nil)
	var p struct {
		AffectedCount   int64 `json:"affected_count"`
		TotalDifference int64 `json:"total_difference"`
	}
	r.json(t, &p)
	// 5000 × (1 − 1250/2500) = 2500
	if p.AffectedCount != 1 || !near(p.TotalDifference, 2500) {
		t.Fatalf("preview = %+v, want 1 / 2500", p)
	}
	e.do(http.MethodPut, "/jks/"+e.jkA+"/tariff", e.adminA, map[string]any{"amount": 250})
	for _, q := range e.myRequests(e.resident) {
		if q.Status == "awaiting_choice" && !near(q.Amount, 2500) {
			t.Fatalf("request amount = %d, want 2500", q.Amount)
		}
	}
}
