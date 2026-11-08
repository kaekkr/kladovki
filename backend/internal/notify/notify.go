// Package notify delivers user notifications over external channels.
//
// Every channel is optional and switched on purely by environment variables,
// so a channel that is not configured is simply skipped:
//
//	Email     SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, SMTP_FROM
//	Telegram  TELEGRAM_BOT_TOKEN, TELEGRAM_BOT_USERNAME, TELEGRAM_WEBHOOK_SECRET
//	SMS       MOBIZON_API_KEY (mobizon.kz)
//	WhatsApp  GREENAPI_INSTANCE_ID, GREENAPI_TOKEN (green-api.com)
//
// In-app notifications are stored in the database and need no channel.
package notify

import (
	"context"
	"net/http"
	"os"
	"time"
)

// Recipient holds the contact points of one user.
type Recipient struct {
	Name           string
	Email          string
	Phone          string // as stored, e.g. +77001112233
	TelegramChatID string
}

// Message is what gets delivered. Body is plain text; Link is an app path
// (e.g. /app/rentals/123) turned into a full URL by the channel when useful.
type Message struct {
	Title string
	Body  string
	Link  string
}

// Channel is one delivery method.
type Channel interface {
	Name() string
	// CanReach reports whether this recipient has a contact for this channel.
	CanReach(r Recipient) bool
	Send(ctx context.Context, r Recipient, m Message) error
}

// Config collects all channel settings.
type Config struct {
	AppBaseURL string // public URL of the app, used for links

	SMTPHost, SMTPPort, SMTPUser, SMTPPassword, SMTPFrom string

	TelegramToken, TelegramBotUsername, TelegramWebhookSecret string
	TelegramAPIBase                                           string // override for tests

	MobizonAPIKey, MobizonAPIBase string

	GreenAPIInstanceID, GreenAPIToken, GreenAPIBase string
}

// ConfigFromEnv reads channel settings from the environment.
func ConfigFromEnv() Config {
	return Config{
		AppBaseURL:            os.Getenv("APP_BASE_URL"),
		SMTPHost:              os.Getenv("SMTP_HOST"),
		SMTPPort:              envOr("SMTP_PORT", "587"),
		SMTPUser:              os.Getenv("SMTP_USER"),
		SMTPPassword:          os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:              os.Getenv("SMTP_FROM"),
		TelegramToken:         os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramBotUsername:   os.Getenv("TELEGRAM_BOT_USERNAME"),
		TelegramWebhookSecret: os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		MobizonAPIKey:         os.Getenv("MOBIZON_API_KEY"),
		MobizonAPIBase:        envOr("MOBIZON_API_BASE", "https://api.mobizon.kz"),
		GreenAPIInstanceID:    os.Getenv("GREENAPI_INSTANCE_ID"),
		GreenAPIToken:         os.Getenv("GREENAPI_TOKEN"),
		GreenAPIBase:          envOr("GREENAPI_BASE", "https://api.green-api.com"),
	}
}

// Channels builds every configured channel.
func (c Config) Channels() []Channel {
	var out []Channel
	if c.SMTPHost != "" && c.SMTPFrom != "" {
		out = append(out, &Email{cfg: c})
	}
	if c.TelegramToken != "" {
		out = append(out, NewTelegram(c))
	}
	if c.MobizonAPIKey != "" {
		out = append(out, &SMS{cfg: c})
	}
	if c.GreenAPIInstanceID != "" && c.GreenAPIToken != "" {
		out = append(out, &WhatsApp{cfg: c})
	}
	return out
}

func (c Config) link(path string) string {
	if path == "" || c.AppBaseURL == "" {
		return ""
	}
	return c.AppBaseURL + path
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// digits keeps only 0-9 and normalises a KZ number to 7XXXXXXXXXX.
func digits(phone string) string {
	out := make([]byte, 0, len(phone))
	for i := 0; i < len(phone); i++ {
		if phone[i] >= '0' && phone[i] <= '9' {
			out = append(out, phone[i])
		}
	}
	s := string(out)
	if len(s) == 11 && s[0] == '8' {
		s = "7" + s[1:]
	}
	return s
}

func text(m Message, link string) string {
	s := m.Title + "\n\n" + m.Body
	if link != "" {
		s += "\n\n" + link
	}
	return s
}
