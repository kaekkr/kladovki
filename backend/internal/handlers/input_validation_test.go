package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterRejectsBadTextWith400(t *testing.T) {
	e := newTestEnv(t)
	cases := map[string]func(map[string]any){
		"name too long":     func(m map[string]any) { m["full_name"] = strings.Repeat("Я", 256) },
		"NUL in name":       func(m map[string]any) { m["full_name"] = "Асель\x00" },
		"NUL in premises":   func(m map[string]any) { m["premises_number"] = "4\x005" },
		"NUL in iin":        func(m map[string]any) { m["iin"] = "01010150000\x00" },
		"password too long": func(m map[string]any) { m["password"] = strings.Repeat("p", 73) },
		"password short":    func(m map[string]any) { m["password"] = "12345" },
	}
	for name, mutate := range cases {
		body := registerBody("+77019998877")
		body["jk_id"] = e.jkA
		mutate(body)
		if r := e.do(http.MethodPost, "/auth/register", "", body); r.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d %s", name, r.Code, r.Body)
		}
	}
	// 255 runes and a 72-byte password are fine.
	ok := registerBody("+77019998877")
	ok["jk_id"], ok["full_name"], ok["password"] = e.jkA, strings.Repeat("Я", 255), strings.Repeat("p", 72)
	if r := e.do(http.MethodPost, "/auth/register", "", ok); r.Code != http.StatusOK {
		t.Fatalf("limits are inclusive: %d %s", r.Code, r.Body)
	}
}

func TestChangePasswordRejectsOverlongPassword(t *testing.T) {
	e := newTestEnv(t)
	r := e.do(http.MethodPut, "/me/password", e.resident, map[string]any{"current_password": "password123", "new_password": strings.Repeat("p", 73)})
	if r.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d %s", r.Code, r.Body)
	}
}

func TestLeadValidatesLengthsAndText(t *testing.T) {
	e := newTestEnv(t)
	for name, body := range map[string]map[string]any{
		"long name":  {"full_name": strings.Repeat("a", 256), "phone": "+77001112233"},
		"long phone": {"full_name": "Иван", "phone": strings.Repeat("7", 51)},
		"long email": {"full_name": "Иван", "phone": "+77001112233", "email": strings.Repeat("a", 251) + "@x.kz"},
		"NUL":        {"full_name": "Ив\x00ан", "phone": "+77001112233"},
	} {
		if r := e.do(http.MethodPost, "/lead", "", body); r.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d %s", name, r.Code, r.Body)
		}
	}
}

func TestBodyLargerThanLimitIs413(t *testing.T) {
	e := newTestEnv(t)
	big := bytes.Repeat([]byte("a"), 1<<20+1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/lead", bytes.NewReader(big))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("want 413, got %d", w.Code)
	}
	// Without Content-Length the reader still stops at the limit.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/lead", bytes.NewReader(big))
	req.ContentLength = -1
	w = httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest && w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("chunked: want 400/413, got %d", w.Code)
	}
}

func TestBadPathOrQueryParamIsNot500(t *testing.T) {
	e := newTestEnv(t)
	for _, path := range []string{"/storages/%00", "/rentals/ab%00cd", "/storages/%FF%FE", "/jks/%00/tariff"} {
		if r := e.do(http.MethodGet, path, e.resident, nil); r.Code != http.StatusNotFound {
			t.Errorf("GET %s: want 404, got %d %s", path, r.Code, r.Body)
		}
	}
	if r := e.do(http.MethodGet, "/residents/jk/"+e.jkA+"?status=%00", e.adminA, nil); r.Code != http.StatusBadRequest {
		t.Errorf("NUL in query: want 400, got %d %s", r.Code, r.Body)
	}
}
