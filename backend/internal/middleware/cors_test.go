package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/config"
)

func corsRouter(cfg config.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS(cfg))
	r.GET("/api/v1/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func preflight(r *gin.Engine, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodOptions, "http://api.test/api/v1/x", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCORSAllowsOnlyTheExactList(t *testing.T) {
	prod := config.Config{Env: config.EnvProduction, AppBaseURL: "https://kladovki.onrender.com/", CORSOrigins: []string{"https://kladovki.kz", " https://www.kladovki.kz "}}
	dev := config.Config{Env: config.EnvDevelopment}
	cases := []struct {
		name   string
		cfg    config.Config
		origin string
		ok     bool
	}{
		{"app base url", prod, "https://kladovki.onrender.com", true},
		{"capacitor android", prod, "https://localhost", true},
		{"capacitor ios", prod, "capacitor://localhost", true},
		{"extra from CORS_ORIGINS", prod, "https://kladovki.kz", true},
		{"extra trimmed", prod, "https://www.kladovki.kz", true},
		{"other render app", prod, "https://evil.onrender.com", false},
		{"suffix trick", prod, "https://kladovki.onrender.com.evil.kz", false},
		{"dev server in production", prod, "http://localhost:4200", false},
		{"plain http localhost", prod, "http://localhost", false},
		{"null origin", prod, "null", false},
		{"dev server in development", dev, "http://localhost:4200", true},
		{"127.0.0.1 in development", dev, "http://127.0.0.1:4200", true},
		{"render app in development", dev, "https://evil.onrender.com", false},
	}
	for _, tc := range cases {
		w := preflight(corsRouter(tc.cfg), tc.origin)
		got := w.Header().Get("Access-Control-Allow-Origin") == tc.origin && w.Code < 300
		if got != tc.ok {
			t.Errorf("%s (%s): allowed=%v (code %d, ACAO %q), want %v", tc.name, tc.origin, got, w.Code,
				w.Header().Get("Access-Control-Allow-Origin"), tc.ok)
		}
		if tc.ok && w.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Errorf("%s: credentials must be allowed for listed origins", tc.name)
		}
	}
}

func TestCORSLetsSameOriginAndNonBrowserRequestsThrough(t *testing.T) {
	r := corsRouter(config.Config{Env: config.EnvProduction})
	for _, origin := range []string{"", "http://api.test"} {
		req := httptest.NewRequest(http.MethodGet, "http://api.test/api/v1/x", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("origin %q: %d", origin, w.Code)
		}
	}
}
