package config

import (
	"log"
	"os"
	"strconv"
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

func Load() Config {
	if loaded := loadEnvFiles(envFiles...); len(loaded) > 0 {
		log.Printf("Info: env loaded from %v", loaded)
	} else {
		log.Println("Info: no .env files found, relying on system environment variables")
	}

	// Supports both APP_ENV and standard ENV variables
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("ENV")
	}
	if env == "" {
		env = "development"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "postgres://kladovki:kladovki_secret@localhost:5432/kladovki?sslmode=disable"
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		if env == "production" {
			log.Fatal("CRITICAL: JWT_SECRET environment variable is required in production!")
		}
		secret = "dev-secret-change-me-in-production-32chars"
		log.Println("WARNING: Using default JWT secret. Do not use in production!")
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
	}
}
