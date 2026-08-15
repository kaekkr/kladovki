package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port         string
	DBPath       string
	JWTSecret    string
	JWTAccessTTL time.Duration
	CookieSecure bool
	CookieName   string
	Env          string
	APIPrefix    string
}

func Load() Config {
	env := os.Getenv("APP_ENV")
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

	secure := os.Getenv("COOKIE_SECURE") == "true" || os.Getenv("COOKIE_SECURE") == "1" || env == "production"

	// API Prefix
	apiPrefix := os.Getenv("API_PREFIX")
	if apiPrefix == "" {
		apiPrefix = "/api/v1"
	}

	return Config{
		Port:         port,
		DBPath:       dbPath,
		JWTSecret:    secret,
		JWTAccessTTL: time.Duration(ttlHours) * time.Hour,
		CookieSecure: secure,
		CookieName:   cookieName,
		Env:          env,
		APIPrefix:    apiPrefix,
	}
}
