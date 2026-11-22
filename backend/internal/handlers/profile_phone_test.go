package handlers

import (
	"net/http"
	"testing"
)

// QA 2026-11-22: Kaspi invoices go to the profile phone, but residents could not change it.

func TestChangeOwnPhone(t *testing.T) {
	e := newTestEnv(t)
	var userID string
	_ = e.db.QueryRow(`SELECT id FROM users WHERE iin='900101300001'`).Scan(&userID)

	r := e.do(http.MethodPut, "/me/phone", userID, map[string]any{"phone": "8 (701) 555-44-33"})
	if r.Code != http.StatusOK {
		t.Fatalf("change phone: %d %s", r.Code, r.Body)
	}
	var phone string
	_ = e.db.QueryRow(`SELECT phone FROM users WHERE id=$1`, userID).Scan(&phone)
	if phone != "+77015554433" {
		t.Fatalf("phone = %q, want canonical +77015554433", phone)
	}

	var otherPhone string
	_ = e.db.QueryRow(`SELECT phone FROM users WHERE id <> $1 AND phone LIKE '+7%' LIMIT 1`, userID).Scan(&otherPhone)

	cases := []struct {
		name  string
		phone string
		want  int
	}{
		{"not a KZ number", "123", http.StatusBadRequest},
		{"someone else's", otherPhone, http.StatusConflict},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if r := e.do(http.MethodPut, "/me/phone", userID, map[string]any{"phone": c.phone}); r.Code != c.want {
				t.Fatalf("want %d, got %d %s", c.want, r.Code, r.Body)
			}
		})
	}

	if r := e.do(http.MethodPut, "/me/phone", "", map[string]any{"phone": "+77015554433"}); r.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: want 401, got %d", r.Code)
	}
}
