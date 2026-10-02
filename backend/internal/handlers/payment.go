package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/service"
)

func (h *Handler) ListPaymentsByJK(c *gin.Context) {
	jkID := c.Param("jk_id")
	if jkID == "" {
		jkID = middleware.JKID(c)
	}

	if jkID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Доступ запрещен: ЖК не определен"})
		return
	}

	payments, err := h.svc.ListPaymentsByJK(c.Request.Context(), jkID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить список платежей"})
		return
	}

	c.JSON(http.StatusOK, payments)
}

type CreatePaymentRequest struct {
	RentalID string `json:"rental_id" binding:"required"`
	Amount   int64  `json:"amount" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
}

func (h *Handler) CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.UserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Не авторизован"})
		return
	}

	// Метод теперь возвращает только error (так как ссылка не нужна, идет push в Kaspi)
	err := h.svc.CreatePaymentInvoice(c.Request.Context(), service.CreatePaymentInput{
		RentalID: req.RentalID,
		UserID:   userID,
		Amount:   req.Amount,
		Phone:    req.Phone,
	}, h.cfg.ApiPayKey, h.cfg.ApiPayBaseURL)
	if err != nil {
		fmt.Printf("ERROR CREATING PAYMENT: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать платеж: " + err.Error()})
		return
	}

	// Возвращаем статус успеха, фронтенд теперь запускает поллинг (ожидание статуса)
	c.JSON(http.StatusOK, gin.H{
		"message": "Счёт успешно создан и отправлен в Kaspi",
	})
}

type InvoicePayload struct {
	ExternalOrderID *string              `json:"external_order_id"`
	Status          models.PaymentStatus `json:"status"`
}

type PaymentWebhookRequest struct {
	Event   string         `json:"event" binding:"required"`
	Invoice InvoicePayload `json:"invoice" binding:"required"`
}

func (h *Handler) PaymentWebhook(c *gin.Context) {
	var req PaymentWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("WEBHOOK BIND ERROR: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Если это тестовый вебхук или external_order_id пустой, просто возвращаем 200 OK
	if req.Event == "webhook.test" || req.Invoice.ExternalOrderID == nil {
		c.Status(http.StatusOK)
		return
	}

	// Обрабатываем реальное изменение статуса счета
	if err := h.svc.HandlePaymentWebhook(c.Request.Context(), *req.Invoice.ExternalOrderID, req.Invoice.Status); err != nil {
		fmt.Printf("WEBHOOK SERVICE ERROR: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить статус платежа"})
		return
	}

	c.Status(http.StatusOK)
}
