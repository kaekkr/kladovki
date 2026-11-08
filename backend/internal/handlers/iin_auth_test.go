package handlers

import (
	"net/http"
	"testing"
)

func (e *testEnv) iinStatus(iin string) (int, bool) {
	r := e.do(http.MethodPost, "/auth/iin-status", "", map[string]any{"iin": iin})
	var out struct {
		Registered bool `json:"registered"`
	}
	if r.Code == http.StatusOK {
		r.json(e.t, &out)
	}
	return r.Code, out.Registered
}

func TestLoginWithIIN(t *testing.T) {
	e := newTestEnv(t)
	r := e.do(http.MethodPost, "/auth/login", "", map[string]any{"login": "900101300001", "password": "password123"})
	if r.Code != http.StatusOK {
		t.Fatalf("IIN login: %d %s", r.Code, r.Body)
	}
	// Old clients sending "email"/"phone" keep working.
	if r := e.do(http.MethodPost, "/auth/login", "", map[string]any{"email": e.adminA + "@x.kz", "password": "password123"}); r.Code != http.StatusOK {
		t.Fatalf("email login: %d %s", r.Code, r.Body)
	}
}

func TestIINStatusTellsFirstLoginApart(t *testing.T) {
	e := newTestEnv(t)
	if code, reg := e.iinStatus("900101300001"); code != http.StatusOK || !reg {
		t.Fatalf("known IIN: %d %v", code, reg)
	}
	if code, reg := e.iinStatus("010101500009"); code != http.StatusOK || reg {
		t.Fatalf("new IIN: %d %v", code, reg)
	}
	if code, _ := e.iinStatus("123"); code != http.StatusBadRequest {
		t.Fatalf("bad IIN: want 400, got %d", code)
	}
}

func TestFirstLoginRegistersIntoQRJK(t *testing.T) {
	e := newTestEnv(t)
	r := e.do(http.MethodPost, "/auth/register", "", map[string]any{
		"iin": "010101500009", "full_name": "Асель", "phone": "+77019998877", "password": "secret1", "jk_id": e.jkB,
	})
	if r.Code != http.StatusOK {
		t.Fatalf("register: %d %s", r.Code, r.Body)
	}
	var jk string
	_ = e.db.QueryRow(`SELECT jk_id FROM users WHERE iin='010101500009'`).Scan(&jk)
	if jk != e.jkB {
		t.Fatalf("jk = %s, want %s", jk, e.jkB)
	}
	if r := e.do(http.MethodPost, "/auth/login", "", map[string]any{"login": "010101500009", "password": "secret1"}); r.Code != http.StatusOK {
		t.Fatalf("login after register: %d", r.Code)
	}
}

func TestRegisterWithoutQRUsesTheOnlyJK(t *testing.T) {
	e := newTestEnv(t)
	body := map[string]any{"iin": "010101500009", "full_name": "Асель", "phone": "+77019998877", "password": "secret1"}

	// Two JKs exist → must come through the OSI's QR link.
	if r := e.do(http.MethodPost, "/auth/register", "", body); r.Code != http.StatusBadRequest {
		t.Fatalf("two JKs, no jk_id: want 400, got %d %s", r.Code, r.Body)
	}

	// Single JK in the system (dad: "при возврате одного ЖК выбора не должно быть").
	e.exec(`UPDATE users SET jk_id = NULL WHERE jk_id = $1`, e.jkB)
	e.exec(`DELETE FROM jks WHERE id = $1`, e.jkB)
	if r := e.do(http.MethodPost, "/auth/register", "", body); r.Code != http.StatusOK {
		t.Fatalf("single JK: %d %s", r.Code, r.Body)
	}
	var jk string
	_ = e.db.QueryRow(`SELECT jk_id FROM users WHERE iin='010101500009'`).Scan(&jk)
	if jk != e.jkA {
		t.Fatalf("jk = %s, want the only JK %s", jk, e.jkA)
	}
}

func TestRegisterValidation(t *testing.T) {
	e := newTestEnv(t)
	base := func() map[string]any {
		return map[string]any{"iin": "010101500009", "full_name": "Асель", "phone": "+77019998877", "password": "secret1", "jk_id": e.jkA}
	}
	cases := []struct {
		name string
		mod  func(m map[string]any)
		want int
	}{
		{"IIN taken", func(m map[string]any) { m["iin"] = "900101300001" }, http.StatusConflict},
		{"phone taken", func(m map[string]any) { m["phone"] = "+77000000003" }, http.StatusConflict},
		{"short password", func(m map[string]any) { m["password"] = "123" }, http.StatusBadRequest},
		{"unknown JK", func(m map[string]any) { m["jk_id"] = "nope" }, http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := base()
			c.mod(m)
			if r := e.do(http.MethodPost, "/auth/register", "", m); r.Code != c.want {
				t.Fatalf("want %d, got %d %s", c.want, r.Code, r.Body)
			}
		})
	}
}
