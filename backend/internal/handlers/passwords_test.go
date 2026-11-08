package handlers

import (
	"net/http"
	"testing"
)

func (e *testEnv) loginCode(login, password string) int {
	return e.do(http.MethodPost, "/auth/login", "", map[string]any{"login": login, "password": password}).Code
}

func TestChangeOwnPassword(t *testing.T) {
	e := newTestEnv(t)
	path := "/me/password"
	if r := e.do(http.MethodPut, path, e.resident, map[string]any{"current_password": "wrong", "new_password": "newpass1"}); r.Code != http.StatusBadRequest {
		t.Fatalf("wrong current: %d", r.Code)
	}
	if r := e.do(http.MethodPut, path, e.resident, map[string]any{"current_password": "password123", "new_password": "123"}); r.Code != http.StatusBadRequest {
		t.Fatalf("short new: %d", r.Code)
	}
	if r := e.do(http.MethodPut, path, e.resident, map[string]any{"current_password": "password123", "new_password": "newpass1"}); r.Code != http.StatusOK {
		t.Fatalf("change: %d %s", r.Code, r.Body)
	}
	if e.loginCode("900101300001", "newpass1") != http.StatusOK || e.loginCode("900101300001", "password123") != http.StatusUnauthorized {
		t.Fatal("new password must work and the old one must not")
	}
	// ОСИ can change theirs too.
	if r := e.do(http.MethodPut, path, e.adminA, map[string]any{"current_password": "password123", "new_password": "admin-new"}); r.Code != http.StatusOK {
		t.Fatalf("admin change: %d", r.Code)
	}
}

// The login page says «забыли пароль? обратитесь в ОСИ» — the ОСИ needs a way to help.
func TestOSIResetsResidentPassword(t *testing.T) {
	e := newTestEnv(t)
	path := "/residents/" + e.resident + "/reset-password"
	if r := e.do(http.MethodPost, path, e.adminB, nil); r.Code != http.StatusNotFound {
		t.Fatalf("other ЖК admin: %d", r.Code)
	}
	if r := e.do(http.MethodPost, path, e.other, nil); r.Code != http.StatusForbidden {
		t.Fatalf("resident: %d", r.Code)
	}
	if r := e.do(http.MethodPost, "/residents/"+e.adminB+"/reset-password", e.adminA, nil); r.Code != http.StatusNotFound {
		t.Fatalf("resetting an admin: %d", r.Code)
	}

	r := e.do(http.MethodPost, path, e.adminA, nil)
	if r.Code != http.StatusOK {
		t.Fatalf("reset: %d %s", r.Code, r.Body)
	}
	var out struct {
		TemporaryPassword string `json:"temporary_password"`
	}
	r.json(t, &out)
	if len(out.TemporaryPassword) < 8 {
		t.Fatalf("temporary password too short: %q", out.TemporaryPassword)
	}
	if e.loginCode("900101300001", "password123") != http.StatusUnauthorized || e.loginCode("900101300001", out.TemporaryPassword) != http.StatusOK {
		t.Fatal("only the temporary password must work")
	}

	var me struct {
		MustChange bool `json:"must_change_password"`
	}
	e.do(http.MethodGet, "/auth/me", e.resident, nil).json(t, &me)
	if !me.MustChange {
		t.Fatal("resident must be asked to set a new password")
	}
	e.do(http.MethodPut, "/me/password", e.resident, map[string]any{"current_password": out.TemporaryPassword, "new_password": "my-own-1"})
	e.do(http.MethodGet, "/auth/me", e.resident, nil).json(t, &me)
	if me.MustChange {
		t.Fatal("flag must clear after the resident sets their own password")
	}
}
