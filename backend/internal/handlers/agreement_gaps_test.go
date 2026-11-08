package handlers

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// Agreement п. 5.2: at more than 2 months of debt the ОСИ may terminate — the ОСИ must know.
func TestDebtOverTwoMonthsAlsoAlertsOSI(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.exec(`UPDATE rentals SET starts_at = now() - interval '150 days', ends_at = now() - interval '62 days' WHERE id=$1`, id)
	if err := e.svc.RunReminders(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := kinds(e.notifications(e.adminA))["debt_warning"]; n != 1 {
		t.Fatalf("ОСИ debt_warning = %d, want 1", n)
	}
	if n := len(e.notifications(e.adminB)); n != 0 {
		t.Fatalf("other ЖК admin notified: %d", n)
	}
}

func TestReleaseOverdueTellsTenant(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.exec(`UPDATE rentals SET ends_at = now() - interval '20 days' WHERE id=$1`, id)
	if r := e.do(http.MethodPatch, "/rentals/"+id+"/release-overdue", e.adminA, map[string]any{"reason": "задолженность"}); r.Code != http.StatusOK {
		t.Fatalf("release: %d %s", r.Code, r.Body)
	}
	found := false
	for _, n := range e.notifications(e.resident) {
		if n.Kind == "rental_cancelled" && strings.Contains(n.Body, "задолженност") {
			found = true
		}
	}
	if !found {
		t.Fatalf("tenant not told about termination for debt: %+v", e.notifications(e.resident))
	}
}

// Agreement п. 6.2: passing the storage to someone else needs the ОСИ to know about it.
func TestHandoverIsReportedToOSI(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.exec(`UPDATE rentals SET ends_at = now() - interval '1 day' WHERE id=$1`, id)
	if r := e.do(http.MethodPatch, "/storages/"+e.storageA+"/transfer", e.resident, map[string]any{"iin": "900101300002"}); r.Code != http.StatusOK {
		t.Fatalf("transfer: %d %s", r.Code, r.Body)
	}
	n := e.notifications(e.adminA)
	if kinds(n)["transfer_new"] != 1 || !strings.Contains(n[0].Body, "900101300002") {
		t.Fatalf("ОСИ notifications = %+v", n)
	}
}

func TestTariffIncreaseNoticeNamesNewPrice(t *testing.T) {
	e := newTestEnv(t)
	e.activeRental(e.resident, e.storageA) // 5 m², 1 000 ₸/м² → 5 000 ₸
	e.do(http.MethodPut, "/jks/"+e.jkA+"/tariff", e.adminA, map[string]any{"amount": 1200})
	for _, n := range e.notifications(e.resident) {
		if n.Kind == "tariff_changed" {
			if !strings.Contains(n.Body, "6 000") || !strings.Contains(n.Body, "5 000") {
				t.Fatalf("notice must give old and new monthly price: %q", n.Body)
			}
			return
		}
	}
	t.Fatal("no tariff_changed notification")
}

// Money returned by the ОСИ is not income.
func TestDashboardIncomeIsNetOfRefunds(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA) // 10 000 paid, 30 of 60 days left
	e.do(http.MethodPost, "/rentals/"+id+"/terminate", e.resident, map[string]any{"comment": "x"})
	var termID string
	var refund int64
	for _, q := range e.adminRequests(e.adminA) {
		if q.Type == "early_termination" {
			termID, refund = q.ID, q.Amount
		}
	}
	if r := e.do(http.MethodPatch, "/requests/"+termID+"/resolve", e.adminA, map[string]any{"action": "approve"}); r.Code != http.StatusOK {
		t.Fatalf("approve: %d", r.Code)
	}
	var stats struct {
		Received int64 `json:"received"`
		Refunded int64 `json:"refunded"`
	}
	e.do(http.MethodGet, "/dashboard/stats", e.adminA, nil).json(t, &stats)
	if stats.Refunded != refund || stats.Received != 10000-refund {
		t.Fatalf("stats = %+v, refund %d", stats, refund)
	}
	var rev struct {
		Points []struct {
			Amount int64 `json:"amount"`
		} `json:"points"`
	}
	e.do(http.MethodGet, "/dashboard/revenue", e.adminA, nil).json(t, &rev)
	var sum int64
	for _, p := range rev.Points {
		sum += p.Amount
	}
	if sum != 10000-refund {
		t.Fatalf("revenue chart sum = %d, want %d", sum, 10000-refund)
	}
}
