package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/legal"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/service"
)

func (h *Handler) ListPaymentsByJK(c *gin.Context) {
	jkID := c.Param("jk_id")
	if !middleware.SameJK(c, jkID) {
		middleware.Forbidden(c)
		return
	}

	// Optional ?from=&to= (YYYY-MM-DD, Kazakhstan days) for the Платежи page.
	var from, to *time.Time
	if c.Query("from") != "" || c.Query("to") != "" {
		f, t, ok := parsePeriod(c)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный период (формат ГГГГ-ММ-ДД)"})
			return
		}
		from, to = &f, &t
	}
	payments, err := h.svc.ListPaymentsByJKPeriod(c.Request.Context(), jkID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить список платежей"})
		return
	}
	if payments == nil {
		payments = []*models.PaymentResponse{}
	}
	c.JSON(http.StatusOK, payments)
}

func (h *Handler) ListMyPayments(c *gin.Context) {
	payments, err := h.svc.ListPaymentsByUser(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить список платежей"})
		return
	}
	c.JSON(http.StatusOK, payments)
}

// Amount is intentionally not accepted: the price comes from the rental in the DB.
type CreatePaymentRequest struct {
	RentalID string `json:"rental_id" binding:"required"`
	Phone    string `json:"phone"`
	// The resident ticks "I accept the agreement" next to the pay button (the contract).
	AcceptedTermsVersion string `json:"accepted_terms_version"`
}

func (h *Handler) CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Не указана аренда"})
		return
	}
	if termsRequired(c, req.AcceptedTermsVersion) {
		return
	}
	userID := middleware.UserID(c)

	// Development / stage (FAKE_PAYMENTS): pay without Kaspi, through the same
	// activation as a real paid webhook.
	if h.cfg.FakePayments || h.cfg.Env == "development" {
		payment, err := h.svc.PayWithoutKaspi(c.Request.Context(), req.RentalID, userID)
		switch {
		case err == nil:
			h.recordConsent(c, userID, legal.ResidentTerms, "payment", req.RentalID)
			c.JSON(http.StatusOK, gin.H{"message": "Тестовая оплата: аренда активирована", "amount": payment.Amount, "test_payment": true})
		case errors.Is(err, service.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Аренда не найдена"})
		case errors.Is(err, service.ErrPaymentPending):
			c.JSON(http.StatusConflict, gin.H{"error": "Оплата уже обрабатывается"})
		default:
			c.JSON(http.StatusConflict, gin.H{"error": "Срок брони истёк. Забронируйте кладовку заново."})
		}
		return
	}

	payment, err := h.svc.CreatePaymentInvoice(c.Request.Context(), service.CreatePaymentInput{
		RentalID: req.RentalID,
		UserID:   userID,
		Phone:    req.Phone,
	}, h.cfg.ApiPayKey, h.cfg.ApiPayBaseURL)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Аренда не найдена"})
		case errors.Is(err, service.ErrPaymentPending):
			c.JSON(http.StatusConflict, gin.H{"error": "Счёт уже отправлен в Kaspi. Подтвердите оплату в приложении Kaspi."})
		case errors.Is(err, service.ErrLocked):
			c.JSON(http.StatusConflict, gin.H{"error": "Срок брони истёк. Забронируйте кладовку заново."})
		case errors.Is(err, service.ErrInvalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите номер телефона, привязанный к Kaspi"})
		default:
			log.Printf("CreatePayment: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "Не удалось отправить счёт в Kaspi. Попробуйте ещё раз."})
		}
		return
	}

	h.recordConsent(c, userID, legal.ResidentTerms, "payment", req.RentalID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Счёт отправлен в Kaspi",
		"amount":  payment.Amount,
	})
}

type webhookEvent struct {
	Event     string `json:"event"`
	IsSandbox bool   `json:"is_sandbox"`
	Invoice   struct {
		ExternalOrderID *string `json:"external_order_id"`
		Status          string  `json:"status"`
	} `json:"invoice"`
}

// PaymentWebhook receives ApiPay events. Requests are authenticated with
// X-Webhook-Signature: sha256=<hex HMAC-SHA256(raw body, APIPAY_WEBHOOK_SECRET)>.
func (h *Handler) PaymentWebhook(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if !h.webhookSignatureOK(body, c.GetHeader("X-Webhook-Signature")) {
		log.Printf("payment webhook: invalid or missing signature")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	var ev webhookEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Only invoice status changes matter; ignore tests, sandbox events in live mode, unknown events.
	if ev.Event != "invoice.status_changed" || ev.Invoice.ExternalOrderID == nil ||
		(ev.IsSandbox && h.cfg.ApiPayLive) {
		c.Status(http.StatusOK)
		return
	}

	err = h.svc.HandlePaymentWebhook(c.Request.Context(), *ev.Invoice.ExternalOrderID, ev.Invoice.Status)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			// Not our payment — acknowledge so ApiPay stops retrying.
			c.Status(http.StatusOK)
			return
		}
		log.Printf("payment webhook: %v", err)
		// 5xx makes ApiPay retry later.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обработать платёж"})
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) webhookSignatureOK(body []byte, header string) bool {
	secret := h.cfg.ApiPayWebhookSecret
	if secret == "" {
		// Unsigned webhooks are accepted only on a developer machine.
		return h.cfg.Env == "development"
	}
	sig, ok := strings.CutPrefix(strings.TrimSpace(header), "sha256=")
	if !ok {
		return false
	}
	got, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(got, mac.Sum(nil))
}

type renewRentalRequest struct {
	Months               int    `json:"months"`
	Phone                string `json:"phone"`
	AcceptedTermsVersion string `json:"accepted_terms_version"`
}

// RenewRental sends a Kaspi invoice that extends the caller's active rental
// by `months` once paid (applied by the webhook, from the current end date).
func (h *Handler) RenewRental(c *gin.Context) {
	var req renewRentalRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Months < 1 || req.Months > service.MaxRentalMonths {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите срок продления"})
		return
	}
	if termsRequired(c, req.AcceptedTermsVersion) {
		return
	}
	in := service.RenewRentalInput{
		RentalID: c.Param("id"),
		UserID:   middleware.UserID(c),
		Months:   req.Months,
		Phone:    req.Phone,
	}

	var payment *models.Payment
	var err error
	if h.cfg.FakePayments || h.cfg.Env == "development" {
		// Development / stage: apply the renewal without ApiPay.
		payment, err = h.svc.PrepareRenewal(c.Request.Context(), in)
		if err == nil {
			err = h.svc.HandlePaymentWebhook(c.Request.Context(), payment.ID, "paid")
		}
	} else {
		payment, err = h.svc.CreateRenewalInvoice(c.Request.Context(), in, h.cfg.ApiPayKey, h.cfg.ApiPayBaseURL)
	}
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Аренда не найдена"})
		case errors.Is(err, service.ErrLocked):
			c.JSON(http.StatusConflict, gin.H{"error": "Продлить можно только оплаченную аренду"})
		case errors.Is(err, service.ErrNotApproved):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrTransferPending):
			c.JSON(http.StatusConflict, gin.H{"error": "Кладовка передаётся новому владельцу — продление недоступно"})
		case errors.Is(err, service.ErrInvalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите номер телефона, привязанный к Kaspi"})
		default:
			log.Printf("RenewRental: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "Не удалось отправить счёт в Kaspi. Попробуйте ещё раз."})
		}
		return
	}
	h.recordConsent(c, in.UserID, legal.ResidentTerms, "renewal", in.RentalID)
	c.JSON(http.StatusCreated, payment)
}
