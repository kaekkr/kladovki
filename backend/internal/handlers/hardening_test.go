package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/kaekkr/kladovki/internal/repository"
)

func TestNotFoundIsNotAServerError(t *testing.T) {
	e := newTestEnv(t)
	cases := []struct{ method, path string }{
		{http.MethodGet, "/rentals/storage/" + e.storage2}, // free storage: no active rental
		{http.MethodGet, "/rentals/does-not-exist"},
		{http.MethodPatch, "/rentals/does-not-exist/cancel"},
		{http.MethodPatch, "/rentals/does-not-exist/force-release"},
	}
	for _, c := range cases {
		if r := e.do(c.method, c.path, e.adminA, nil); r.Code != http.StatusNotFound {
			t.Errorf("%s %s: %d %s, want 404", c.method, c.path, r.Code, r.Body)
		}
	}
}

func TestDeleteStorageWithHistoryIsAConflict(t *testing.T) {
	e := newTestEnv(t)
	e.activeRental(e.resident, e.storageA)
	r := e.do(http.MethodDelete, "/storages/"+e.storageA, e.adminA, nil)
	if r.Code != http.StatusConflict {
		t.Fatalf("delete with rentals: %d %s, want 409", r.Code, r.Body)
	}
	if r := e.do(http.MethodDelete, "/storages/"+e.storage2, e.adminA, nil); r.Code != http.StatusOK {
		t.Fatalf("delete unused: %d", r.Code)
	}
}

// Viewing a rental must not write to the database (and not before the permission check).
func TestViewingALapsedHoldDoesNotModifyData(t *testing.T) {
	e := newTestEnv(t)
	id := e.lockAs(e.resident, e.storageA, 1)
	e.exec(`UPDATE rentals SET ends_at = now() - interval '1 minute' WHERE id=$1`, id)

	if r := e.do(http.MethodGet, "/rentals/"+id, e.other, nil); r.Code != http.StatusNotFound {
		t.Fatalf("stranger: %d", r.Code)
	}
	var got struct {
		Status string `json:"status"`
	}
	e.do(http.MethodGet, "/rentals/"+id, e.resident, nil).json(t, &got)
	if got.Status != "expired" {
		t.Fatalf("lapsed hold shown as %q, want expired", got.Status)
	}
	if s := e.rentalState(id); s.Status != "locked" || s.Storage != "locked" {
		t.Fatalf("a GET changed the database: %+v", s)
	}
	e.runExpiryWorker()
	if s := e.rentalState(id); s.Status != "expired" || s.Storage != "free" {
		t.Fatalf("after worker: %+v", s)
	}
}

// Requests about a rental must close when the rental ends, so nothing is paid twice.
func TestRequestsCloseWhenRentalEnds(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.do(http.MethodPut, "/jks/"+e.jkA+"/tariff", e.adminA, map[string]any{"amount": 800}) // awaiting_choice
	e.do(http.MethodPost, "/rentals/"+id+"/terminate", e.resident, map[string]any{"comment": "x"})

	var term string
	for _, q := range e.adminRequests(e.adminA) {
		if q.Type == "early_termination" {
			term = q.ID
		}
	}
	if r := e.do(http.MethodPatch, "/requests/"+term+"/resolve", e.adminA, map[string]any{"action": "approve"}); r.Code != http.StatusOK {
		t.Fatalf("approve termination: %d", r.Code)
	}
	if n := e.count(`SELECT count(*) FROM requests WHERE rental_id=$1 AND type='tariff_recalc' AND status='superseded'`, id); n != 1 {
		t.Fatalf("tariff choice must be closed once the rental ended (superseded = %d)", n)
	}

	// Release for debt closes them too.
	id2 := e.activeRental(e.other, e.storage2)
	e.exec(`INSERT INTO requests (id, jk_id, user_id, rental_id, type, status, amount) VALUES ('rq-open', $1, $2, $3, 'early_termination', 'open', 100)`, e.jkA, e.other, id2)
	e.exec(`UPDATE rentals SET ends_at = now() - interval '20 days' WHERE id=$1`, id2)
	e.do(http.MethodPatch, "/rentals/"+id2+"/release-overdue", e.adminA, map[string]any{"reason": "долг"})
	if n := e.count(`SELECT count(*) FROM requests WHERE id='rq-open' AND status='superseded'`); n != 1 {
		t.Fatal("open request must be closed on release")
	}
}

// Choosing «refund» for a rental that has already ended must not open a refund.
func TestTariffChoiceOnEndedRental(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.do(http.MethodPut, "/jks/"+e.jkA+"/tariff", e.adminA, map[string]any{"amount": 800})
	req := e.myRequests(e.resident)[0]
	e.exec(`UPDATE rentals SET status='cancelled' WHERE id=$1`, id) // ended by other means, request left over
	if r := e.do(http.MethodPost, "/requests/"+req.ID+"/choice", e.resident, map[string]any{"choice": "refund"}); r.Code != http.StatusConflict {
		t.Fatalf("choice on ended rental: %d, want 409", r.Code)
	}
	if n := e.count(`SELECT count(*) FROM requests WHERE status='open'`); n != 0 {
		t.Fatalf("open requests = %d, want 0", n)
	}
}

// Revenue is grouped by Kazakhstan days (UTC+5): 03:00 Almaty belongs to that day, not the previous one.
func TestRevenueChartUsesKazakhstanDays(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	// 22:00 UTC yesterday = 03:00 today in Kazakhstan.
	e.exec(`UPDATE payments SET created_at = date_trunc('day', now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' - interval '2 hours' WHERE rental_id=$1`, id)
	var rev struct {
		Points []struct {
			Date   string `json:"date"`
			Amount int64  `json:"amount"`
		} `json:"points"`
	}
	e.do(http.MethodGet, "/dashboard/revenue", e.adminA, nil).json(t, &rev)
	var want string
	_ = e.db.QueryRow(`SELECT to_char(date_trunc('day', now() AT TIME ZONE 'UTC'), 'YYYY-MM-DD')`).Scan(&want)
	if len(rev.Points) != 1 || rev.Points[0].Date != want {
		t.Fatalf("points = %+v, want one point on %s", rev.Points, want)
	}
}

// Two server instances must not send the same SMS/Telegram message twice.
func TestDeliveriesAreClaimedOnce(t *testing.T) {
	e := newTestEnv(t)
	e.exec(`UPDATE users SET telegram_chat_id = '555' WHERE id=$1`, e.resident)
	e.activeRental(e.resident, e.storageA) // payment_success → telegram delivery
	repo := repository.New(e.db)
	first, err := repo.PendingDeliveries(context.Background(), 100)
	if err != nil || len(first) == 0 {
		t.Fatalf("first claim: %v %d", err, len(first))
	}
	second, err := repo.PendingDeliveries(context.Background(), 100)
	if err != nil || len(second) != 0 {
		t.Fatalf("second worker got %d already-claimed deliveries", len(second))
	}
}
