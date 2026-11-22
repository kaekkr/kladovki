package middleware

import (
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/config"
)

// capacitorOrigins: the Android app (Capacitor 8 defaults to androidScheme
// "https", so the WebView origin is https://localhost) and iOS (capacitor://localhost).
var capacitorOrigins = []string{"https://localhost", "capacitor://localhost"}

// devOrigins: `ng serve`, allowed only with APP_ENV=development.
var devOrigins = []string{"http://localhost:4200", "http://127.0.0.1:4200"}

// AllowedOrigins is the exact CORS allow-list (credentials are allowed, so
// no wildcards and no suffix matching): APP_BASE_URL's origin, the Capacitor
// origins, CORS_ORIGINS, and the dev server in development only.
func AllowedOrigins(cfg config.Config) map[string]bool {
	allowed := map[string]bool{}
	add := func(raw string) {
		if o := originOf(raw); o != "" {
			allowed[o] = true
		} else if strings.TrimSpace(raw) != "" {
			log.Printf("WARNING: CORS: ignoring %q (want scheme://host[:port])", raw)
		}
	}
	add(cfg.AppBaseURL)
	for _, o := range capacitorOrigins {
		add(o)
	}
	for _, o := range cfg.CORSOrigins {
		add(o)
	}
	if cfg.Env == config.EnvDevelopment {
		for _, o := range devOrigins {
			add(o)
		}
	}
	return allowed
}

// originOf reduces a URL like https://kladovki.onrender.com/ to its origin.
func originOf(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Scheme + "://" + u.Host)
}

// CORS answers cross-origin requests only for AllowedOrigins. Same-origin
// requests (the website itself) and non-browser clients pass untouched.
func CORS(cfg config.Config) gin.HandlerFunc {
	allowed := AllowedOrigins(cfg)
	return cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool { return allowed[origin] },
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
			"X-Kladovki-Client", // mobile app asks for a bearer token
		},
		ExposeHeaders:    []string{"Content-Length", "Retry-After"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
