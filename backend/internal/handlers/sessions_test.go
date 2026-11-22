package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// doTok calls the API with a fixed bearer token (as the Android app does);
// native=true also sends X-Kladovki-Client: native.
func (e *testEnv) doTok(method, path, token string, body any, native bool) resp {
	e.t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, "/api/v1"+path, rdr)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if native {
		req.Header.Set("X-Kladovki-Client", "native")
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{Code: w.Code, Body: w.Body.Bytes(), Header: w.Header()}
}

func (e *testEnv) nativeToken(login string) string {
	e.t.Helper()
	code, body := e.loginAs(login, true)
	tok, _ := body["access_token"].(string)
	if code != http.StatusOK || tok == "" {
		e.t.Fatalf("native login %s: %d %v", login, code, body)
	}
	return tok
}

func (e *testEnv) wantCode(r resp, want int, what string) {
	e.t.Helper()
	if r.Code != want {
		e.t.Fatalf("%s: want %d, got %d %s", what, want, r.Code, r.Body)
	}
}

func TestLogoutRevokesTheToken(t *testing.T) {
	e := newTestEnv(t)
	tok := e.nativeToken("900101300001")
	e.wantCode(e.doTok(http.MethodGet, "/auth/me", tok, nil, false), http.StatusOK, "before logout")
	e.wantCode(e.doTok(http.MethodPost, "/auth/logout", tok, nil, false), http.StatusOK, "logout")
	e.wantCode(e.doTok(http.MethodGet, "/auth/me", tok, nil, false), http.StatusUnauthorized, "after logout")
	e.wantCode(e.doTok(http.MethodGet, "/rentals/my", tok, nil, false), http.StatusUnauthorized, "data after logout")
}

func TestPasswordChangeRevokesOldSessionsButKeepsTheCaller(t *testing.T) {
	e := newTestEnv(t)
	tok := e.nativeToken("900101300001")
	other := e.nativeToken("900101300001") // a second device
	r := e.doTok(http.MethodPut, "/me/password", tok, map[string]any{"current_password": "password123", "new_password": "newpass1"}, true)
	e.wantCode(r, http.StatusOK, "change")
	var out struct {
		AccessToken string `json:"access_token"`
	}
	r.json(t, &out)
	if out.AccessToken == "" {
		t.Fatalf("native caller must get a fresh token: %s", r.Body)
	}
	e.wantCode(e.doTok(http.MethodGet, "/auth/me", tok, nil, false), http.StatusUnauthorized, "old token")
	e.wantCode(e.doTok(http.MethodGet, "/auth/me", other, nil, false), http.StatusUnauthorized, "other device")
	e.wantCode(e.doTok(http.MethodGet, "/auth/me", out.AccessToken, nil, false), http.StatusOK, "fresh token")
	// The website gets a fresh cookie instead.
	web := e.doTok(http.MethodPut, "/me/password", out.AccessToken, map[string]any{"current_password": "newpass1", "new_password": "newpass2"}, false)
	e.wantCode(web, http.StatusOK, "web change")
	if !strings.Contains(strings.Join(web.Header.Values("Set-Cookie"), ";"), "access_token=ey") {
		t.Fatalf("web caller must get a fresh cookie: %v", web.Header)
	}
}

func TestPasswordResetsRevokeSessions(t *testing.T) {
	e := newTestEnv(t)
	resTok := e.nativeToken("900101300001")
	e.wantCode(e.do(http.MethodPost, "/residents/"+e.resident+"/reset-password", e.adminA, nil), http.StatusOK, "ОСИ reset")
	e.wantCode(e.doTok(http.MethodGet, "/auth/me", resTok, nil, false), http.StatusUnauthorized, "resident after reset")

	adminTok := e.nativeToken(e.adminA + "@x.kz")
	e.wantCode(e.do(http.MethodPost, "/platform/admins/"+e.adminA+"/reset-password", e.mkSuperadmin(), nil), http.StatusOK, "platform reset")
	e.wantCode(e.doTok(http.MethodGet, "/auth/me", adminTok, nil, false), http.StatusUnauthorized, "admin after reset")
}

func TestDeletedUserTokenIsRejected(t *testing.T) {
	e := newTestEnv(t)
	id := uuid.NewString()
	e.exec(`INSERT INTO users (id,role,full_name,phone,password_hash,created_at) VALUES ($1,'resident','Gone','+77005550000','x',$2)`, id, time.Now().UTC())
	tok := e.token(id)
	e.exec(`DELETE FROM users WHERE id=$1`, id)
	e.wantCode(e.doTok(http.MethodGet, "/auth/me", tok, nil, false), http.StatusUnauthorized, "deleted user")
}

func TestRoleAndJKComeFromTheDatabase(t *testing.T) {
	e := newTestEnv(t)
	// A token minted while the user was an ОСИ of ЖК A...
	tok, _, err := e.tokens.Generate(e.resident, "admin", e.jkA, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	// ...is not an admin token once the DB says resident.
	e.wantCode(e.doTok(http.MethodGet, "/residents/jk/"+e.jkA, tok, nil, false), http.StatusForbidden, "stale admin role")
	// And an ОСИ moved to ЖК B no longer manages ЖК A.
	adminTok := e.token(e.adminA)
	e.exec(`UPDATE users SET jk_id=$1 WHERE id=$2`, e.jkB, e.adminA)
	e.wantCode(e.doTok(http.MethodGet, "/residents/jk/"+e.jkA, adminTok, nil, false), http.StatusForbidden, "stale jk")
}

func TestMustChangePasswordIsEnforcedByTheServer(t *testing.T) {
	e := newTestEnv(t)
	e.exec(`UPDATE users SET must_change_password=true WHERE id=$1`, e.resident)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/rentals/my"},
		{http.MethodGet, "/notifications/unread-count"},
		{http.MethodPost, "/rentals/lock"},
		{http.MethodGet, "/me/channels"},
	} {
		r := e.do(c.method, c.path, e.resident, map[string]any{"storage_id": e.storageA, "months": 1})
		if r.Code != http.StatusForbidden || !strings.Contains(string(r.Body), `"code":"password_change_required"`) ||
			!strings.Contains(string(r.Body), "Сначала смените временный пароль") {
			t.Errorf("%s %s: %d %s", c.method, c.path, r.Code, r.Body)
		}
	}
	e.wantCode(e.do(http.MethodGet, "/auth/me", e.resident, nil), http.StatusOK, "me")
	e.wantCode(e.do(http.MethodGet, "/legal/status", e.resident, nil), http.StatusOK, "legal status")
	e.wantCode(e.do(http.MethodGet, "/legal/resident_terms", "", nil), http.StatusOK, "legal text")
	e.wantCode(e.do(http.MethodPut, "/me/password", e.resident, map[string]any{"current_password": "password123", "new_password": "newpass1"}), http.StatusOK, "change")
	e.wantCode(e.do(http.MethodGet, "/rentals/my", e.resident, nil), http.StatusOK, "after change")
}
