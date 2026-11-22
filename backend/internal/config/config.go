package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	DBPath              string
	JWTSecret           string
	JWTAccessTTL        time.Duration
	CookieSecure        bool
	CookieName          string
	Env                 string
	APIPrefix           string
	ApiPayKey           string
	ApiPayBaseURL       string
	ApiPayWebhookSecret string
	// ApiPayLive: real money mode — sandbox webhook events are then ignored.
	ApiPayLive bool
	// FakePayments: «Оплатить» / «Продлить» complete immediately without Kaspi
	// (STAGE BRANCH: on unless FAKE_PAYMENTS=false). Never merge this default
	// into main — production must take real payments.
	FakePayments bool
	// SuperadminEmail/SuperadminPassword bootstrap the first platform operator
	// at startup (both must be set). The password is never logged.
	SuperadminEmail    string
	SuperadminPassword string
	// RateLimits: the zero value disables every limit (tests rely on that);
	// Load() fills in DefaultRateLimits().
	RateLimits RateLimits

	// AppBaseURL (APP_BASE_URL) is the site's public URL; its origin is allowed by CORS.
	AppBaseURL string
	// CORSOrigins (CORS_ORIGINS, comma-separated) are extra exact origins.
	CORSOrigins []string
	// TrustedProxies: CIDRs/IPs whose X-Forwarded-For is believed for c.ClientIP().
	// Empty = trust no proxy (ClientIP is the TCP peer). See resolveTrustedProxies.
	TrustedProxies []string
}

// privateNetworks are the ranges a platform load balancer (Render) reaches the app from.
var privateNetworks = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8", "::1/128", "fc00::/7"}

// resolveTrustedProxies: TRUSTED_PROXIES (comma-separated CIDRs/IPs), plus the
// private ranges when TRUST_RENDER_PROXY=true. Nothing set → trust nobody.
func resolveTrustedProxies(list string, trustRender bool) []string {
	out := splitList(list)
	if trustRender {
		out = append(out, privateNetworks...)
	}
	return out
}

// splitList splits a comma-separated env value, dropping blanks.
func splitList(v string) []string {
	var out []string
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// RateLimit allows Requests events per Window; Requests <= 0 disables it.
type RateLimit struct {
	Requests int
	Window   time.Duration
}

type RateLimits struct {
	// Auth: per client IP, shared by /auth/login, /auth/iin-status, /auth/register.
	Auth RateLimit
	// Lead: per client IP, POST /lead (landing page form).
	Lead RateLimit
	// Lock: per client IP, POST /rentals/lock (holds a storage room).
	Lock RateLimit
	// LoginFailures: failed passwords per login identifier, then 429.
	LoginFailures RateLimit
}

// DefaultRateLimits are generous for people and tight for scripts.
func DefaultRateLimits() RateLimits {
	return RateLimits{
		Auth:          RateLimit{Requests: 30, Window: time.Minute},
		Lead:          RateLimit{Requests: 10, Window: time.Hour},
		Lock:          RateLimit{Requests: 20, Window: 10 * time.Minute},
		LoginFailures: RateLimit{Requests: 10, Window: 15 * time.Minute},
	}
}

// envFiles are read in order; a value from an earlier file wins, and real
// environment variables always win over files. /etc/secrets is where Render
// mounts "Secret Files" — upload your .env.local there instead of typing
// every variable into the dashboard.
var envFiles = []string{".env.local", "/etc/secrets/.env.local", ".env", "/etc/secrets/.env"}

// loadEnvFiles loads the files that exist and returns their paths.
func loadEnvFiles(paths ...string) []string {
	var loaded []string
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		vals, err := godotenv.Read(p)
		if err != nil {
			log.Printf("WARNING: could not parse %s: %v", p, err)
			continue
		}
		for k, v := range vals {
			// Empty placeholders (KEY=) must not hide a value from a later file.
			if _, set := os.LookupEnv(k); v != "" && !set {
				_ = os.Setenv(k, v)
			}
		}
		loaded = append(loaded, p)
	}
	return loaded
}

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
	// devJWTSecret is public (it is in this repo): only APP_ENV=development may use it.
	devJWTSecret = "dev-secret-change-me-in-production-32chars"
)

// resolveEnv picks APP_ENV (or ENV). Unset means production: forgetting the
// variable on a server must never switch on dev shortcuts (public JWT secret,
// unsigned webhooks, payment without Kaspi, non-Secure cookie).
func resolveEnv(appEnv, env string) (string, error) {
	v := appEnv
	if v == "" {
		v = env
	}
	switch v {
	case "":
		return EnvProduction, nil
	case EnvDevelopment, EnvProduction:
		return v, nil
	}
	return "", fmt.Errorf("APP_ENV=%q is not supported: use exactly %q or %q (unset means %q)",
		v, EnvDevelopment, EnvProduction, EnvProduction)
}

// resolveJWTSecret returns JWT_SECRET; the built-in fallback is allowed only in development.
func resolveJWTSecret(env, secret string) (string, error) {
	if secret != "" {
		return secret, nil
	}
	if env == EnvDevelopment {
		return devJWTSecret, nil
	}
	return "", errors.New("JWT_SECRET is required unless APP_ENV=development")
}

func Load() Config {
	if loaded := loadEnvFiles(envFiles...); len(loaded) > 0 {
		log.Printf("Info: env loaded from %v", loaded)
	} else {
		log.Println("Info: no .env files found, relying on system environment variables")
	}

	env, err := resolveEnv(os.Getenv("APP_ENV"), os.Getenv("ENV"))
	if err != nil {
		log.Fatalf("CRITICAL: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "postgres://kladovki:kladovki_secret@localhost:5432/kladovki?sslmode=disable"
	}

	secret, err := resolveJWTSecret(env, os.Getenv("JWT_SECRET"))
	if err != nil {
		log.Fatalf("CRITICAL: %v", err)
	}
	if secret == devJWTSecret {
		log.Println("WARNING: Using default JWT secret (APP_ENV=development). Do not use in production!")
	}

	ttlHours := 24
	if v := os.Getenv("JWT_TTL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttlHours = n
		}
	}

	cookieName := os.Getenv("COOKIE_NAME")
	if cookieName == "" {
		cookieName = "access_token"
	}

	// Auto-enforces secure cookies in production unless explicitly overridden
	secure := os.Getenv("COOKIE_SECURE") == "true" || os.Getenv("COOKIE_SECURE") == "1"
	if !secure && env == "production" {
		secure = true
	}

	apiPrefix := os.Getenv("API_PREFIX")
	if apiPrefix == "" {
		apiPrefix = "/api/v1"
	}

	// Настройки ApiPay
	apiPayKey := os.Getenv("APIPAY_KEY")
	if apiPayKey == "" && env == "production" {
		log.Println("WARNING: APIPAY_KEY is not set — Kaspi payments will fail")
	}

	apiPayBaseURL := os.Getenv("APIPAY_BASE_URL")
	if apiPayBaseURL == "" {
		apiPayBaseURL = "https://api.apipay.kz/api"
	}

	apiPayWebhookSecret := os.Getenv("APIPAY_WEBHOOK_SECRET")
	if apiPayWebhookSecret == "" && env != "development" {
		log.Println("WARNING: APIPAY_WEBHOOK_SECRET is not set — payment webhooks will be rejected")
	}

	return Config{
		Port:          port,
		DBPath:        dbPath,
		JWTSecret:     secret,
		JWTAccessTTL:  time.Duration(ttlHours) * time.Hour,
		CookieSecure:  secure,
		CookieName:    cookieName,
		Env:           env,
		APIPrefix:     apiPrefix,
		ApiPayKey:     apiPayKey,
		ApiPayBaseURL: apiPayBaseURL,

		ApiPayWebhookSecret: apiPayWebhookSecret,
		ApiPayLive:          os.Getenv("APIPAY_LIVE") == "true",
		FakePayments:        fakePayments(),

		SuperadminEmail:    os.Getenv("SUPERADMIN_EMAIL"),
		SuperadminPassword: os.Getenv("SUPERADMIN_PASSWORD"),

		RateLimits: DefaultRateLimits(),

		AppBaseURL:     os.Getenv("APP_BASE_URL"),
		CORSOrigins:    splitList(os.Getenv("CORS_ORIGINS")),
		TrustedProxies: resolveTrustedProxies(os.Getenv("TRUSTED_PROXIES"), os.Getenv("TRUST_RENDER_PROXY") == "true"),
	}
}

// fakePayments: stage branch default is ON (test payments), FAKE_PAYMENTS=false disables it.
func fakePayments() bool {
	on := os.Getenv("FAKE_PAYMENTS") != "false"
	if on {
		log.Println("WARNING: FAKE_PAYMENTS is on — payments complete without Kaspi (stage/testing only)")
	}
	return on
}
