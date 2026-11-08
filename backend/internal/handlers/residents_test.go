package handlers

import (
	"net/http"
	"testing"
)

type residentView struct {
	ID             string `json:"id"`
	FullName       string `json:"full_name"`
	IIN            string `json:"iin"`
	Status         string `json:"status"`
	ReviewComment  string `json:"review_comment"`
	OwnerType      string `json:"owner_type"`
	PremisesKind   string `json:"premises_kind"`
	PremisesNumber string `json:"premises_number"`
	CompanyBIN     string `json:"company_bin"`
	CompanyName    string `json:"company_name"`
}

// register signs up a new resident in jkA and returns their user id.
func (e *testEnv) register(body map[string]any) string {
	e.t.Helper()
	base := map[string]any{
		"iin": "010101500009", "full_name": "Асель", "phone": "+77019998877",
		"password": "secret1", "jk_id": e.jkA, "premises_number": "45",
	}
	for k, v := range body {
		base[k] = v
	}
	r := e.do(http.MethodPost, "/auth/register", "", base)
	if r.Code != http.StatusOK {
		e.t.Fatalf("register: %d %s", r.Code, r.Body)
	}
	var u residentView
	r.json(e.t, &u)
	return u.ID
}

func (e *testEnv) me(userID string) residentView {
	e.t.Helper()
	var u residentView
	e.do(http.MethodGet, "/auth/me", userID, nil).json(e.t, &u)
	return u
}

func (e *testEnv) residents(adminID, jkID, status string) (int, []residentView) {
	e.t.Helper()
	r := e.do(http.MethodGet, "/residents/jk/"+jkID+"?status="+status, adminID, nil)
	var list []residentView
	if r.Code == http.StatusOK {
		r.json(e.t, &list)
	}
	return r.Code, list
}

func (e *testEnv) review(adminID, userID, action, comment string) int {
	e.t.Helper()
	return e.do(http.MethodPatch, "/residents/"+userID+"/"+action, adminID, map[string]any{"comment": comment}).Code
}

func TestNewResidentIsPendingAndCannotBook(t *testing.T) {
	e := newTestEnv(t)
	id := e.register(map[string]any{"premises_kind": "flat", "premises_number": "45"})

	if u := e.me(id); u.Status != "pending" || u.PremisesNumber != "45" || u.OwnerType != "person" {
		t.Fatalf("me = %+v", u)
	}
	if e.me(e.resident).Status != "active" {
		t.Fatal("existing residents must stay active")
	}
	r := e.do(http.MethodPost, "/rentals/lock", id, map[string]any{"storage_id": e.storageA, "months": 1})
	if r.Code != http.StatusForbidden {
		t.Fatalf("pending user booking: %d, want 403", r.Code)
	}
	if n := kinds(e.notifications(e.adminA))["resident_new"]; n != 1 {
		t.Fatalf("ОСИ must be notified about the new resident, got %d", n)
	}
	if n := len(e.notifications(e.adminB)); n != 0 {
		t.Fatalf("other ЖК admin notified: %d", n)
	}
}

func TestCompanyRegistration(t *testing.T) {
	e := newTestEnv(t)
	id := e.register(map[string]any{
		"owner_type": "company", "company_bin": "180940012345", "company_name": "ТОО Аптека Плюс",
		"premises_kind": "commercial", "premises_number": "3",
	})
	_, list := e.residents(e.adminA, e.jkA, "pending")
	if len(list) != 1 || list[0].ID != id || list[0].CompanyBIN != "180940012345" ||
		list[0].CompanyName != "ТОО Аптека Плюс" || list[0].PremisesKind != "commercial" || list[0].IIN != "010101500009" {
		t.Fatalf("pending list = %+v", list)
	}
}

func TestOSIApprovesResident(t *testing.T) {
	e := newTestEnv(t)
	id := e.register(nil)

	if code, _ := e.residents(e.adminB, e.jkA, "pending"); code != http.StatusForbidden {
		t.Fatalf("other ЖК admin list: %d", code)
	}
	if code, _ := e.residents(e.resident, e.jkA, "pending"); code != http.StatusForbidden {
		t.Fatalf("resident list: %d", code)
	}
	if code := e.review(e.adminB, id, "approve", ""); code != http.StatusNotFound {
		t.Fatalf("other ЖК admin approve: %d, want 404", code)
	}
	if code := e.review(e.adminA, e.adminB, "approve", ""); code != http.StatusNotFound {
		t.Fatalf("approving an admin: %d, want 404", code)
	}

	var count struct {
		Pending int `json:"pending"`
	}
	e.do(http.MethodGet, "/residents/pending-count", e.adminA, nil).json(t, &count)
	if count.Pending != 1 {
		t.Fatalf("pending count = %d", count.Pending)
	}

	if code := e.review(e.adminA, id, "approve", ""); code != http.StatusOK {
		t.Fatalf("approve: %d", code)
	}
	if e.me(id).Status != "active" {
		t.Fatal("status should be active")
	}
	if n := kinds(e.notifications(id))["account_status"]; n != 1 {
		t.Fatalf("resident not told about approval: %d", n)
	}
	if r := e.do(http.MethodPost, "/rentals/lock", id, map[string]any{"storage_id": e.storageA, "months": 1}); r.Code != http.StatusCreated {
		t.Fatalf("approved user booking: %d %s", r.Code, r.Body)
	}
	if _, all := e.residents(e.adminA, e.jkA, ""); len(all) != 3 { // resident, other, new one
		t.Fatalf("all residents = %d", len(all))
	}
}

func TestRejectedResidentCanFixAndResubmit(t *testing.T) {
	e := newTestEnv(t)
	id := e.register(map[string]any{"premises_number": "999"})

	if code := e.review(e.adminA, id, "reject", ""); code != http.StatusBadRequest {
		t.Fatalf("reject without reason: %d, want 400", code)
	}
	if code := e.review(e.adminA, id, "reject", "Квартиры 999 нет"); code != http.StatusOK {
		t.Fatalf("reject: %d", code)
	}
	if u := e.me(id); u.Status != "rejected" || u.ReviewComment != "Квартиры 999 нет" {
		t.Fatalf("me = %+v", u)
	}

	r := e.do(http.MethodPut, "/me/verification", id, map[string]any{"premises_kind": "flat", "premises_number": "99"})
	if r.Code != http.StatusOK {
		t.Fatalf("resubmit: %d %s", r.Code, r.Body)
	}
	if u := e.me(id); u.Status != "pending" || u.PremisesNumber != "99" {
		t.Fatalf("after resubmit = %+v", u)
	}
	if n := kinds(e.notifications(e.adminA))["resident_new"]; n != 2 {
		t.Fatalf("ОСИ must be notified again, got %d", n)
	}
	// An active resident cannot reset themselves to pending.
	if r := e.do(http.MethodPut, "/me/verification", e.resident, map[string]any{"premises_number": "1"}); r.Code != http.StatusConflict {
		t.Fatalf("active resubmit: %d, want 409", r.Code)
	}
}

func TestBlockedResidentCannotBookOrRenew(t *testing.T) {
	e := newTestEnv(t)
	rentalID := e.activeRental(e.resident, e.storageA)

	if code := e.review(e.adminA, e.resident, "block", "Переехал"); code != http.StatusOK {
		t.Fatalf("block: %d", code)
	}
	if r := e.do(http.MethodPost, "/rentals/lock", e.resident, map[string]any{"storage_id": e.storage2, "months": 1}); r.Code != http.StatusForbidden {
		t.Fatalf("blocked booking: %d", r.Code)
	}
	if code, _, _ := e.renew(e.resident, rentalID, 1); code != http.StatusForbidden {
		t.Fatalf("blocked renewal: %d", code)
	}
	// The existing paid rental stays.
	if s := e.rentalState(rentalID); s.Status != "active" {
		t.Fatalf("rental status = %s", s.Status)
	}
	if code := e.review(e.adminA, e.resident, "approve", ""); code != http.StatusOK {
		t.Fatalf("unblock: %d", code)
	}
	if code, _, _ := e.renew(e.resident, rentalID, 1); code != http.StatusCreated {
		t.Fatalf("renew after unblock: %d", code)
	}
}
