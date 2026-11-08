package handlers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/kaekkr/kladovki/internal/legal"
)

// termsV is the current resident agreement version clients must send when they accept it.
var termsV = legal.Version(legal.ResidentTerms)

type acceptance struct {
	Document, Version, Context, RentalID string
}

func (e *testEnv) acceptances(userID string) []acceptance {
	e.t.Helper()
	rows, err := e.db.Query(`SELECT document, version, context, COALESCE(rental_id,'') FROM agreement_acceptances WHERE user_id=$1 ORDER BY accepted_at`, userID)
	if err != nil {
		e.t.Fatal(err)
	}
	defer rows.Close()
	var out []acceptance
	for rows.Next() {
		var a acceptance
		_ = rows.Scan(&a.Document, &a.Version, &a.Context, &a.RentalID)
		out = append(out, a)
	}
	return out
}

func TestAgreementTextsArePublic(t *testing.T) {
	e := newTestEnv(t)
	var d struct{ Doc, Version, Lang, Title, Text string }
	r := e.do(http.MethodGet, "/legal/resident_terms?lang=kk", "", nil)
	if r.Code != http.StatusOK {
		t.Fatalf("get: %d", r.Code)
	}
	r.json(t, &d)
	if d.Version != termsV || d.Lang != "kk" || !strings.Contains(d.Title, "ПАЙДАЛАНУШЫ") {
		t.Fatalf("doc = %+v", d)
	}
	if r := e.do(http.MethodGet, "/legal/osi_rules", "", nil); r.Code != http.StatusOK {
		t.Fatalf("osi rules: %d", r.Code)
	}
	if r := e.do(http.MethodGet, "/legal/nope", "", nil); r.Code != http.StatusNotFound {
		t.Fatalf("unknown doc: %d", r.Code)
	}
}

func TestRegistrationRequiresCurrentAgreement(t *testing.T) {
	e := newTestEnv(t)
	body := map[string]any{
		"iin": "010101500009", "full_name": "Асель", "phone": "+77019998877",
		"password": "secret1", "jk_id": e.jkA, "premises_number": "45",
	}
	if r := e.do(http.MethodPost, "/auth/register", "", body); r.Code != http.StatusBadRequest {
		t.Fatalf("no consent: %d, want 400", r.Code)
	}
	body["accepted_terms_version"] = "2020-01-01"
	if r := e.do(http.MethodPost, "/auth/register", "", body); r.Code != http.StatusBadRequest {
		t.Fatalf("old version: %d, want 400", r.Code)
	}
	id := e.register(nil) // sends the current version
	if a := e.acceptances(id); len(a) != 1 || a[0].Document != legal.ResidentTerms || a[0].Version != termsV || a[0].Context != "registration" {
		t.Fatalf("acceptances = %+v", a)
	}
}

func TestPaymentAndRenewalRequireConsent(t *testing.T) {
	e := newTestEnv(t)
	id := e.lockAs(e.resident, e.storageA, 1)
	if r := e.do(http.MethodPost, "/payments", e.resident, map[string]any{"rental_id": id, "phone": "+77001112233"}); r.Code != http.StatusBadRequest {
		t.Fatalf("payment without consent: %d, want 400", r.Code)
	}
	if r := e.do(http.MethodPost, "/payments", e.resident, map[string]any{"rental_id": id, "phone": "+77001112233", "accepted_terms_version": termsV}); r.Code != http.StatusOK {
		t.Fatalf("payment: %d %s", r.Code, r.Body)
	}
	e.webhook(paidEvent(e.paymentIDFor(id)), "valid")

	if r := e.do(http.MethodPost, "/rentals/"+id+"/renew", e.resident, map[string]any{"months": 1, "phone": "+77001112233"}); r.Code != http.StatusBadRequest {
		t.Fatalf("renew without consent: %d, want 400", r.Code)
	}
	if code, _, _ := e.renew(e.resident, id, 1); code != http.StatusCreated {
		t.Fatalf("renew: %d", code)
	}
	a := e.acceptances(e.resident)
	if len(a) != 2 || a[0].Context != "payment" || a[1].Context != "renewal" || a[0].RentalID != id || a[1].RentalID != id {
		t.Fatalf("acceptances = %+v", a)
	}
}

func TestOSIAcceptsRegulation(t *testing.T) {
	e := newTestEnv(t)
	type st struct {
		Version  string `json:"version"`
		Accepted bool   `json:"accepted"`
	}
	status := func(user string) map[string]st {
		var s map[string]st
		e.do(http.MethodGet, "/legal/status", user, nil).json(t, &s)
		return s
	}
	if s := status(e.adminA); s[legal.OSIRules].Accepted || s[legal.OSIRules].Version == "" {
		t.Fatalf("status before = %+v", s)
	}
	if r := e.do(http.MethodPost, "/legal/accept", e.resident, map[string]any{"doc": legal.OSIRules, "version": legal.Version(legal.OSIRules)}); r.Code != http.StatusForbidden {
		t.Fatalf("resident accepting ОСИ rules: %d", r.Code)
	}
	if r := e.do(http.MethodPost, "/legal/accept", e.adminA, map[string]any{"doc": legal.OSIRules, "version": "old"}); r.Code != http.StatusBadRequest {
		t.Fatalf("old version: %d", r.Code)
	}
	if r := e.do(http.MethodPost, "/legal/accept", e.adminA, map[string]any{"doc": legal.OSIRules, "version": legal.Version(legal.OSIRules)}); r.Code != http.StatusOK {
		t.Fatalf("accept: %d %s", r.Code, r.Body)
	}
	if s := status(e.adminA); !s[legal.OSIRules].Accepted {
		t.Fatalf("status after = %+v", s)
	}
}
