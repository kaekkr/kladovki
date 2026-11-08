package notify

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

var msg = Message{Title: "Срок аренды истекает", Body: "Кладовка №12 — до 15.11", Link: "/app/rentals/r1"}
var who = Recipient{Name: "Иван", Email: "ivan@example.kz", Phone: "+7 (700) 111-22-33", TelegramChatID: "555"}

func TestChannelsOnlyWhenConfigured(t *testing.T) {
	if got := (Config{}).Channels(); len(got) != 0 {
		t.Fatalf("no config → no channels, got %d", len(got))
	}
	c := Config{SMTPHost: "h", SMTPFrom: "f", TelegramToken: "t", MobizonAPIKey: "k", GreenAPIInstanceID: "i", GreenAPIToken: "t"}
	names := []string{}
	for _, ch := range c.Channels() {
		names = append(names, ch.Name())
	}
	if strings.Join(names, ",") != "email,telegram,sms,whatsapp" {
		t.Fatalf("channels = %v", names)
	}
}

func TestTelegramSend(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/botTOKEN/sendMessage" {
			t.Errorf("path %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tg := NewTelegram(Config{TelegramToken: "TOKEN", TelegramAPIBase: srv.URL, AppBaseURL: "https://app.kz"})
	if !tg.CanReach(who) || tg.CanReach(Recipient{}) {
		t.Fatal("CanReach wrong")
	}
	if err := tg.Send(context.Background(), who, msg); err != nil {
		t.Fatal(err)
	}
	if got["chat_id"] != "555" || !strings.Contains(got["text"].(string), "Кладовка №12") ||
		!strings.Contains(got["text"].(string), "https://app.kz/app/rentals/r1") {
		t.Fatalf("payload %v", got)
	}
}

func TestTelegramErrorIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"ok":false,"description":"bot was blocked by the user"}`))
	}))
	defer srv.Close()
	tg := NewTelegram(Config{TelegramToken: "T", TelegramAPIBase: srv.URL})
	if err := tg.Send(context.Background(), who, msg); err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("want blocked error, got %v", err)
	}
}

func TestSMSMobizon(t *testing.T) {
	var form url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.URL.Path, "/service/Message/SendSmsMessage") || r.URL.Query().Get("apiKey") != "KEY" {
			t.Errorf("url %s", r.URL)
		}
		_ = r.ParseForm()
		form = r.PostForm
		_, _ = w.Write([]byte(`{"code":0,"data":{"messageId":"1"},"message":""}`))
	}))
	defer srv.Close()
	sms := &SMS{cfg: Config{MobizonAPIKey: "KEY", MobizonAPIBase: srv.URL}}
	if err := sms.Send(context.Background(), who, msg); err != nil {
		t.Fatal(err)
	}
	if form.Get("recipient") != "77001112233" || !strings.Contains(form.Get("text"), "Кладовка №12") {
		t.Fatalf("form %v", form)
	}
}

func TestSMSMobizonError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":1,"message":"Недостаточно средств"}`))
	}))
	defer srv.Close()
	sms := &SMS{cfg: Config{MobizonAPIKey: "KEY", MobizonAPIBase: srv.URL}}
	if err := sms.Send(context.Background(), who, msg); err == nil {
		t.Fatal("want error")
	}
}

func TestWhatsAppGreenAPI(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/waInstance1101/sendMessage/TOK" {
			t.Errorf("path %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"idMessage":"ABC"}`))
	}))
	defer srv.Close()
	wa := &WhatsApp{cfg: Config{GreenAPIInstanceID: "1101", GreenAPIToken: "TOK", GreenAPIBase: srv.URL}}
	if err := wa.Send(context.Background(), who, msg); err != nil {
		t.Fatal(err)
	}
	if got["chatId"] != "77001112233@c.us" || !strings.Contains(got["message"].(string), "Срок аренды") {
		t.Fatalf("payload %v", got)
	}
}

// fakeSMTP accepts one message and records it.
func fakeSMTP(t *testing.T) (addr string, received func() string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var data strings.Builder
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
		say := func(s string) { _, _ = rw.WriteString(s + "\r\n"); _ = rw.Flush() }
		say("220 fake")
		inData := false
		for {
			line, err := rw.ReadString('\n')
			if err != nil {
				return
			}
			if inData {
				if line == ".\r\n" {
					inData = false
					say("250 ok")
					continue
				}
				mu.Lock()
				data.WriteString(line)
				mu.Unlock()
				continue
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				say("250 fake")
			case strings.HasPrefix(cmd, "DATA"):
				inData = true
				say("354 go")
			case strings.HasPrefix(cmd, "QUIT"):
				say("221 bye")
				return
			default:
				say("250 ok")
			}
		}
	}()
	t.Cleanup(func() { ln.Close(); <-done })
	return ln.Addr().String(), func() string { mu.Lock(); defer mu.Unlock(); return data.String() }
}

func TestEmailSMTP(t *testing.T) {
	addr, received := fakeSMTP(t)
	host, port, _ := net.SplitHostPort(addr)
	e := &Email{cfg: Config{SMTPHost: host, SMTPPort: port, SMTPFrom: "noreply@kladovki.kz", AppBaseURL: "https://app.kz"}}
	if !e.CanReach(who) || e.CanReach(Recipient{}) {
		t.Fatal("CanReach wrong")
	}
	if err := e.Send(context.Background(), who, msg); err != nil {
		t.Fatal(err)
	}
	got := received()
	for _, want := range []string{"To: ivan@example.kz", "Subject: =?utf-8?", "https://app.kz/app/rentals/r1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("mail missing %q:\n%s", want, got)
		}
	}
}
