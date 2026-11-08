package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

// ---------- Email (SMTP) ----------

// Email sends plain-text mail over SMTP (STARTTLS when the server offers it).
// Free options for testing: a Gmail account with an app password
// (smtp.gmail.com:587), or Brevo / Mailjet free SMTP plans.
type Email struct{ cfg Config }

func (e *Email) Name() string              { return "email" }
func (e *Email) CanReach(r Recipient) bool { return strings.Contains(r.Email, "@") }

func (e *Email) Send(ctx context.Context, r Recipient, m Message) error {
	body := m.Body
	if l := e.cfg.link(m.Link); l != "" {
		body += "\n\nОткрыть в приложении: " + l
	}
	body += "\n\n—\nКладовки"

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", e.cfg.SMTPFrom)
	fmt.Fprintf(&b, "To: %s\r\n", r.Email)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", m.Title))
	b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n")
	b.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))

	addr := net.JoinHostPort(e.cfg.SMTPHost, e.cfg.SMTPPort)
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	c, err := smtp.NewClient(conn, e.cfg.SMTPHost)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(tlsConfig(e.cfg.SMTPHost)); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if e.cfg.SMTPUser != "" {
		if err := c.Auth(smtp.PlainAuth("", e.cfg.SMTPUser, e.cfg.SMTPPassword, e.cfg.SMTPHost)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := c.Mail(addressOnly(e.cfg.SMTPFrom)); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := c.Rcpt(r.Email); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := io.WriteString(w, b.String()); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return c.Quit()
}

// addressOnly turns "Кладовки <noreply@x.kz>" into "noreply@x.kz".
func addressOnly(from string) string {
	if i := strings.LastIndex(from, "<"); i >= 0 {
		return strings.TrimSuffix(from[i+1:], ">")
	}
	return from
}

// ---------- Telegram ----------

// Telegram sends messages through a bot. Free. A user links their chat by
// opening t.me/<bot>?start=<token> (see LinkURL) — the bot webhook then
// stores their chat id.
type Telegram struct {
	cfg  Config
	base string
}

func NewTelegram(c Config) *Telegram {
	base := c.TelegramAPIBase
	if base == "" {
		base = "https://api.telegram.org"
	}
	return &Telegram{cfg: c, base: base}
}

func (t *Telegram) Name() string              { return "telegram" }
func (t *Telegram) CanReach(r Recipient) bool { return r.TelegramChatID != "" }

func (t *Telegram) Send(ctx context.Context, r Recipient, m Message) error {
	return t.SendText(ctx, r.TelegramChatID, text(m, t.cfg.link(m.Link)))
}

// SendText posts a raw text message to a chat.
func (t *Telegram) SendText(ctx context.Context, chatID, msg string) error {
	return postJSON(ctx, t.base+"/bot"+t.cfg.TelegramToken+"/sendMessage",
		map[string]any{"chat_id": chatID, "text": msg, "disable_web_page_preview": true},
		func(body []byte) error {
			var resp struct {
				OK          bool   `json:"ok"`
				Description string `json:"description"`
			}
			if json.Unmarshal(body, &resp) != nil || !resp.OK {
				return fmt.Errorf("telegram: %s", resp.Description)
			}
			return nil
		})
}

// LinkURL is the deep link that connects a user's Telegram to their account.
func (c Config) TelegramLinkURL(token string) string {
	if c.TelegramBotUsername == "" {
		return ""
	}
	return "https://t.me/" + c.TelegramBotUsername + "?start=" + url.QueryEscape(token)
}

// ---------- SMS (Mobizon) ----------

// SMS sends through Mobizon (mobizon.kz), widely used in Kazakhstan. Paid per
// message; a sender name must be approved by Mobizon before production use.
type SMS struct{ cfg Config }

func (s *SMS) Name() string              { return "sms" }
func (s *SMS) CanReach(r Recipient) bool { return len(digits(r.Phone)) == 11 }

func (s *SMS) Send(ctx context.Context, r Recipient, m Message) error {
	endpoint := strings.TrimRight(s.cfg.MobizonAPIBase, "/") + "/service/Message/SendSmsMessage?" +
		url.Values{"output": {"json"}, "api": {"v1"}, "apiKey": {s.cfg.MobizonAPIKey}}.Encode()
	form := url.Values{"recipient": {digits(r.Phone)}, "text": {m.Title + ". " + m.Body}}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return do(req, func(body []byte) error {
		var resp struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &resp) != nil || resp.Code != 0 {
			return fmt.Errorf("mobizon: code %d %s", resp.Code, resp.Message)
		}
		return nil
	})
}

// ---------- WhatsApp (Green API) ----------

// WhatsApp sends through Green API (green-api.com) — a WhatsApp account linked
// by QR code. Has a free developer plan for testing. For large volume the
// official Meta WhatsApp Cloud API (template messages) is the alternative.
type WhatsApp struct{ cfg Config }

func (w *WhatsApp) Name() string              { return "whatsapp" }
func (w *WhatsApp) CanReach(r Recipient) bool { return len(digits(r.Phone)) == 11 }

func (w *WhatsApp) Send(ctx context.Context, r Recipient, m Message) error {
	endpoint := fmt.Sprintf("%s/waInstance%s/sendMessage/%s",
		strings.TrimRight(w.cfg.GreenAPIBase, "/"), w.cfg.GreenAPIInstanceID, w.cfg.GreenAPIToken)
	return postJSON(ctx, endpoint,
		map[string]any{"chatId": digits(r.Phone) + "@c.us", "message": text(m, w.cfg.link(m.Link))},
		func(body []byte) error {
			var resp struct {
				IDMessage string `json:"idMessage"`
			}
			if json.Unmarshal(body, &resp) != nil || resp.IDMessage == "" {
				return fmt.Errorf("green-api: unexpected response %s", truncate(body))
			}
			return nil
		})
}

// ---------- helpers ----------

func postJSON(ctx context.Context, endpoint string, payload any, check func([]byte) error) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return do(req, check)
}

func do(req *http.Request, check func([]byte) error) error {
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err := check(body); err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, truncate(body))
	}
	return nil
}

func truncate(b []byte) string {
	if len(b) > 200 {
		return string(b[:200]) + "…"
	}
	return string(b)
}

// SetWebhook registers the bot webhook (idempotent; called on startup).
func (t *Telegram) SetWebhook(ctx context.Context, webhookURL string) error {
	return postJSON(ctx, t.base+"/bot"+t.cfg.TelegramToken+"/setWebhook",
		map[string]any{
			"url":             webhookURL,
			"secret_token":    t.cfg.TelegramWebhookSecret,
			"allowed_updates": []string{"message"},
		},
		func(body []byte) error {
			var resp struct {
				OK          bool   `json:"ok"`
				Description string `json:"description"`
			}
			if json.Unmarshal(body, &resp) != nil || !resp.OK {
				return fmt.Errorf("telegram setWebhook: %s", resp.Description)
			}
			return nil
		})
}
