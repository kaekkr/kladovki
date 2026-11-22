package handlers

import "testing"

// QA 2026-11-22 / owner decision: new ЖК hold a booking for 5 minutes by default.
func TestNewJKHoldsBookingFiveMinutes(t *testing.T) {
	e := newTestEnv(t)
	e.exec(`INSERT INTO jks (id, name, address, owner_id) SELECT 'jk-new', 'Новый', 'адрес', owner_id FROM jks WHERE id = $1`, e.jkA)
	var n int
	if err := e.db.QueryRow(`SELECT lock_minutes FROM jks WHERE id = 'jk-new'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("lock_minutes = %d, want 5", n)
	}
}
