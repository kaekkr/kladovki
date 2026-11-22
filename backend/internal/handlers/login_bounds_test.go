package handlers

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kaekkr/kladovki/internal/config"
)

// Code review: the login identifier went unchecked into the limiter map and the DB.
func TestLoginRejectsMalformedIdentifier(t *testing.T) {
	e := newTestEnvWith(t, func(c *config.Config) {
		c.RateLimits.LoginFailures = config.RateLimit{Requests: 1, Window: 15 * time.Minute}
	})
	for _, login := range []string{strings.Repeat("я", 256), "9001013\x0000001", "a\x00@b.kz"} {
		for i := 0; i < 2; i++ { // the second try would be 429 if the first had counted
			r := e.do(http.MethodPost, "/auth/login", "", map[string]any{"login": login, "password": "password123"})
			if r.Code != http.StatusBadRequest {
				t.Fatalf("login of %d bytes: want 400, got %d %s", len(login), r.Code, r.Body)
			}
		}
	}
	// 255 characters is still an ordinary (failed) login.
	r := e.do(http.MethodPost, "/auth/login", "", map[string]any{"login": strings.Repeat("я", 255), "password": "x"})
	if r.Code != http.StatusUnauthorized {
		t.Fatalf("255 chars: want 401, got %d", r.Code)
	}
}
