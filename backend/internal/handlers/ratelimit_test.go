package handlers

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kaekkr/kladovki/internal/config"
)

func TestLoginFailuresPerIdentifierAre429(t *testing.T) {
	e := newTestEnvWith(t, func(c *config.Config) {
		c.RateLimits.LoginFailures = config.RateLimit{Requests: 3, Window: 15 * time.Minute}
	})
	bad := map[string]any{"login": "900101300001", "password": "wrong-one"}
	for i := 0; i < 3; i++ {
		if r := e.do(http.MethodPost, "/auth/login", "", bad); r.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: want 401, got %d", i+1, r.Code)
		}
	}
	// Even the right password is refused now, and the message is Russian.
	r := e.do(http.MethodPost, "/auth/login", "", map[string]any{"login": "900101300001", "password": "password123"})
	if r.Code != http.StatusTooManyRequests || !strings.Contains(string(r.Body), "Слишком много попыток") {
		t.Fatalf("after 3 failures: %d %s", r.Code, r.Body)
	}
	// Another account is not affected.
	if r := e.do(http.MethodPost, "/auth/login", "", map[string]any{"login": "900101300002", "password": "password123"}); r.Code != http.StatusOK {
		t.Fatalf("other user: %d %s", r.Code, r.Body)
	}
	// The phone spelling of an account shares its counter with the canonical form.
	for _, login := range []string{"87000000004", "+7 700 000 00 04", "+77000000004"} {
		e.do(http.MethodPost, "/auth/login", "", map[string]any{"login": login, "password": "nope-nope"})
	}
	if r := e.do(http.MethodPost, "/auth/login", "", map[string]any{"login": "+77000000004", "password": "password123"}); r.Code != http.StatusTooManyRequests {
		t.Fatalf("phone spellings must share a counter: %d", r.Code)
	}
}

func TestPublicEndpointsAreLimitedPerIP(t *testing.T) {
	e := newTestEnvWith(t, func(c *config.Config) {
		c.RateLimits.Auth = config.RateLimit{Requests: 2, Window: time.Minute}
		c.RateLimits.Lead = config.RateLimit{Requests: 1, Window: time.Minute}
		c.RateLimits.Lock = config.RateLimit{Requests: 1, Window: time.Minute}
	})
	for i := 0; i < 2; i++ {
		e.do(http.MethodPost, "/auth/iin-status", "", map[string]any{"iin": "900101300001"})
	}
	for _, path := range []string{"/auth/iin-status", "/auth/login", "/auth/register"} {
		if r := e.do(http.MethodPost, path, "", map[string]any{}); r.Code != http.StatusTooManyRequests {
			t.Errorf("%s after the auth budget: want 429, got %d", path, r.Code)
		}
	}
	lead := map[string]any{"full_name": "Иван", "phone": "+77001112233"}
	if r := e.do(http.MethodPost, "/lead", "", lead); r.Code != http.StatusCreated {
		t.Fatalf("first lead: %d %s", r.Code, r.Body)
	}
	if r := e.do(http.MethodPost, "/lead", "", lead); r.Code != http.StatusTooManyRequests {
		t.Fatalf("second lead: want 429, got %d", r.Code)
	}
	e.do(http.MethodPost, "/rentals/lock", e.resident, map[string]any{"storage_id": e.storageA, "months": 1})
	if r := e.do(http.MethodPost, "/rentals/lock", e.resident, map[string]any{"storage_id": e.storage2, "months": 1}); r.Code != http.StatusTooManyRequests {
		t.Fatalf("second lock: want 429, got %d", r.Code)
	}
}
