package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type notification struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Read  bool   `json:"read"`
}

func (e *testEnv) notifications(userID string) []notification {
	e.t.Helper()
	r := e.do(http.MethodGet, "/notifications", userID, nil)
	if r.Code != http.StatusOK {
		e.t.Fatalf("notifications: %d %s", r.Code, r.Body)
	}
	var out []notification
	r.json(e.t, &out)
	return out
}

func kinds(list []notification) map[string]int {
	m := map[string]int{}
	for _, n := range list {
		m[n.Kind]++
	}
	return m
}

func TestNotificationsAreprivate_AndCanBeRead(t *testing.T) {
	e := newTestEnv(t)
	e.activeRental(e.resident, e.storageA) // → payment_success notification

	mine := e.notifications(e.resident)
	if kinds(mine)["payment_success"] != 1 {
		t.Fatalf("resident notifications = %+v", mine)
	}
	if got := e.notifications(e.other); len(got) != 0 {
		t.Fatalf("other resident sees %+v", got)
	}

	var count struct {
		Unread int `json:"unread"`
	}
	e.do(http.MethodGet, "/notifications/unread-count", e.resident, nil).json(t, &count)
	if count.Unread != 1 {
		t.Fatalf("unread = %d", count.Unread)
	}
	if r := e.do(http.MethodPost, "/notifications/read-all", e.resident, nil); r.Code != http.StatusOK {
		t.Fatalf("read-all: %d", r.Code)
	}
	e.do(http.MethodGet, "/notifications/unread-count", e.resident, nil).json(t, &count)
	if count.Unread != 0 {
		t.Fatalf("unread after read-all = %d", count.Unread)
	}
}

func TestReminderBeforeRentalEnds_OnlyOnce(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.exec(`UPDATE rentals SET ends_at = now() + interval '6 days' WHERE id=$1`, id)

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if err := e.svc.RunReminders(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if n := kinds(e.notifications(e.resident))["rental_expiring"]; n != 1 {
		t.Fatalf("rental_expiring notifications = %d, want 1", n)
	}
}

func TestOverdueNotifiesTenantAndOSI(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.exec(`UPDATE rentals SET starts_at = now() - interval '40 days', ends_at = now() - interval '3 days' WHERE id=$1`, id)

	if err := e.svc.RunReminders(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := kinds(e.notifications(e.resident))["rental_overdue"]; n != 1 {
		t.Fatalf("tenant rental_overdue = %d", n)
	}
	if n := kinds(e.notifications(e.adminA))["rental_overdue"]; n != 1 {
		t.Fatalf("admin rental_overdue = %d", n)
	}
	if n := len(e.notifications(e.adminB)); n != 0 {
		t.Fatalf("admin of other JK got %d notifications", n)
	}
}

func TestDebtOverTwoMonthsWarning(t *testing.T) {
	e := newTestEnv(t)
	id := e.activeRental(e.resident, e.storageA)
	e.exec(`UPDATE rentals SET starts_at = now() - interval '150 days', ends_at = now() - interval '61 days' WHERE id=$1`, id)
	if err := e.svc.RunReminders(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := kinds(e.notifications(e.resident))["debt_warning"]; n != 1 {
		t.Fatalf("debt_warning = %d", n)
	}
}

func TestTariffAndRequestEventsNotify(t *testing.T) {
	e := newTestEnv(t)
	rentalID := e.activeRental(e.resident, e.storageA)

	e.do(http.MethodPut, "/jks/"+e.jkA+"/tariff", e.adminA, map[string]any{"amount": 800})
	if n := kinds(e.notifications(e.resident))["tariff_recalc"]; n != 1 {
		t.Fatalf("tariff_recalc = %d", n)
	}
	e.do(http.MethodPut, "/jks/"+e.jkA+"/tariff", e.adminA, map[string]any{"amount": 1200})
	if n := kinds(e.notifications(e.resident))["tariff_changed"]; n != 1 {
		t.Fatalf("tariff_changed on increase = %d", n)
	}

	e.do(http.MethodPost, "/rentals/"+rentalID+"/terminate", e.resident, nil)
	if n := kinds(e.notifications(e.adminA))["request_new"]; n != 1 {
		t.Fatalf("admin request_new = %d", n)
	}
	reqs := e.adminRequests(e.adminA)
	var termID string
	for _, q := range reqs {
		if q.Type == "early_termination" {
			termID = q.ID
		}
	}
	e.do(http.MethodPatch, "/requests/"+termID+"/resolve", e.adminA, map[string]any{"action": "approve"})
	if n := kinds(e.notifications(e.resident))["request_resolved"]; n != 1 {
		t.Fatalf("request_resolved = %d", n)
	}
}

func TestTelegramLinkingAndDelivery(t *testing.T) {
	e := newTestEnv(t)

	r := e.do(http.MethodPost, "/me/telegram/link", e.resident, nil)
	if r.Code != http.StatusOK {
		t.Fatalf("link: %d %s", r.Code, r.Body)
	}
	var link struct {
		URL string `json:"url"`
	}
	r.json(t, &link)
	const prefix = "https://t.me/kladovki_test_bot?start="
	if !strings.HasPrefix(link.URL, prefix) {
		t.Fatalf("link url = %q", link.URL)
	}
	token := strings.TrimPrefix(link.URL, prefix)

	update := map[string]any{"message": map[string]any{"chat": map[string]any{"id": 999}, "text": "/start " + token}}
	if code := e.telegramWebhook(update, "wrong"); code != http.StatusUnauthorized {
		t.Fatalf("wrong secret: want 401, got %d", code)
	}
	if code := e.telegramWebhook(update, telegramSecret); code != http.StatusOK {
		t.Fatalf("webhook: %d", code)
	}

	var ch struct {
		Telegram bool `json:"telegram"`
	}
	e.do(http.MethodGet, "/me/channels", e.resident, nil).json(t, &ch)
	if !ch.Telegram {
		t.Fatal("telegram should be linked")
	}

	e.activeRental(e.resident, e.storageA) // payment_success → telegram delivery
	if err := e.svc.DispatchPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	sent := e.telegram.messages()
	var toUser int
	for _, m := range sent {
		if m.ChatID == "999" && strings.Contains(m.Text, "Оплата") {
			toUser++
		}
	}
	if toUser != 1 {
		t.Fatalf("telegram messages = %+v", sent)
	}

	// Unlink.
	if r := e.do(http.MethodDelete, "/me/telegram", e.resident, nil); r.Code != http.StatusOK {
		t.Fatalf("unlink: %d", r.Code)
	}
	e.do(http.MethodGet, "/me/channels", e.resident, nil).json(t, &ch)
	if ch.Telegram {
		t.Fatal("telegram should be unlinked")
	}
}

func (e *testEnv) telegramWebhook(update any, secret string) int {
	b, _ := json.Marshal(update)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telegram/webhook", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w.Code
}
