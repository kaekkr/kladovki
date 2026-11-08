package handlers

import (
	"net/http"
	"strings"
	"testing"
)

// Agreement п. 5.2 / ОСИ regulation п. 4.2–4.4: forced release must leave a record —
// reason, opening/inventory note, police, costs charged to the debtor — visible to the tenant.
func TestReleaseOverdueKeepsTerminationRecord(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.exec(`UPDATE rentals SET ends_at = now() - interval '70 days' WHERE id=$1`, id)
	path := "/rentals/" + id + "/release-overdue"

	if r := e.do(http.MethodPatch, path, e.adminA, map[string]any{"reason": " "}); r.Code != http.StatusBadRequest {
		t.Fatalf("no reason: %d, want 400", r.Code)
	}
	if r := e.do(http.MethodPatch, path, e.adminA, map[string]any{"reason": "x", "costs": -5}); r.Code != http.StatusBadRequest {
		t.Fatalf("negative costs: %d, want 400", r.Code)
	}
	body := map[string]any{
		"reason": "Задолженность 70 дней", "note": "Вскрыто 09.11, опись: 3 коробки, велосипед",
		"police": true, "costs": 15000,
	}
	if r := e.do(http.MethodPatch, path, e.adminA, body); r.Code != http.StatusOK {
		t.Fatalf("release: %d %s", r.Code, r.Body)
	}

	var rental struct {
		Status      string `json:"status"`
		Termination *struct {
			Reason string `json:"reason"`
			Note   string `json:"note"`
			Police bool   `json:"police"`
			Costs  int64  `json:"costs"`
		} `json:"termination"`
	}
	e.do(http.MethodGet, "/rentals/"+id, e.resident, nil).json(t, &rental)
	tm := rental.Termination
	if rental.Status != "expired" || tm == nil || tm.Reason != "Задолженность 70 дней" || !tm.Police || tm.Costs != 15000 ||
		!strings.Contains(tm.Note, "велосипед") {
		t.Fatalf("rental = %+v termination = %+v", rental, tm)
	}
	for _, n := range e.notifications(e.resident) {
		if n.Kind == "rental_cancelled" {
			if !strings.Contains(n.Body, "Задолженность 70 дней") || !strings.Contains(n.Body, "15 000") {
				t.Fatalf("notice must give reason and costs: %q", n.Body)
			}
			return
		}
	}
	t.Fatal("tenant not notified")
}

func TestReleaseOverdueErrorsAreNotLeaked(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA) // not overdue
	r := e.do(http.MethodPatch, "/rentals/"+id+"/release-overdue", e.adminA, map[string]any{"reason": "x"})
	if r.Code != http.StatusConflict || strings.Contains(string(r.Body), "repository") || strings.Contains(string(r.Body), "rental must") {
		t.Fatalf("got %d %s", r.Code, r.Body)
	}
}
