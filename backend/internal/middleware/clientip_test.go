package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/config"
)

func clientIP(t *testing.T, proxies []string, remote string, headers map[string]string) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := ConfigureClientIP(r, config.Config{TrustedProxies: proxies}); err != nil {
		t.Fatalf("configure: %v", err)
	}
	r.GET("/ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })
	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = remote
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Body.String()
}

func TestClientIPIgnoresForwardedHeadersByDefault(t *testing.T) {
	got := clientIP(t, nil, "10.1.2.3:4000", map[string]string{"X-Forwarded-For": "1.2.3.4", "X-Real-IP": "5.6.7.8"})
	if got != "10.1.2.3" {
		t.Fatalf("ClientIP = %s, want the TCP peer 10.1.2.3", got)
	}
}

func TestClientIPBehindTrustedProxyTakesTheHopTheProxySaw(t *testing.T) {
	private := []string{"10.0.0.0/8"}
	// The client spoofs the left part; the proxy appends the address it saw.
	if got := clientIP(t, private, "10.1.2.3:4000", map[string]string{"X-Forwarded-For": "6.6.6.6, 203.0.113.9"}); got != "203.0.113.9" {
		t.Fatalf("ClientIP = %s, want 203.0.113.9", got)
	}
	// X-Real-IP is never believed (a proxy may pass the client's value through).
	if got := clientIP(t, private, "10.1.2.3:4000", map[string]string{"X-Real-IP": "6.6.6.6"}); got != "10.1.2.3" {
		t.Fatalf("X-Real-IP must be ignored: %s", got)
	}
	// A direct connection from outside the trusted range can't forge anything.
	if got := clientIP(t, private, "198.51.100.1:4000", map[string]string{"X-Forwarded-For": "6.6.6.6"}); got != "198.51.100.1" {
		t.Fatalf("untrusted peer: %s", got)
	}
}

func TestConfigureClientIPRejectsBadCIDR(t *testing.T) {
	if err := ConfigureClientIP(gin.New(), config.Config{TrustedProxies: []string{"not-a-cidr"}}); err == nil {
		t.Fatal("want an error for an invalid TRUSTED_PROXIES entry")
	}
}
