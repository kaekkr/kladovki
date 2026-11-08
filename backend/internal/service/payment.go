package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/repository"
)

// PaymentWindow: once a Kaspi invoice is sent the booking is held at least this
// long, so a user who is confirming in Kaspi does not lose the storage.
const PaymentWindow = 10 * time.Minute

var apipayClient = &http.Client{Timeout: 15 * time.Second}

func (s *Service) ListPaymentsByJK(ctx context.Context, jkID string) ([]*models.PaymentResponse, error) {
	if jkID == "" {
		return nil, ErrInvalid
	}
	return s.repo.ListPaymentsByJK(ctx, jkID)
}

func (s *Service) ListPaymentsByUser(ctx context.Context, userID string) ([]*models.PaymentResponse, error) {
	if userID == "" {
		return nil, ErrInvalid
	}
	return s.repo.ListPaymentResponsesByUser(ctx, userID)
}

func (s *Service) ListRentalsByUser(ctx context.Context, userID string) ([]*models.Rental, error) {
	if userID == "" {
		return nil, ErrInvalid
	}
	return s.repo.ListRentalsByUser(ctx, userID)
}

type CreatePaymentInput struct {
	RentalID string
	UserID   string
	Phone    string // optional; defaults to the user's phone
}

// CreatePaymentInvoice records a pending payment (amount computed from the
// rental in the DB) and asks ApiPay to push a Kaspi invoice to the phone.
func (s *Service) CreatePaymentInvoice(ctx context.Context, in CreatePaymentInput, apipayKey, apipayBaseURL string) (*models.Payment, error) {
	if in.RentalID == "" || in.UserID == "" {
		return nil, ErrInvalid
	}

	phone := strings.TrimSpace(in.Phone)
	if phone == "" {
		user, err := s.repo.GetUserByID(ctx, in.UserID)
		if err != nil {
			return nil, fmt.Errorf("get user: %w", err)
		}
		phone = user.Phone
	}
	phone = formatPhoneNumber(phone)
	if len(phone) != 11 {
		return nil, ErrInvalid
	}

	payment, err := s.repo.PreparePayment(ctx, in.RentalID, in.UserID, PaymentWindow)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return nil, ErrNotFound
		case errors.Is(err, repository.ErrLocked):
			return nil, ErrLocked
		}
		return nil, err
	}

	if err := sendApiPayInvoice(ctx, apipayKey, apipayBaseURL, payment, phone); err != nil {
		_ = s.repo.MarkPaymentFailed(ctx, payment.ID)
		return nil, err
	}
	return payment, nil
}

func sendApiPayInvoice(ctx context.Context, key, baseURL string, p *models.Payment, phone string) error {
	body, err := json.Marshal(map[string]any{
		"amount":            p.Amount,
		"phone_number":      phone,
		"external_order_id": p.ID,
		"description":       "Oplata arendy kladovki", // ApiPay limit: 60 chars
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/v1/invoices", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", key)

	resp, err := apipayClient.Do(req)
	if err != nil {
		return fmt.Errorf("apipay request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("apipay returned status %d, body: %v", resp.StatusCode, errResp)
	}
	return nil
}

// HandlePaymentWebhook applies an ApiPay invoice status to our payment.
// Status values per ApiPay: pending, processing, paid, cancelled, expired, error, partially_refunded.
func (s *Service) HandlePaymentWebhook(ctx context.Context, paymentID, status string) error {
	if paymentID == "" {
		return ErrInvalid
	}

	switch status {
	case "paid":
		err := s.repo.ActivatePaidRental(ctx, paymentID)
		if errors.Is(err, repository.ErrPaidButStorageTaken) {
			// Money received but storage went to someone else — OSI must refund.
			log.Printf("payment %s: paid but storage no longer available; refund required", paymentID)
			return nil
		}
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	case "cancelled", "expired", "error":
		return s.repo.MarkPaymentFailed(ctx, paymentID)
	default:
		return nil // pending/processing/refund events: nothing to do yet
	}
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
