package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/config"
)

// ConfigureClientIP decides what c.ClientIP() (rate limits, consent evidence)
// believes. gin.Default() trusts every proxy, so any client could pick its
// own IP with X-Forwarded-For. Here only cfg.TrustedProxies may forward it;
// empty means ClientIP is the TCP peer. Behind a trusted proxy gin walks
// X-Forwarded-For from the right and returns the first untrusted hop — the
// address the proxy itself saw — so a spoofed left part is ignored.
// X-Real-IP is never read: a proxy may pass the client's value through.
func ConfigureClientIP(r *gin.Engine, cfg config.Config) error {
	r.ForwardedByClientIP = true
	r.RemoteIPHeaders = []string{"X-Forwarded-For"}
	r.TrustedPlatform = ""
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}
	return nil
}
