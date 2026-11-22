package handlers

import (
	"net/http"
	"strings"
	"sync"
	"testing"
)

// QA: one resident could hold every free storage. At most one unpaid hold per user.
func TestOneUnpaidHoldPerUser(t *testing.T) {
	e := newTestEnv(t)
	first := e.lockAs(e.resident, e.storageA, 1)

	r := e.do(http.MethodPost, "/rentals/lock", e.resident, map[string]any{"storage_id": e.storage2, "months": 1})
	if r.Code != http.StatusConflict || !strings.Contains(string(r.Body), "неоплаченная бронь") {
		t.Fatalf("second hold: %d %s, want 409", r.Code, r.Body)
	}
	// Another resident is not affected.
	if r := e.do(http.MethodPost, "/rentals/lock", e.other, map[string]any{"storage_id": e.storage2, "months": 1}); r.Code != http.StatusCreated {
		t.Fatalf("other resident: %d %s", r.Code, r.Body)
	}
	// Once the first hold has lapsed the resident may book again.
	e.exec(`UPDATE rentals SET ends_at = now() - interval '1 second' WHERE id=$1`, first)
	e.runExpiryWorker()
	if r := e.do(http.MethodPost, "/rentals/lock", e.resident, map[string]any{"storage_id": e.storageA, "months": 1}); r.Code != http.StatusCreated {
		t.Fatalf("after the hold lapsed: %d %s", r.Code, r.Body)
	}
}

func TestOneUnpaidHoldPerUserUnderRace(t *testing.T) {
	e := newTestEnv(t)
	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i, st := range []string{e.storageA, e.storage2} {
		wg.Add(1)
		go func(i int, st string) {
			defer wg.Done()
			codes[i] = e.do(http.MethodPost, "/rentals/lock", e.resident, map[string]any{"storage_id": st, "months": 1}).Code
		}(i, st)
	}
	wg.Wait()
	if n := e.count(`SELECT count(*) FROM rentals WHERE user_id=$1 AND status='locked'`, e.resident); n != 1 {
		t.Fatalf("holds = %d (codes %v), want exactly 1", n, codes)
	}
}
