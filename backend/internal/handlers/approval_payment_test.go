package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/kaekkr/kladovki/internal/service"
)

// QA: a resident blocked/rejected after booking could still pay the earlier hold.
func TestPaymentRequiresApprovedAccount(t *testing.T) {
	cases := []struct{ status, want string }{
		{"blocked", "заблокирован"},
		{"rejected", "отклонил"},
		{"pending", "ожидает подтверждения"},
	}
	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			e := newTestEnv(t)
			id := e.lockAs(e.resident, e.storageA, 1)
			e.exec(`UPDATE users SET status=$2 WHERE id=$1`, e.resident, tc.status)

			r := e.do(http.MethodPost, "/payments", e.resident, map[string]any{"rental_id": id, "phone": "+77001112233", "accepted_terms_version": termsV})
			if r.Code != http.StatusForbidden || !strings.Contains(string(r.Body), tc.want) {
				t.Fatalf("pay: %d %s, want 403 containing %q", r.Code, r.Body, tc.want)
			}
			if n := e.count(`SELECT count(*) FROM payments WHERE rental_id=$1`, id); n != 0 {
				t.Fatalf("payments = %d, want 0", n)
			}
			// The development shortcut must refuse too.
			_, err := e.svc.ConfirmPayment(context.Background(), service.ConfirmPaymentInput{RentalID: id, UserID: e.resident})
			if !errors.Is(err, service.ErrNotApproved) {
				t.Fatalf("dev confirm: %v, want ErrNotApproved", err)
			}
			// Booking and renewal give the same status-specific message.
			r = e.do(http.MethodPost, "/rentals/lock", e.resident, map[string]any{"storage_id": e.storage2, "months": 1})
			if r.Code != http.StatusForbidden || !strings.Contains(string(r.Body), tc.want) {
				t.Fatalf("lock: %d %s", r.Code, r.Body)
			}
		})
	}
}
