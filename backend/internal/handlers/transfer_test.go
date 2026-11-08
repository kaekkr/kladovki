package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/kaekkr/kladovki/internal/repository"
)

func (e *testEnv) storageState(id string) (status, transferIIN string) {
	e.t.Helper()
	_ = e.db.QueryRow(`SELECT status, COALESCE(transfer_to_iin,'') FROM storages WHERE id=$1`, id).Scan(&status, &transferIIN)
	return
}

func (e *testEnv) runExpiryWorker() {
	e.t.Helper()
	if err := repository.New(e.db).ExpireRentals(context.Background()); err != nil {
		e.t.Fatal(err)
	}
}

// setUpHandover: the resident's paid period ended yesterday; they named `other` as successor.
func (e *testEnv) setUpHandover() (oldRental string) {
	e.t.Helper()
	oldRental = e.activeRental(e.resident, e.storageA)
	e.exec(`UPDATE rentals SET ends_at = now() - interval '1 day' WHERE id=$1`, oldRental)
	if r := e.do(http.MethodPatch, "/storages/"+e.storageA+"/transfer", e.resident, map[string]any{"iin": "900101300002"}); r.Code != http.StatusOK {
		e.t.Fatalf("set transfer: %d %s", r.Code, r.Body)
	}
	return oldRental
}

// The successor books but never pays: the old tenant must keep the storage, not lose it to a dead lock.
func TestUnpaidTakeoverDoesNotStrandStorage(t *testing.T) {
	e := newTestEnv(t)
	oldRental := e.setUpHandover()

	lock := e.lockAs(e.other, e.storageA, 1)
	if s := e.rentalState(oldRental); s.Status != "active" {
		t.Fatalf("old rental must stay active until the successor pays, got %s", s.Status)
	}

	e.exec(`UPDATE rentals SET ends_at = now() - interval '1 minute' WHERE id=$1`, lock)
	e.runExpiryWorker()
	if st, iin := e.storageState(e.storageA); st != "occupied" || iin == "" {
		t.Fatalf("after the unpaid takeover: status=%s transfer=%q, want occupied + transfer kept", st, iin)
	}

	// The handover window passes: the old tenant must be able to renew again.
	e.exec(`UPDATE storages SET transfer_until = now() - interval '1 hour' WHERE id=$1`, e.storageA)
	if code, _, _ := e.renew(e.resident, oldRental, 1); code != http.StatusCreated {
		t.Fatalf("renew after lapsed transfer: %d, want 201", code)
	}
	if _, iin := e.storageState(e.storageA); iin != "" {
		t.Fatalf("lapsed transfer must be cleared, got %q", iin)
	}
}

func TestPaidTakeoverClosesOldRental(t *testing.T) {
	e := newTestEnv(t)
	oldRental := e.setUpHandover()
	lock := e.lockAs(e.other, e.storageA, 1)
	if code := e.pay(e.other, lock); code != http.StatusOK {
		t.Fatalf("pay: %d", code)
	}
	e.webhook(paidEvent(e.paymentIDFor(lock)), "valid")

	if s := e.rentalState(oldRental); s.Status != "expired" {
		t.Fatalf("old rental = %s, want expired", s.Status)
	}
	if s := e.rentalState(lock); s.Status != "active" {
		t.Fatalf("successor rental = %s, want active", s.Status)
	}
	if st, iin := e.storageState(e.storageA); st != "occupied" || iin != "" {
		t.Fatalf("storage = %s transfer=%q", st, iin)
	}
}

// A lapsed transfer on a free storage must not block anyone from booking it.
func TestLapsedTransferDoesNotBlockBooking(t *testing.T) {
	e := newTestEnv(t)
	e.exec(`UPDATE storages SET transfer_to_iin='999999999999', transfer_until=now() - interval '1 day' WHERE id=$1`, e.storage2)
	if r := e.do(http.MethodPost, "/rentals/lock", e.resident, map[string]any{"storage_id": e.storage2, "months": 1}); r.Code != http.StatusCreated {
		t.Fatalf("lock: %d %s", r.Code, r.Body)
	}
	if _, iin := e.storageState(e.storage2); iin != "" {
		t.Fatalf("lapsed transfer must be cleared, got %q", iin)
	}
}

// ОСИ force-release of a successor's unpaid booking returns the storage to the current tenant.
func TestForceReleaseOfTakeoverLockKeepsStorageOccupied(t *testing.T) {
	e := newTestEnv(t)
	e.setUpHandover()
	lock := e.lockAs(e.other, e.storageA, 1)
	if r := e.do(http.MethodPatch, "/rentals/"+lock+"/force-release", e.adminA, nil); r.Code != http.StatusOK {
		t.Fatalf("force-release: %d %s", r.Code, r.Body)
	}
	if st, _ := e.storageState(e.storageA); st != "occupied" {
		t.Fatalf("storage = %s, want occupied", st)
	}
}
