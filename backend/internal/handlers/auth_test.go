package handlers

import (
	"net/http"
	"testing"
)

func TestLoginWrongPasswordIs401(t *testing.T) {
	e := newTestEnv(t)
	r := e.do(http.MethodPost, "/auth/login", "", map[string]any{"email": e.resident + "@x.kz", "password": "nope"})
	if r.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d %s", r.Code, r.Body)
	}
}

// Dad's report (Oct 7): when the DB was down the app said "неверный пароль".
func TestLoginWhenDatabaseIsDownIs503(t *testing.T) {
	e := newTestEnv(t)
	e.db.Close()
	r := e.do(http.MethodPost, "/auth/login", "", map[string]any{"email": "x@x.kz", "password": "password123"})
	if r.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d %s", r.Code, r.Body)
	}
}
