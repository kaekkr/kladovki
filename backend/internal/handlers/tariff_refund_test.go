package handlers

import (
	"net/http"
	"strings"
	"testing"
)

type rentalView struct {
	Status        string `json:"status"`
	TotalPaid     int64  `json:"total_paid"`
	RefundPending int64  `json:"refund_pending"`
}

func (e *testEnv) rentalView(userID, rentalID string) rentalView {
	e.t.Helper()
	var v rentalView
	r := e.do(http.MethodGet, "/rentals/"+rentalID, userID, nil)
	if r.Code != http.StatusOK {
		e.t.Fatalf("get rental: %d %s", r.Code, r.Body)
	}
	r.json(e.t, &v)
	return v
}

// QA: the ОСИ must hear about a refund choice, like about an early termination.
func TestTariffRefundChoiceNotifiesOSI(t *testing.T) {
	e := newTestEnv(t)
	e.activeRental(e.resident, e.storageA)
	before := kinds(e.notifications(e.adminA))["request_new"]
	e.chooseTariff(e.resident, 500, "refund")
	if got := kinds(e.notifications(e.adminA))["request_new"]; got != before+1 {
		t.Fatalf("ОСИ request_new notifications = %d, want %d", got, before+1)
	}
}

// QA: rejecting a tariff refund would leave the tenant with neither extension nor money.
func TestTariffRefundCannotBeRejected(t *testing.T) {
	e := newTestEnv(t)
	e.activeRental(e.resident, e.storageA)
	req := e.chooseTariff(e.resident, 500, "refund")
	r := e.do(http.MethodPatch, "/requests/"+req.ID+"/resolve", e.adminA, map[string]any{"action": "reject", "comment": "нет"})
	if r.Code != http.StatusConflict || !strings.Contains(string(r.Body), "отклонить нельзя") {
		t.Fatalf("reject tariff refund: %d %s, want 409", r.Code, r.Body)
	}
	if got := e.adminRequests(e.adminA)[0]; got.Status != "open" {
		t.Fatalf("request = %+v, must stay open", got)
	}
}

// «Оплачено» shows what was paid until the money is actually returned; the
// rental tells the UI how much is to be returned.
func TestTariffRefundPaidAmountUntilReturned(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	req := e.chooseTariff(e.resident, 500, "refund")

	v := e.rentalView(e.resident, id)
	if v.TotalPaid != 10000 || !near(v.RefundPending, 2500) {
		t.Fatalf("before return: %+v, want total_paid 10000 + refund_pending 2500", v)
	}
	if r := e.do(http.MethodPatch, "/requests/"+req.ID+"/resolve", e.adminA, map[string]any{"action": "approve"}); r.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", r.Code, r.Body)
	}
	v = e.rentalView(e.resident, id)
	if !near(v.TotalPaid, 7500) || v.RefundPending != 0 {
		t.Fatalf("after return: %+v, want total_paid 7500, refund_pending 0", v)
	}
}

// The tariff refund is owed even if the rental ends before the ОСИ pays it.
func TestTariffRefundSurvivesRentalEnd(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.chooseTariff(e.resident, 500, "refund")
	if r := e.do(http.MethodPatch, "/rentals/"+id+"/cancel", e.adminA, nil); r.Code != http.StatusOK {
		t.Fatalf("cancel: %d %s", r.Code, r.Body)
	}
	var tariff, cancel int64
	for _, q := range e.adminRequests(e.adminA) {
		if q.Status != "open" {
			continue
		}
		if q.Type == "tariff_recalc" {
			tariff = q.Amount
		}
		if q.Type == "payment_refund" {
			cancel = q.Amount
		}
	}
	// 5000 unused: 2500 via the tariff refund + 2500 for the cancelled time.
	if !near(tariff, 2500) || !near(cancel, 2500) {
		t.Fatalf("open refunds: tariff %d, cancel %d, want 2500 + 2500", tariff, cancel)
	}
}
