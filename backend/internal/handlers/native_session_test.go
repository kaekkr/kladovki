package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The Android app (Capacitor) runs at https://localhost and calls the API on
// another site, so the session cookie is never sent back: login "works" but
// every data request is 401. The app therefore asks for the token and sends
// it as "Authorization: Bearer".
func (e *testEnv) loginAs(login string, native bool) (int, map[string]any) {
	e.t.Helper()
	b, _ := json.Marshal(map[string]any{"login": login, "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if native {
		req.Header.Set("X-Kladovki-Client", "native")
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestNativeAppGetsTokenAndCanLoadData(t *testing.T) {
	e := newTestEnv(t)

	// The website keeps using the HttpOnly cookie only — no token in the body.
	if _, body := e.loginAs("900101300001", false); body["access_token"] != nil {
		t.Fatal("web login must not expose the token to JavaScript")
	}

	code, body := e.loginAs("900101300001", true)
	tok, _ := body["access_token"].(string)
	if code != http.StatusOK || tok == "" {
		t.Fatalf("native login: %d %v", code, body)
	}
	// With only the bearer token (no cookie), the app can load its data.
	for _, path := range []string{"/auth/me", "/storages/jk/" + e.jkA, "/rentals/my"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1"+path, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		w := httptest.NewRecorder()
		e.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s with bearer: %d %s", path, w.Code, w.Body)
		}
	}
}

func TestNativeRegistrationGetsToken(t *testing.T) {
	e := newTestEnv(t)
	b, _ := json.Marshal(map[string]any{
		"iin": "010101500009", "full_name": "Асель", "phone": "+77019998877", "password": "secret1",
		"jk_id": e.jkA, "premises_number": "45", "accepted_terms_version": termsV,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Kladovki-Client", "native")
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if w.Code != http.StatusOK || out["access_token"] == nil {
		t.Fatalf("native register: %d %v", w.Code, out)
	}
}
