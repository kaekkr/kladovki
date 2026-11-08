package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/notify"
)

// Notification kinds.
const (
	KindPaymentSuccess  = "payment_success"
	KindRentalExpiring  = "rental_expiring"
	KindRentalOverdue   = "rental_overdue"
	KindDebtWarning     = "debt_warning"
	KindTariffChanged   = "tariff_changed"
	KindTariffRecalc    = "tariff_recalc"
	KindRequestNew      = "request_new"
	KindRequestResolved = "request_resolved"
)

// Paid channels (SMS, WhatsApp) are used only for messages that matter.
var importantKinds = map[string]bool{
	KindRentalExpiring: true,
	KindRentalOverdue:  true,
	KindDebtWarning:    true,
	KindTariffRecalc:   true,
}

var paidChannels = map[string]bool{"sms": true, "whatsapp": true}

const maxDeliveryAttempts = 6

// Kazakhstan uses UTC+5 nationwide since 2024; fixed zone avoids tzdata dependency.
var kzTime = time.FixedZone("UTC+5", 5*60*60)

func kzDate(t time.Time) string { return t.In(kzTime).Format("02.01.2006") }

func money(v int64) string {
	s := fmt.Sprintf("%d", v)
	out := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += " "
		}
		out += string(c)
	}
	return out + " ₸"
}

// SetNotifier configures external channels (see package notify).
func (s *Service) SetNotifier(cfg notify.Config) {
	s.notifyCfg = cfg
	s.channels = map[string]notify.Channel{}
	for _, ch := range cfg.Channels() {
		s.channels[ch.Name()] = ch
	}
}

func (s *Service) NotifyConfig() notify.Config { return s.notifyCfg }

// Notify stores an in-app notification and queues delivery on every channel
// the user can be reached on. dedupeKey makes it idempotent ("" = always new).
func (s *Service) Notify(ctx context.Context, userID, kind string, m notify.Message, dedupeKey string) {
	contacts, err := s.repo.UserContacts(ctx, userID)
	if err != nil {
		log.Printf("notify %s: %v", userID, err)
		return
	}
	var channels []string
	for name, ch := range s.channels {
		if paidChannels[name] && !importantKinds[kind] {
			continue
		}
		if ch.CanReach(contacts) {
			channels = append(channels, name)
		}
	}
	if _, err := s.repo.CreateNotification(ctx, userID, kind, m.Title, m.Body, m.Link, dedupeKey, channels); err != nil {
		log.Printf("notify %s (%s): %v", userID, kind, err)
	}
}

func (s *Service) notifyJKAdmins(ctx context.Context, jkID, kind string, m notify.Message, dedupeKey string) {
	admins, err := s.repo.ListJKAdminIDs(ctx, jkID)
	if err != nil {
		log.Printf("notify admins %s: %v", jkID, err)
		return
	}
	for _, id := range admins {
		s.Notify(ctx, id, kind, m, dedupeKey)
	}
}

// DispatchPending sends queued deliveries once. Failures are retried with back-off.
func (s *Service) DispatchPending(ctx context.Context) error {
	items, err := s.repo.PendingDeliveries(ctx, 100)
	if err != nil {
		return err
	}
	for _, d := range items {
		ch, ok := s.channels[d.Channel]
		if !ok || !ch.CanReach(d.Recipient) {
			_ = s.repo.FinishDelivery(ctx, d.ID, nil, true, maxDeliveryAttempts)
			continue
		}
		sendErr := ch.Send(ctx, d.Recipient, d.Message)
		if sendErr != nil {
			log.Printf("delivery %s via %s failed: %v", d.ID, d.Channel, sendErr)
		}
		if err := s.repo.FinishDelivery(ctx, d.ID, sendErr, false, maxDeliveryAttempts); err != nil {
			return err
		}
	}
	return nil
}

// RunReminders creates time-based notifications. Safe to run any number of
// times — every reminder has a dedupe key.
//
//   - 7 days and 1 day before the paid period ends (tenant)
//   - overdue: right away, then weekly (tenant + OSI admins)
//   - more than 2 months overdue: warning about possible termination (agreement 5.2)
func (s *Service) RunReminders(ctx context.Context) error {
	now := time.Now()
	rentals, err := s.repo.RentalsForReminders(ctx, now.Add(7*24*time.Hour))
	if err != nil {
		return err
	}
	for _, r := range rentals {
		link := "/app/rentals/" + r.RentalID
		end := kzDate(r.EndsAt)
		base := r.RentalID + ":" + r.EndsAt.UTC().Format("20060102")

		if r.EndsAt.After(now) {
			key, when := "exp7:"+base, "через "+daysWord(int(math.Ceil(r.EndsAt.Sub(now).Hours()/24)))
			if r.EndsAt.Sub(now) <= 24*time.Hour {
				key, when = "exp1:"+base, "завтра"
			}
			s.Notify(ctx, r.UserID, KindRentalExpiring, notify.Message{
				Title: "Срок аренды истекает " + when,
				Body:  fmt.Sprintf("Аренда кладовки №%s оплачена до %s. Продлите аренду, чтобы кладовка осталась за вами.", r.StorageNumber, end),
				Link:  link,
			}, key)
			continue
		}

		days := int(now.Sub(r.EndsAt).Hours() / 24)
		week := days / 7
		s.Notify(ctx, r.UserID, KindRentalOverdue, notify.Message{
			Title: "Аренда просрочена",
			Body: fmt.Sprintf("Оплаченный период кладовки №%s закончился %s. Пожалуйста, оплатите аренду или свяжитесь с ОСИ.",
				r.StorageNumber, end),
			Link: link,
		}, fmt.Sprintf("overdue:%s:%d", base, week))

		severity := "Просрочка"
		if days > 14 {
			severity = "Просрочка более 2 недель"
		}
		s.notifyJKAdmins(ctx, r.JKID, KindRentalOverdue, notify.Message{
			Title: fmt.Sprintf("%s: кладовка №%s", severity, r.StorageNumber),
			Body:  fmt.Sprintf("Аренда закончилась %s (%s назад). Свяжитесь с арендатором.", end, daysWord(days)),
			Link:  "/admin/rentals",
		}, fmt.Sprintf("overdue:%s:%d", base, week))

		if days > 60 {
			s.Notify(ctx, r.UserID, KindDebtWarning, notify.Message{
				Title: "Задолженность более 2 месяцев",
				Body: fmt.Sprintf("По кладовке №%s задолженность более 2 месяцев. Согласно соглашению ОСИ вправе расторгнуть договор и освободить кладовку.",
					r.StorageNumber),
				Link: link,
			}, "debt60:"+base)
		}
	}
	return nil
}

func daysWord(n int) string {
	if n < 0 {
		n = -n
	}
	w := "дней"
	switch {
	case n%10 == 1 && n%100 != 11:
		w = "день"
	case n%10 >= 2 && n%10 <= 4 && (n%100 < 12 || n%100 > 14):
		w = "дня"
	}
	return fmt.Sprintf("%d %s", n, w)
}

// StartNotificationWorker sends queued messages every 30 s and runs reminders hourly.
func (s *Service) StartNotificationWorker(ctx context.Context) {
	send := time.NewTicker(30 * time.Second)
	remind := time.NewTicker(time.Hour)
	defer send.Stop()
	defer remind.Stop()

	if err := s.RunReminders(ctx); err != nil {
		log.Printf("reminders: %v", err)
	}
	for {
		select {
		case <-send.C:
			if err := s.DispatchPending(ctx); err != nil {
				log.Printf("dispatch: %v", err)
			}
		case <-remind.C:
			if err := s.RunReminders(ctx); err != nil {
				log.Printf("reminders: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// ---------- event notifications ----------

func (s *Service) notifyPaymentSuccess(ctx context.Context, paymentID string) {
	rentalID, err := s.repo.RentalIDForPayment(ctx, paymentID)
	if err != nil {
		return
	}
	r, err := s.repo.RentalForNotice(ctx, rentalID)
	if err != nil {
		return
	}
	s.Notify(ctx, r.UserID, KindPaymentSuccess, notify.Message{
		Title: "Оплата получена",
		Body:  fmt.Sprintf("Аренда кладовки №%s оплачена до %s.", r.StorageNumber, kzDate(r.EndsAt)),
		Link:  "/app/rentals/" + r.RentalID,
	}, "paid:"+paymentID)
}

func (s *Service) notifyRefundNeeded(ctx context.Context, paymentID string) {
	rentalID, err := s.repo.RentalIDForPayment(ctx, paymentID)
	if err != nil {
		return
	}
	r, err := s.repo.RentalForNotice(ctx, rentalID)
	if err != nil {
		return
	}
	s.notifyJKAdmins(ctx, r.JKID, KindRequestNew, notify.Message{
		Title: "Нужен возврат оплаты",
		Body:  fmt.Sprintf("Оплата за кладовку №%s поступила после окончания брони, кладовка уже занята. Верните деньги арендатору.", r.StorageNumber),
		Link:  "/admin/requests",
	}, "refund:"+paymentID)
}

func (s *Service) notifyTariffChange(ctx context.Context, jkID string, oldTariff, newTariff int64) {
	if newTariff < oldTariff {
		reqs, err := s.repo.AwaitingChoiceRequests(ctx, jkID)
		if err != nil {
			return
		}
		for _, q := range reqs {
			s.Notify(ctx, q.UserID, KindTariffRecalc, notify.Message{
				Title: "Тариф снижен — выберите перерасчёт",
				Body: fmt.Sprintf("Цена кладовки №%s снизилась до %s в месяц. За неиспользованный период вам положено %s: продлите аренду или запросите возврат.",
					q.StorageNumber, money(q.NewPrice), money(q.Amount)),
				Link: "/app/rentals/" + q.RentalID,
			}, "req:"+q.ID)
		}
		return
	}
	if newTariff == oldTariff {
		return
	}
	rentals, err := s.repo.ActiveRentalsOfJK(ctx, jkID)
	if err != nil {
		return
	}
	for _, r := range rentals {
		s.Notify(ctx, r.UserID, KindTariffChanged, notify.Message{
			Title: "Тариф изменён",
			Body: fmt.Sprintf("ОСИ повысило тариф. Для кладовки №%s прежняя цена сохраняется до конца оплаченного периода (%s), далее — по новому тарифу.",
				r.StorageNumber, kzDate(r.EndsAt)),
			Link: "/app/rentals/" + r.RentalID,
		}, "")
	}
}

func (s *Service) notifyNewRequest(ctx context.Context, q *models.Request) {
	s.notifyJKAdmins(ctx, q.JKID, KindRequestNew, notify.Message{
		Title: "Новая заявка: досрочное завершение",
		Body:  fmt.Sprintf("%s просит досрочно завершить аренду кладовки №%s. К возврату: %s.", q.UserName, q.StorageNumber, money(q.Amount)),
		Link:  "/admin/requests",
	}, "req:"+q.ID)
}

func (s *Service) notifyRequestResolved(ctx context.Context, requestID string) {
	q, err := s.repo.GetRequestByID(ctx, requestID)
	if err != nil {
		return
	}
	verdict := "одобрена"
	if q.Status == models.RequestRejected {
		verdict = "отклонена"
	}
	body := fmt.Sprintf("Ваша заявка по кладовке №%s %s.", q.StorageNumber, verdict)
	if q.Resolution != "" {
		body += " Комментарий ОСИ: " + q.Resolution
	}
	s.Notify(ctx, q.UserID, KindRequestResolved, notify.Message{
		Title: "Заявка " + verdict,
		Body:  body,
		Link:  "/app/rentals/" + q.RentalID,
	}, "resolved:"+q.ID)
}

// ---------- API helpers ----------

func (s *Service) ListNotifications(ctx context.Context, userID string) ([]*models.Notification, error) {
	return s.repo.ListNotifications(ctx, userID, 100)
}

func (s *Service) UnreadNotifications(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadNotifications(ctx, userID)
}

func (s *Service) MarkNotificationsRead(ctx context.Context, userID, id string) error {
	return s.repo.MarkNotificationsRead(ctx, userID, id)
}

// Channels reports which channels can reach the user and which are available on the server.
type ChannelStatus struct {
	Email             bool   `json:"email"`
	Telegram          bool   `json:"telegram"`
	SMS               bool   `json:"sms"`
	WhatsApp          bool   `json:"whatsapp"`
	TelegramAvailable bool   `json:"telegram_available"`
	EmailAddress      string `json:"email_address"`
}

func (s *Service) UserChannels(ctx context.Context, userID string) (ChannelStatus, error) {
	rc, err := s.repo.UserContacts(ctx, userID)
	if err != nil {
		return ChannelStatus{}, err
	}
	reach := func(name string) bool {
		ch, ok := s.channels[name]
		return ok && ch.CanReach(rc)
	}
	return ChannelStatus{
		Email:             reach("email"),
		Telegram:          rc.TelegramChatID != "",
		SMS:               reach("sms"),
		WhatsApp:          reach("whatsapp"),
		TelegramAvailable: s.notifyCfg.TelegramToken != "" && s.notifyCfg.TelegramBotUsername != "",
		EmailAddress:      rc.Email,
	}, nil
}

// TelegramLinkURL creates a one-time deep link that connects the user's Telegram.
func (s *Service) TelegramLinkURL(ctx context.Context, userID string) (string, error) {
	if s.notifyCfg.TelegramBotUsername == "" || s.notifyCfg.TelegramToken == "" {
		return "", ErrInvalid
	}
	token, err := s.repo.CreateTelegramLinkToken(ctx, userID)
	if err != nil {
		return "", err
	}
	return s.notifyCfg.TelegramLinkURL(token), nil
}

// HandleTelegramStart links a chat after the user pressed Start in the bot.
func (s *Service) HandleTelegramStart(ctx context.Context, chatID, token string) {
	tg, ok := s.channels["telegram"].(*notify.Telegram)
	if !ok {
		return
	}
	if token == "" {
		_ = tg.SendText(ctx, chatID, "Здравствуйте! Чтобы получать уведомления, откройте приложение «Кладовки» → Профиль → «Подключить Telegram».")
		return
	}
	if _, err := s.repo.LinkTelegram(ctx, token, chatID); err != nil {
		_ = tg.SendText(ctx, chatID, "Ссылка устарела. Откройте приложение и нажмите «Подключить Telegram» ещё раз.")
		return
	}
	_ = tg.SendText(ctx, chatID, "Готово! Уведомления о вашей аренде будут приходить сюда.")
}

func (s *Service) UnlinkTelegram(ctx context.Context, userID string) error {
	return s.repo.UnlinkTelegram(ctx, userID)
}
