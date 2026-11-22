package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kaekkr/kladovki/internal/service"
)

// mkSuperadmin inserts a platform operator (no ЖК) with password "password123".
func (e *testEnv) mkSuperadmin() string {
	e.t.Helper()
	hash, _ := service.HashPassword("password123")
	id := uuid.NewString()
	e.exec(`INSERT INTO users (id,role,full_name,phone,email,password_hash,created_at)
		VALUES ($1,'superadmin','Operator',$2,$3,$4,$5)`,
		id, "superadmin:"+id[:8], id+"@x.kz", hash, time.Now().UTC())
	return id
}

// changeTempPassword replaces a temporary password (the server allows nothing else before).
func (e *testEnv) changeTempPassword(userID, temp string) {
	e.t.Helper()
	r := e.do(http.MethodPut, "/me/password", userID, map[string]any{"current_password": temp, "new_password": "own-password-1"})
	if r.Code != http.StatusOK {
		e.t.Fatalf("change temporary password: %d %s", r.Code, r.Body)
	}
}

type createdJK struct {
	JK struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		Address         string `json:"address"`
		CadastralNumber string `json:"cadastral_number"`
	} `json:"jk"`
	Admin struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"admin"`
	TemporaryPassword string `json:"temporary_password"`
}

func newJKBody(cadastral, email, phone string) map[string]any {
	return map[string]any{
		"name":             "  ЖК Новый  ",
		"address":          "Астана, ул. Новая 1",
		"cadastral_number": cadastral,
		"admin": map[string]any{
			"full_name": "Иванов Иван",
			"email":     email,
			"phone":     phone,
		},
	}
}

func (e *testEnv) createJK(super string, body map[string]any) createdJK {
	e.t.Helper()
	r := e.do(http.MethodPost, "/platform/jks", super, body)
	if r.Code != http.StatusCreated {
		e.t.Fatalf("create jk: %d %s", r.Code, r.Body)
	}
	var out createdJK
	r.json(e.t, &out)
	return out
}

func TestPlatformRoutesRequireSuperadmin(t *testing.T) {
	e := newTestEnv(t)
	routes := []struct{ method, path string }{
		{http.MethodGet, "/platform/jks"},
		{http.MethodPost, "/platform/jks"},
		{http.MethodGet, "/platform/jks/" + e.jkA},
		{http.MethodPut, "/platform/jks/" + e.jkA},
		{http.MethodGet, "/platform/jks/" + e.jkA + "/admins"},
		{http.MethodPost, "/platform/jks/" + e.jkA + "/admins"},
		{http.MethodPost, "/platform/admins/" + e.adminA + "/reset-password"},
	}
	callers := []struct {
		name, id string
		want     int
	}{
		{"anonymous", "", http.StatusUnauthorized},
		{"resident", e.resident, http.StatusForbidden},
		{"admin", e.adminA, http.StatusForbidden},
	}
	for _, rt := range routes {
		for _, c := range callers {
			body := map[string]any{"name": "x", "address": "y"}
			if r := e.do(rt.method, rt.path, c.id, body); r.Code != c.want {
				t.Errorf("%s %s as %s: got %d, want %d", rt.method, rt.path, c.name, r.Code, c.want)
			}
		}
	}
}

func TestPlatformCreateJKWithAdmin(t *testing.T) {
	e := newTestEnv(t)
	super := e.mkSuperadmin()

	out := e.createJK(super, newJKBody(" 21:319:021:848:8 ", "New.Admin@Example.kz", "+77011234567"))
	if out.JK.ID == "" || out.JK.Name != "ЖК Новый" || out.JK.CadastralNumber != "21:319:021:848:8" {
		t.Fatalf("unexpected jk: %+v", out.JK)
	}
	if len(out.TemporaryPassword) < 10 {
		t.Fatalf("temporary password too short: %q", out.TemporaryPassword)
	}

	var role, jk string
	var mustChange bool
	if err := e.db.QueryRow(`SELECT role, COALESCE(jk_id,''), must_change_password FROM users WHERE id=$1`, out.Admin.ID).
		Scan(&role, &jk, &mustChange); err != nil {
		t.Fatal(err)
	}
	if role != "admin" || jk != out.JK.ID || !mustChange {
		t.Fatalf("admin row: role=%s jk=%s must_change=%v", role, jk, mustChange)
	}
	var owner string
	_ = e.db.QueryRow(`SELECT owner_id FROM jks WHERE id=$1`, out.JK.ID).Scan(&owner)
	if owner != out.Admin.ID {
		t.Fatalf("jk owner = %s, want %s", owner, out.Admin.ID)
	}

	// The new ОСИ logs in with the temporary password (email is case-insensitive).
	r := e.do(http.MethodPost, "/auth/login", "", map[string]any{"login": "new.admin@example.kz", "password": out.TemporaryPassword})
	if r.Code != http.StatusOK {
		t.Fatalf("login: %d %s", r.Code, r.Body)
	}
	var me struct {
		Role       string `json:"role"`
		JKID       string `json:"jk_id"`
		MustChange bool   `json:"must_change_password"`
	}
	r.json(t, &me)
	if me.Role != "admin" || me.JKID != out.JK.ID || !me.MustChange {
		t.Fatalf("login profile: %+v", me)
	}
	// …must replace the temporary password first, then can use their own ЖК's ОСИ routes.
	if r := e.do(http.MethodGet, "/rentals/jk/"+out.JK.ID, out.Admin.ID, nil); r.Code != http.StatusForbidden {
		t.Fatalf("before the password change: want 403, got %d %s", r.Code, r.Body)
	}
	e.changeTempPassword(out.Admin.ID, out.TemporaryPassword)
	if r := e.do(http.MethodGet, "/rentals/jk/"+out.JK.ID, out.Admin.ID, nil); r.Code != http.StatusOK {
		t.Fatalf("new admin own ЖК: %d %s", r.Code, r.Body)
	}

	// The list shows stats.
	var list []struct {
		ID         string `json:"id"`
		Storages   int    `json:"storages_count"`
		Admins     int    `json:"admins_count"`
		Residents  int    `json:"residents_count"`
		Cadastral  string `json:"cadastral_number"`
		Address    string `json:"address"`
		Name       string `json:"name"`
		CreatedAt  string `json:"created_at"`
		NotPresent string `json:"password_hash"`
	}
	e.do(http.MethodGet, "/platform/jks", super, nil).json(t, &list)
	if len(list) != 3 {
		t.Fatalf("list: %d jks", len(list))
	}
	stats := map[string][3]int{}
	for _, j := range list {
		stats[j.ID] = [3]int{j.Storages, j.Admins, j.Residents}
	}
	if got := stats[e.jkA]; got != [3]int{2, 1, 2} {
		t.Fatalf("jkA stats = %v", got)
	}
	if got := stats[out.JK.ID]; got != [3]int{0, 1, 0} {
		t.Fatalf("new jk stats = %v", got)
	}

	// One ЖК by id.
	if r := e.do(http.MethodGet, "/platform/jks/"+out.JK.ID, super, nil); r.Code != http.StatusOK {
		t.Fatalf("get jk: %d", r.Code)
	}
	if r := e.do(http.MethodGet, "/platform/jks/"+uuid.NewString(), super, nil); r.Code != http.StatusNotFound {
		t.Fatalf("get unknown jk: %d", r.Code)
	}
}

func TestPlatformCreateJKValidation(t *testing.T) {
	e := newTestEnv(t)
	super := e.mkSuperadmin()
	e.createJK(super, newJKBody("21:319:021:848:8", "first@example.kz", "+77010000001"))

	var usersBefore, jksBefore int
	_ = e.db.QueryRow(`SELECT count(*) FROM users`).Scan(&usersBefore)
	_ = e.db.QueryRow(`SELECT count(*) FROM jks`).Scan(&jksBefore)

	existingEmail := e.adminA + "@X.KZ" // seeded lower-case; must match case-insensitively
	cases := []struct {
		name string
		body map[string]any
		want int
	}{
		{"duplicate cadastral", newJKBody("21 : 319:021:848:8", "a@example.kz", "+77010000002"), http.StatusConflict},
		{"invalid cadastral", newJKBody("21:319:abc", "b@example.kz", "+77010000003"), http.StatusBadRequest},
		{"duplicate email", newJKBody("", existingEmail, "+77010000004"), http.StatusConflict},
		{"duplicate phone", newJKBody("", "c@example.kz", "+77010000001"), http.StatusConflict},
		{"bad email", newJKBody("", "not-an-email", "+77010000005"), http.StatusBadRequest},
		{"no phone", newJKBody("", "d@example.kz", "   "), http.StatusBadRequest},
		{"no name", map[string]any{"name": " ", "address": "x", "admin": map[string]any{"full_name": "A", "email": "e@example.kz", "phone": "+77010000006"}}, http.StatusBadRequest},
		{"no address", map[string]any{"name": "N", "address": "", "admin": map[string]any{"full_name": "A", "email": "f@example.kz", "phone": "+77010000007"}}, http.StatusBadRequest},
		{"no admin name", map[string]any{"name": "N", "address": "x", "admin": map[string]any{"full_name": "", "email": "g@example.kz", "phone": "+77010000008"}}, http.StatusBadRequest},
	}
	for _, c := range cases {
		r := e.do(http.MethodPost, "/platform/jks", super, c.body)
		if r.Code != c.want {
			t.Errorf("%s: got %d %s, want %d", c.name, r.Code, r.Body, c.want)
		}
		var body struct {
			Error string `json:"error"`
		}
		r.json(t, &body)
		if body.Error == "" {
			t.Errorf("%s: no error message", c.name)
		}
	}

	var usersAfter, jksAfter int
	_ = e.db.QueryRow(`SELECT count(*) FROM users`).Scan(&usersAfter)
	_ = e.db.QueryRow(`SELECT count(*) FROM jks`).Scan(&jksAfter)
	if usersAfter != usersBefore || jksAfter != jksBefore {
		t.Fatalf("failed creates left rows: users %d→%d, jks %d→%d", usersBefore, usersAfter, jksBefore, jksAfter)
	}

	// Two ЖК without a cadastral number are fine.
	e.createJK(super, newJKBody("", "h@example.kz", "+77010000009"))
	e.createJK(super, newJKBody("", "i@example.kz", "+77010000010"))
}

func TestPlatformUpdateJK(t *testing.T) {
	e := newTestEnv(t)
	super := e.mkSuperadmin()
	out := e.createJK(super, newJKBody("21:319:021:848:8", "u@example.kz", "+77010000011"))

	path := "/platform/jks/" + e.jkA
	r := e.do(http.MethodPut, path, super, map[string]any{"name": " ЖК А ", "address": "Алматы", "cadastral_number": "21:319:021:848:9"})
	if r.Code != http.StatusOK {
		t.Fatalf("update: %d %s", r.Code, r.Body)
	}
	var name, address, cad string
	_ = e.db.QueryRow(`SELECT name, address, cadastral_number FROM jks WHERE id=$1`, e.jkA).Scan(&name, &address, &cad)
	if name != "ЖК А" || address != "Алматы" || cad != "21:319:021:848:9" {
		t.Fatalf("after update: %q %q %q", name, address, cad)
	}

	if r := e.do(http.MethodPut, path, super, map[string]any{"name": "ЖК А", "address": "Алматы", "cadastral_number": "21:319:021:848:8"}); r.Code != http.StatusConflict {
		t.Fatalf("taken cadastral: %d", r.Code)
	}
	if r := e.do(http.MethodPut, path, super, map[string]any{"name": "ЖК А", "address": "Алматы", "cadastral_number": "21:3"}); r.Code != http.StatusBadRequest {
		t.Fatalf("invalid cadastral: %d", r.Code)
	}
	if r := e.do(http.MethodPut, path, super, map[string]any{"name": "", "address": "Алматы"}); r.Code != http.StatusBadRequest {
		t.Fatalf("empty name: %d", r.Code)
	}
	// Keeping its own cadastral number is not a conflict; clearing it is allowed.
	if r := e.do(http.MethodPut, "/platform/jks/"+out.JK.ID, super, map[string]any{"name": "Другое имя", "address": "x", "cadastral_number": "21:319:021:848:8"}); r.Code != http.StatusOK {
		t.Fatalf("same cadastral: %d %s", r.Code, r.Body)
	}
	if r := e.do(http.MethodPut, path, super, map[string]any{"name": "ЖК А", "address": "Алматы", "cadastral_number": ""}); r.Code != http.StatusOK {
		t.Fatalf("clear cadastral: %d", r.Code)
	}
	if r := e.do(http.MethodPut, "/platform/jks/"+uuid.NewString(), super, map[string]any{"name": "N", "address": "A"}); r.Code != http.StatusNotFound {
		t.Fatalf("unknown jk: %d", r.Code)
	}
}

func TestPlatformAdminsOfJK(t *testing.T) {
	e := newTestEnv(t)
	super := e.mkSuperadmin()
	path := "/platform/jks/" + e.jkA + "/admins"

	var admins []struct {
		ID         string `json:"id"`
		Email      string `json:"email"`
		MustChange bool   `json:"must_change_password"`
	}
	e.do(http.MethodGet, path, super, nil).json(t, &admins)
	if len(admins) != 1 || admins[0].ID != e.adminA {
		t.Fatalf("admins of jkA: %+v", admins)
	}

	r := e.do(http.MethodPost, path, super, map[string]any{"full_name": "Второй ОСИ", "email": "second@example.kz", "phone": "+77010000020"})
	if r.Code != http.StatusCreated {
		t.Fatalf("add admin: %d %s", r.Code, r.Body)
	}
	var added struct {
		Admin struct {
			ID string `json:"id"`
		} `json:"admin"`
		TemporaryPassword string `json:"temporary_password"`
	}
	r.json(t, &added)
	if e.loginCode("second@example.kz", added.TemporaryPassword) != http.StatusOK {
		t.Fatal("added admin must log in with the temporary password")
	}
	e.changeTempPassword(added.Admin.ID, added.TemporaryPassword)
	if r := e.do(http.MethodGet, "/rentals/jk/"+e.jkA, added.Admin.ID, nil); r.Code != http.StatusOK {
		t.Fatalf("added admin on own ЖК: %d", r.Code)
	}
	e.do(http.MethodGet, path, super, nil).json(t, &admins)
	if len(admins) != 2 {
		t.Fatalf("want 2 admins, got %d", len(admins))
	}

	if r := e.do(http.MethodPost, path, super, map[string]any{"full_name": "Дубль", "email": "SECOND@example.kz", "phone": "+77010000021"}); r.Code != http.StatusConflict {
		t.Fatalf("duplicate email: %d", r.Code)
	}
	if r := e.do(http.MethodPost, "/platform/jks/"+uuid.NewString()+"/admins", super, map[string]any{"full_name": "X", "email": "x@example.kz", "phone": "+77010000022"}); r.Code != http.StatusNotFound {
		t.Fatalf("unknown jk: %d", r.Code)
	}
	if r := e.do(http.MethodGet, "/platform/jks/"+uuid.NewString()+"/admins", super, nil); r.Code != http.StatusNotFound {
		t.Fatalf("list unknown jk: %d", r.Code)
	}
}

func TestPlatformResetAdminPassword(t *testing.T) {
	e := newTestEnv(t)
	super := e.mkSuperadmin()

	r := e.do(http.MethodPost, "/platform/admins/"+e.adminB+"/reset-password", super, nil)
	if r.Code != http.StatusOK {
		t.Fatalf("reset: %d %s", r.Code, r.Body)
	}
	var out struct {
		TemporaryPassword string `json:"temporary_password"`
	}
	r.json(t, &out)
	login := e.adminB + "@x.kz"
	if e.loginCode(login, "password123") != http.StatusUnauthorized || e.loginCode(login, out.TemporaryPassword) != http.StatusOK {
		t.Fatal("only the temporary password must work")
	}
	var mustChange bool
	_ = e.db.QueryRow(`SELECT must_change_password FROM users WHERE id=$1`, e.adminB).Scan(&mustChange)
	if !mustChange {
		t.Fatal("admin must be asked to choose a new password")
	}

	// Only ОСИ accounts: not residents, not other superadmins, not unknown ids.
	for _, id := range []string{e.resident, super, uuid.NewString()} {
		if r := e.do(http.MethodPost, "/platform/admins/"+id+"/reset-password", super, nil); r.Code != http.StatusNotFound {
			t.Errorf("reset %s: %d", id, r.Code)
		}
	}
}

// The superadmin has no ЖК: ОСИ and ЖК-scoped routes stay closed.
func TestSuperadminCannotUseOSIRoutes(t *testing.T) {
	e := newTestEnv(t)
	super := e.mkSuperadmin()
	for _, p := range []string{
		"/rentals/jk/" + e.jkA,
		"/payments/jk/" + e.jkA,
		"/residents/jk/" + e.jkA,
		"/dashboard/stats",
		"/requests/open-count",
		"/jks/" + e.jkA,
	} {
		if r := e.do(http.MethodGet, p, super, nil); r.Code != http.StatusForbidden {
			t.Errorf("GET %s as superadmin: %d", p, r.Code)
		}
	}

	// Login and /auth/me work with an empty ЖК.
	r := e.do(http.MethodPost, "/auth/login", "", map[string]any{"login": super + "@x.kz", "password": "password123"})
	if r.Code != http.StatusOK {
		t.Fatalf("superadmin login: %d %s", r.Code, r.Body)
	}
	var me struct {
		Role string  `json:"role"`
		JKID *string `json:"jk_id"`
	}
	e.do(http.MethodGet, "/auth/me", super, nil).json(t, &me)
	if me.Role != "superadmin" || me.JKID != nil {
		t.Fatalf("me: %+v", me)
	}
}

// The superadmin is created with `go run ./cmd/superadmin` (no env vars).
func TestSetSuperadmin(t *testing.T) {
	e := newTestEnv(t)
	ctx := context.Background()

	res, err := e.svc.SetSuperadmin(ctx, " Owner@Example.kz ", "bootstrap-pass")
	if err != nil || res != service.SuperadminCreated {
		t.Fatalf("first run: %v %v", res, err)
	}
	if e.loginCode("owner@example.kz", "bootstrap-pass") != http.StatusOK {
		t.Fatal("bootstrapped superadmin must log in")
	}

	// Running the command again for the same email sets a new password.
	res, err = e.svc.SetSuperadmin(ctx, "owner@example.kz", "another-pass")
	if err != nil || res != service.SuperadminPasswordReset {
		t.Fatalf("second run: %v %v", res, err)
	}
	var n int
	_ = e.db.QueryRow(`SELECT count(*) FROM users WHERE role='superadmin'`).Scan(&n)
	if n != 1 {
		t.Fatalf("want 1 superadmin, got %d", n)
	}
	if e.loginCode("owner@example.kz", "another-pass") != http.StatusOK {
		t.Fatal("new password must work")
	}
	if e.loginCode("owner@example.kz", "bootstrap-pass") == http.StatusOK {
		t.Fatal("old password must stop working")
	}

	// An email that belongs to an ОСИ is never promoted.
	res, err = e.svc.SetSuperadmin(ctx, e.adminA+"@x.kz", "whatever-pass")
	if err != nil || res != service.SuperadminConflict {
		t.Fatalf("other role: %v %v", res, err)
	}
	var role string
	_ = e.db.QueryRow(`SELECT role FROM users WHERE id=$1`, e.adminA).Scan(&role)
	if role != "admin" {
		t.Fatalf("admin role changed to %s", role)
	}

	// Invalid input is refused.
	if _, err := e.svc.SetSuperadmin(ctx, "not-an-email", "bootstrap-pass"); err == nil {
		t.Fatal("bad email must fail")
	}
	if _, err := e.svc.SetSuperadmin(ctx, "x@example.kz", "123"); err == nil {
		t.Fatal("short password must fail")
	}
}
