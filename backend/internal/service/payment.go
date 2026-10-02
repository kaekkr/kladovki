package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/kaekkr/kladovki/internal/models"
)

func (s *Service) ListPaymentsByJK(ctx context.Context, jkID string) ([]*models.PaymentResponse, error) {
	if jkID == "" {
		return nil, ErrInvalid
	}
	return s.repo.ListPaymentsByJK(ctx, jkID)
}

type CreatePaymentInput struct {
	RentalID string
	UserID   string
	Amount   int64
	Phone    string
}

// Создание платежа: сохранение в базу (pending) + запрос в ApiPay на отправку push в Kaspi
func (s *Service) CreatePaymentInvoice(ctx context.Context, input CreatePaymentInput, apipayKey, apipayBaseURL string) error {
	if input.RentalID == "" || input.UserID == "" || input.Amount <= 0 || input.Phone == "" {
		return ErrInvalid
	}

	payment := &models.Payment{
		RentalID: input.RentalID,
		UserID:   input.UserID,
		Amount:   input.Amount,
		Provider: "apipay",
		Status:   models.PaymentStatusPending,
	}

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return fmt.Errorf("create payment in db: %w", err)
	}

	cleanedPhone := formatPhoneNumber(input.Phone)

	// Описание не должно превышать 60 символов по требованиям ApiPay
	description := "Oplata arendy"
	if len(description) > 60 {
		description = description[:60]
	}

	payload := map[string]any{
		"amount":            input.Amount,
		"phone_number":      cleanedPhone,
		"external_order_id": payment.ID,
		"description":       description,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Исправлен путь на /api/v1/invoices согласно документации ApiPay
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apipayBaseURL+"/v1/invoices", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", apipayKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("apipay request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("apipay returned status %d, body: %v", resp.StatusCode, errResp)
	}

	// ApiPay возвращает данные созданного инвойса (id, status и т.д.), payment_url больше нет
	var apiResp struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return err
	}

	return nil
}

// Обработка вебхука от ApiPay
func (s *Service) HandlePaymentWebhook(ctx context.Context, externalOrderID string, status models.PaymentStatus) error {
	if externalOrderID == "" || status == "" {
		return ErrInvalid
	}

	dbStatus := status
	switch status {
	case "paid", "completed":
		dbStatus = models.PaymentStatusSuccess // "success"
	case "fail", "rejected":
		dbStatus = models.PaymentStatusFailed // "failed"
	}

	return s.repo.CompletePaymentAndActivateRental(ctx, externalOrderID, dbStatus)
}

func formatPhoneNumber(phone string) string {
	digits := ""
	for _, ch := range phone {
		if ch >= '0' && ch <= '9' {
			digits += string(ch)
		}
	}

	if strings.HasPrefix(digits, "7") && len(digits) == 11 {
		digits = "8" + digits[1:]
	}

	return digits
}
