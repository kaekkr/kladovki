package handlers

import (
	"net/http"
	"strings"
	"testing"
)

// Tester (2026-11-22): no QR — the resident's ЖК is found from the premises
// cadastral number 21:319:021:848:8:ВП-2 (ЖК = 21:319:021:848:8). The ОСИ still approves.

const jkBCadastral = "21:319:021:848:8"

func (e *testEnv) cadastralBody(cadastral string) map[string]any {
	return map[string]any{
		"accepted_terms_version": termsV, "iin": "010101500009", "full_name": "Асель",
		"phone": "+77019998877", "password": "secret1", "premises_cadastral": cadastral,
	}
}

func TestRegisterFindsJKByCadastralNumber(t *testing.T) {
	e := newTestEnv(t)
	e.exec(`UPDATE jks SET cadastral_number = $2 WHERE id = $1`, e.jkB, jkBCadastral)

	r := e.do(http.MethodPost, "/auth/register", "", e.cadastralBody(" 21:319:021:848:8:вп-2 "))
	if r.Code != http.StatusOK {
		t.Fatalf("register: %d %s", r.Code, r.Body)
	}
	var jk, number, kind, cad, status string
	_ = e.db.QueryRow(`SELECT jk_id, premises_number, premises_kind, premises_cadastral, status
		FROM users WHERE iin='010101500009'`).Scan(&jk, &number, &kind, &cad, &status)
	if jk != e.jkB || number != "ВП-2" || kind != "commercial" || cad != "21:319:021:848:8:ВП-2" {
		t.Fatalf("got jk=%s number=%s kind=%s cadastral=%s", jk, number, kind, cad)
	}
	if status != "pending" {
		t.Fatalf("ownership is still confirmed by the ОСИ: status=%s", status)
	}
}

func TestRegisterCadastralErrors(t *testing.T) {
	e := newTestEnv(t)
	e.exec(`UPDATE jks SET cadastral_number = $2 WHERE id = $1`, e.jkB, jkBCadastral)

	cases := []struct {
		name string
		mod  func(m map[string]any)
	}{
		{"malformed", func(m map[string]any) { m["premises_cadastral"] = "21-319-ВП" }},
		{"no unit facet", func(m map[string]any) { m["premises_cadastral"] = jkBCadastral }},
		{"ЖК not connected", func(m map[string]any) { m["premises_cadastral"] = "11:111:111:111:1:5" }},
		{"QR of another ЖК", func(m map[string]any) {
			m["premises_cadastral"] = jkBCadastral + ":5"
			m["jk_id"] = e.jkA
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := e.cadastralBody("")
			c.mod(m)
			r := e.do(http.MethodPost, "/auth/register", "", m)
			if r.Code != http.StatusBadRequest || !strings.Contains(string(r.Body), "адастров") && !strings.Contains(string(r.Body), "ЖК") {
				t.Fatalf("want 400 about the cadastral number, got %d %s", r.Code, r.Body)
			}
		})
	}
}

func TestResubmitWithCadastralOfAnotherJKIsRefused(t *testing.T) {
	e := newTestEnv(t)
	e.exec(`UPDATE jks SET cadastral_number = $2 WHERE id = $1`, e.jkB, jkBCadastral)
	if r := e.do(http.MethodPost, "/auth/register", "", e.cadastralBody(jkBCadastral+":5")); r.Code != http.StatusOK {
		t.Fatalf("register: %d %s", r.Code, r.Body)
	}
	var userID string
	_ = e.db.QueryRow(`SELECT id FROM users WHERE iin='010101500009'`).Scan(&userID)

	e.exec(`UPDATE jks SET cadastral_number = '22:222:222:222:2' WHERE id = $1`, e.jkA)
	r := e.do(http.MethodPut, "/me/verification", userID, map[string]any{"premises_cadastral": "22:222:222:222:2:7"})
	if r.Code != http.StatusBadRequest {
		t.Fatalf("another ЖК's premises: want 400, got %d %s", r.Code, r.Body)
	}
	r = e.do(http.MethodPut, "/me/verification", userID, map[string]any{"premises_cadastral": jkBCadastral + ":17"})
	if r.Code != http.StatusOK {
		t.Fatalf("own ЖК: %d %s", r.Code, r.Body)
	}
	var number string
	_ = e.db.QueryRow(`SELECT premises_number FROM users WHERE id=$1`, userID).Scan(&number)
	if number != "17" {
		t.Fatalf("premises_number = %s, want 17", number)
	}
}

func TestJKAndMeExposeCadastralNumbers(t *testing.T) {
	e := newTestEnv(t)
	e.exec(`UPDATE jks SET cadastral_number = $2, address = 'Астана, ул. 1' WHERE id = $1`, e.jkB, jkBCadastral)
	if r := e.do(http.MethodPost, "/auth/register", "", e.cadastralBody(jkBCadastral+":5")); r.Code != http.StatusOK {
		t.Fatalf("register: %d %s", r.Code, r.Body)
	}
	var userID string
	_ = e.db.QueryRow(`SELECT id FROM users WHERE iin='010101500009'`).Scan(&userID)

	var jk struct {
		Address         string `json:"address"`
		CadastralNumber string `json:"cadastral_number"`
	}
	r := e.do(http.MethodGet, "/jks/"+e.jkB, userID, nil)
	r.json(t, &jk)
	if jk.CadastralNumber != jkBCadastral || jk.Address != "Астана, ул. 1" {
		t.Fatalf("GET /jks/:id = %+v", jk)
	}

	var me struct {
		PremisesCadastral string `json:"premises_cadastral"`
	}
	e.do(http.MethodGet, "/auth/me", userID, nil).json(t, &me)
	if me.PremisesCadastral != jkBCadastral+":5" {
		t.Fatalf("/auth/me premises_cadastral = %q", me.PremisesCadastral)
	}
}
