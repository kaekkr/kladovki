package handlers

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kaekkr/kladovki/internal/database"
	"github.com/kaekkr/kladovki/internal/service"
)

// Code review: users.phone rows from before normalization may be stored as
// 87001234567 or "8 700 …". They are the same number as +77001234567.

func (e *testEnv) legacyPhoneUser(phone, password string) string {
	e.t.Helper()
	hash, _ := service.HashPassword(password)
	id := uuid.NewString()
	e.exec(`INSERT INTO users (id,role,full_name,phone,password_hash,jk_id,created_at) VALUES ($1,'resident','Old',$2,$3,$4,$5)`,
		id, phone, hash, e.jkA, time.Now().UTC())
	return id
}

func TestLegacyPhoneSpellingsCountAsTaken(t *testing.T) {
	e := newTestEnv(t)
	e.legacyPhoneUser("8 (701) 555-44-33", "legacy11")

	body := registerBody("+77015554433")
	body["jk_id"] = e.jkA
	if r := e.do(http.MethodPost, "/auth/register", "", body); r.Code != http.StatusConflict {
		t.Fatalf("register with a legacy-stored number: want 409, got %d %s", r.Code, r.Body)
	}
	r := e.do(http.MethodPut, "/me/phone", e.other, map[string]any{"phone": "+7 701 555 44 33", "current_password": "password123"})
	if r.Code != http.StatusConflict {
		t.Fatalf("update to a legacy-stored number: want 409, got %d %s", r.Code, r.Body)
	}
	super := e.mkSuperadmin()
	if r := e.do(http.MethodPost, "/platform/jks", super, newJKBody("", "new@example.kz", "87015554433")); r.Code != http.StatusConflict {
		t.Fatalf("platform admin with a legacy-stored number: want 409, got %d %s", r.Code, r.Body)
	}
}

func TestLoginMatchesLegacyPhoneByDigits(t *testing.T) {
	e := newTestEnv(t)
	id := e.legacyPhoneUser("87015554433", "legacy11")
	for _, login := range []string{"+77015554433", "8 701 555 44 33"} {
		if code, got := e.loginID(map[string]any{"login": login, "password": "legacy11"}); code != http.StatusOK || got != id {
			t.Fatalf("login %q: %d %s, want %s", login, code, got, id)
		}
	}
}

// Two rows with the same number: never pick one arbitrarily.
func TestLoginWithAmbiguousLegacyPhoneIsRefused(t *testing.T) {
	e := newTestEnv(t)
	e.legacyPhoneUser("87015554433", "legacy11")
	e.legacyPhoneUser("+77015554433", "legacy11")
	if code, _ := e.loginID(map[string]any{"login": "+77015554433", "password": "legacy11"}); code != http.StatusUnauthorized {
		t.Fatalf("ambiguous phone login: want 401, got %d", code)
	}
}

func TestMigrationNormalizesLegacyPhonesWithoutCollisions(t *testing.T) {
	e := newTestEnv(t)
	alone := e.legacyPhoneUser("8 (701) 111-22-33", "x123456")
	dupA := e.legacyPhoneUser("87012223344", "x123456")
	dupB := e.legacyPhoneUser("+7 701 222 33 44", "x123456")
	odd := e.legacyPhoneUser("12345", "x123456")
	super := e.mkSuperadmin()
	var superPhone string
	_ = e.db.QueryRow(`SELECT phone FROM users WHERE id=$1`, super).Scan(&superPhone)

	for run := 0; run < 2; run++ { // a second boot changes nothing more
		db, err := database.Open(os.Getenv("TEST_DATABASE_URL"))
		if err != nil {
			t.Fatalf("re-run migrations: %v", err)
		}
		db.Close()
	}
	want := map[string]string{
		alone: "+77011112233",
		dupA:  "87012223344",      // collides with dupB: left for a person to sort out
		dupB:  "+7 701 222 33 44", // idem
		odd:   "12345",
		super: superPhone,
	}
	for id, phone := range want {
		var got string
		_ = e.db.QueryRow(`SELECT phone FROM users WHERE id=$1`, id).Scan(&got)
		if got != phone {
			t.Errorf("user %s phone = %q, want %q", id[:8], got, phone)
		}
	}
}
