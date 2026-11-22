package handlers

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kaekkr/kladovki/internal/config"
)

// QA 2026-11-22: Kaspi invoices go to the profile phone, but residents could not change it.

func TestChangeOwnPhone(t *testing.T) {
	e := newTestEnv(t)
	var userID string
	_ = e.db.QueryRow(`SELECT id FROM users WHERE iin='900101300001'`).Scan(&userID)

	r := e.do(http.MethodPut, "/me/phone", userID, map[string]any{"phone": "8 (701) 555-44-33", "current_password": "password123"})
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
			body := map[string]any{"phone": c.phone, "current_password": "password123"}
			if r := e.do(http.MethodPut, "/me/phone", userID, body); r.Code != c.want {
				t.Fatalf("want %d, got %d %s", c.want, r.Code, r.Body)
			}
		})
	}

	if r := e.do(http.MethodPut, "/me/phone", "", map[string]any{"phone": "+77015554433"}); r.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: want 401, got %d", r.Code)
	}
}

// Code review: a hijacked session must not be able to redirect Kaspi invoices.
func TestChangePhoneNeedsCurrentPassword(t *testing.T) {
	e := newTestEnv(t)
	if r := e.do(http.MethodPut, "/me/phone", e.resident, map[string]any{"phone": "+77015554433"}); r.Code != http.StatusBadRequest {
		t.Fatalf("no current_password: want 400, got %d %s", r.Code, r.Body)
	}
	r := e.do(http.MethodPut, "/me/phone", e.resident, map[string]any{"phone": "+77015554433", "current_password": "wrong-one"})
	if r.Code != http.StatusForbidden || !strings.Contains(string(r.Body), "пароль") {
		t.Fatalf("wrong password: want 403, got %d %s", r.Code, r.Body)
	}
	var phone string
	_ = e.db.QueryRow(`SELECT phone FROM users WHERE id=$1`, e.resident).Scan(&phone)
	if phone == "+77015554433" {
		t.Fatal("phone changed with a wrong password")
	}
}

func TestChangePhoneIsRateLimitedPerIP(t *testing.T) {
	e := newTestEnvWith(t, func(c *config.Config) {
		c.RateLimits.Auth = config.RateLimit{Requests: 2, Window: time.Minute}
	})
	body := map[string]any{"phone": "+77015554433", "current_password": "wrong-one"}
	for i := 0; i < 2; i++ {
		e.do(http.MethodPut, "/me/phone", e.resident, body)
	}
	if r := e.do(http.MethodPut, "/me/phone", e.resident, body); r.Code != http.StatusTooManyRequests {
		t.Fatalf("after the auth budget: want 429, got %d", r.Code)
	}
}
