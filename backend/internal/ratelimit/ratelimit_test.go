package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestLimiter(limit int, window time.Duration) (*Limiter, *clock) {
	c := &clock{t: time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)}
	l := New(limit, window)
	l.now = c.now
	l.lastSweep = c.t
	return l, c
}

func TestAllow_LimitsPerKeyAndResetsAfterWindow(t *testing.T) {
	l, c := newTestLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !l.Allow("a") {
			t.Fatalf("hit %d should pass", i+1)
		}
	}
	if l.Allow("a") {
		t.Fatal("4th hit in the window must be refused")
	}
	if !l.Allow("b") {
		t.Fatal("another key has its own budget")
	}
	c.advance(time.Minute)
	if !l.Allow("a") {
		t.Fatal("a new window starts a new budget")
	}
}

func TestBlockedAddReset(t *testing.T) {
	l, c := newTestLimiter(2, 15*time.Minute)
	l.Add("u")
	if l.Blocked("u") {
		t.Fatal("1 failure < limit")
	}
	l.Add("u")
	if !l.Blocked("u") {
		t.Fatal("2 failures reach the limit")
	}
	if got := l.RetryAfter("u"); got != 15*time.Minute {
		t.Fatalf("RetryAfter = %v", got)
	}
	l.Reset("u")
	if l.Blocked("u") {
		t.Fatal("reset clears the key")
	}
	l.Add("u")
	l.Add("u")
	c.advance(15 * time.Minute)
	if l.Blocked("u") {
		t.Fatal("block expires with the window")
	}
}

func TestDisabledLimiterAllowsEverything(t *testing.T) {
	var nilLimiter *Limiter
	off := New(0, time.Minute)
	for _, l := range []*Limiter{nilLimiter, off} {
		for i := 0; i < 100; i++ {
			l.Add("k")
			if !l.Allow("k") || l.Blocked("k") {
				t.Fatal("a disabled limiter never refuses")
			}
		}
	}
}

func TestSweepDropsExpiredKeys(t *testing.T) {
	l, c := newTestLimiter(5, time.Minute)
	for _, k := range []string{"a", "b", "c"} {
		l.Allow(k)
	}
	c.advance(2 * time.Minute)
	l.Allow("d") // triggers the periodic sweep
	if n := l.size(); n != 1 {
		t.Fatalf("keys after sweep = %d, want 1", n)
	}
}

func TestMiddlewareAnswers429InRussian(t *testing.T) {
	gin.SetMode(gin.TestMode)
	l, _ := newTestLimiter(1, time.Minute)
	r := gin.New()
	r.POST("/x", PerIP(l), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	call := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/x", nil)
		req.RemoteAddr = "203.0.113.7:5000"
		r.ServeHTTP(w, req)
		return w
	}
	if w := call(); w.Code != http.StatusNoContent {
		t.Fatalf("first: %d", w.Code)
	}
	w := call()
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") == "" {
		t.Fatalf("second: %d %v", w.Code, w.Header())
	}
	if body := w.Body.String(); !strings.Contains(body, "Слишком много") || !strings.Contains(body, "rate_limited") {
		t.Fatalf("body = %s", body)
	}
}
