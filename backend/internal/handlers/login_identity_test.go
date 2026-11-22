package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kaekkr/kladovki/internal/service"
)

func registerBody(phone string) map[string]any {
	return map[string]any{
		"accepted_terms_version": termsV, "iin": "010101500009", "full_name": "Асель",
		"phone": phone, "password": "secret1", "jk_id": "", "premises_number": "45",
	}
}

// loginID posts a login and returns the status and the logged-in user id.
func (e *testEnv) loginID(body map[string]any) (int, string) {
	e.t.Helper()
	r := e.do(http.MethodPost, "/auth/login", "", body)
	var out struct {
		ID string `json:"id"`
	}
	if r.Code == http.StatusOK {
		r.json(e.t, &out)
	}
	return r.Code, out.ID
}

func TestRegisterRejectsPhoneThatIsAnIINOrEmail(t *testing.T) {
	e := newTestEnv(t)
	for _, phone := range []string{"900101300001", e.adminA + "@x.kz", "abc", "+7701999887", "+1 202 555 0100", "+770199988770"} {
		body := registerBody(phone)
		body["jk_id"] = e.jkA
		if r := e.do(http.MethodPost, "/auth/register", "", body); r.Code != http.StatusBadRequest {
			t.Errorf("phone %q: want 400, got %d %s", phone, r.Code, r.Body)
		}
	}
}

func TestRegisterNormalizesKazakhPhone(t *testing.T) {
	e := newTestEnv(t)
	body := registerBody("8 (701) 999-88-77")
	body["jk_id"] = e.jkA
	if r := e.do(http.MethodPost, "/auth/register", "", body); r.Code != http.StatusOK {
		t.Fatalf("register: %d %s", r.Code, r.Body)
	}
	var phone string
	_ = e.db.QueryRow(`SELECT phone FROM users WHERE iin='010101500009'`).Scan(&phone)
	if phone != "+77019998877" {
		t.Fatalf("stored phone = %q, want +77019998877", phone)
	}
	for _, login := range []string{"+77019998877", "87019998877", "7 701 999 88 77", "+7 (701) 999-88-77"} {
		if code, _ := e.loginID(map[string]any{"login": login, "password": "secret1"}); code != http.StatusOK {
			t.Errorf("login %q: %d", login, code)
		}
	}
	// Same number in another spelling is a duplicate.
	dup := registerBody("+7 701 999 88 77")
	dup["iin"], dup["jk_id"] = "010101500010", e.jkA
	if r := e.do(http.MethodPost, "/auth/register", "", dup); r.Code != http.StatusConflict {
		t.Fatalf("duplicate phone: want 409, got %d %s", r.Code, r.Body)
	}
}

func TestLoginIsTypedSoAPlantedPhoneCannotHijackIINOrEmail(t *testing.T) {
	e := newTestEnv(t)
	// A legacy attacker row whose phone is the victim's IIN / email.
	hash, _ := service.HashPassword("attacker1")
	for _, phone := range []string{"900101300001", e.adminA + "@x.kz"} {
		e.exec(`INSERT INTO users (id,role,full_name,phone,password_hash,created_at) VALUES ($1,'resident','Attacker',$2,$3,$4)`,
			uuid.NewString(), phone, hash, time.Now().UTC())
	}
	// Rewrite the victims' tuples so an unordered scan meets the attacker rows first.
	e.exec(`UPDATE users SET full_name = full_name WHERE id IN ($1,$2)`, e.resident, e.adminA)
	if code, id := e.loginID(map[string]any{"login": "900101300001", "password": "password123"}); code != http.StatusOK || id != e.resident {
		t.Fatalf("IIN login: %d id=%s, want the victim %s", code, id, e.resident)
	}
	if code, id := e.loginID(map[string]any{"login": e.adminA + "@x.kz", "password": "password123"}); code != http.StatusOK || id != e.adminA {
		t.Fatalf("email login: %d id=%s, want the victim %s", code, id, e.adminA)
	}
	// The planted rows are not reachable through those identifiers.
	if code, _ := e.loginID(map[string]any{"login": "900101300001", "password": "attacker1"}); code != http.StatusUnauthorized {
		t.Fatalf("attacker via IIN: want 401, got %d", code)
	}
	// Legacy fields still work.
	if code, id := e.loginID(map[string]any{"phone": "+77000000003", "password": "password123"}); code != http.StatusOK || id != e.resident {
		t.Fatalf("legacy phone field: %d %s", code, id)
	}
	if code, id := e.loginID(map[string]any{"email": e.adminA + "@X.KZ", "password": "password123"}); code != http.StatusOK || id != e.adminA {
		t.Fatalf("legacy email field: %d %s", code, id)
	}
}

func TestLoginMatchesLegacyNonCanonicalPhone(t *testing.T) {
	e := newTestEnv(t)
	hash, _ := service.HashPassword("legacy11")
	id := uuid.NewString()
	e.exec(`INSERT INTO users (id,role,full_name,phone,password_hash,created_at) VALUES ($1,'resident','Old',$2,$3,$4)`,
		id, "8-700-555-44-33", hash, time.Now().UTC())
	if code, got := e.loginID(map[string]any{"login": "8-700-555-44-33", "password": "legacy11"}); code != http.StatusOK || got != id {
		t.Fatalf("legacy raw phone: %d %s", code, got)
	}
}

func TestSuperadminPlaceholderPhoneIsNotALogin(t *testing.T) {
	e := newTestEnv(t)
	hash, _ := service.HashPassword("password123")
	id := uuid.NewString()
	for _, phone := range []string{"superadmin:boss@x.kz", "superadmin:" + uuid.NewString()} {
		e.exec(`INSERT INTO users (id,role,full_name,phone,email,password_hash,created_at) VALUES ($1,'superadmin','Op',$2,$3,$4,$5)`,
			uuid.NewString(), phone, uuid.NewString()+"@y.kz", hash, time.Now().UTC())
		if code, _ := e.loginID(map[string]any{"login": phone, "password": "password123"}); code != http.StatusUnauthorized {
			t.Errorf("placeholder %q: want 401, got %d", phone, code)
		}
	}
	e.exec(`INSERT INTO users (id,role,full_name,phone,email,password_hash,created_at) VALUES ($1,'superadmin','Op','superadmin:boss2@x.kz','boss2@x.kz',$2,$3)`,
		id, hash, time.Now().UTC())
	if code, got := e.loginID(map[string]any{"login": "boss2@x.kz", "password": "password123"}); code != http.StatusOK || got != id {
		t.Fatalf("superadmin email login: %d %s", code, got)
	}
}

func TestPlatformAdminPhoneIsValidatedAndNormalized(t *testing.T) {
	e := newTestEnv(t)
	super := e.mkSuperadmin()
	if r := e.do(http.MethodPost, "/platform/jks", super, newJKBody("", "p1@example.kz", "900101300001")); r.Code != http.StatusBadRequest {
		t.Fatalf("IIN as admin phone: want 400, got %d %s", r.Code, r.Body)
	}
	out := e.createJK(super, newJKBody("", "p2@example.kz", "8 701 123 45 67"))
	if out.Admin.Phone != "+77011234567" {
		t.Fatalf("admin phone = %q, want +77011234567", out.Admin.Phone)
	}
}
