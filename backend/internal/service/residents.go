package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/notify"
	"github.com/kaekkr/kladovki/internal/repository"
)

// Only owners of a flat or commercial premises in the ЖК may rent (user
// agreement п. 2.1). Until the egov.kz check exists, the ОСИ confirms it.
var (
	ErrNotApproved      = errors.New("Аккаунт ожидает подтверждения ОСИ. Бронирование станет доступно после проверки.")
	ErrPremisesRequired = errors.New("Укажите номер квартиры или помещения.")
	ErrPremisesKind     = errors.New("Укажите тип помещения: квартира или коммерческое помещение.")
	ErrInvalidBIN       = errors.New("БИН компании должен состоять из 12 цифр.")
	ErrCompanyName      = errors.New("Укажите название компании.")
	ErrReasonRequired   = errors.New("Укажите причину отказа.")
	ErrAlreadyReviewed  = errors.New("Аккаунт уже подтверждён.")
)

// normalizeVerification validates what a user declares about their property.
func normalizeVerification(v models.Verification) (models.Verification, error) {
	v.OwnerType = strings.TrimSpace(v.OwnerType)
	v.PremisesKind = strings.TrimSpace(v.PremisesKind)
	v.PremisesNumber = strings.TrimSpace(v.PremisesNumber)
	v.CompanyBIN = strings.TrimSpace(v.CompanyBIN)
	v.CompanyName = strings.TrimSpace(v.CompanyName)

	if v.OwnerType == "" {
		v.OwnerType = models.OwnerPerson
	}
	if v.PremisesKind == "" {
		v.PremisesKind = models.PremisesFlat
	}
	if v.PremisesKind != models.PremisesFlat && v.PremisesKind != models.PremisesCommercial {
		return v, ErrPremisesKind
	}
	if v.PremisesNumber == "" || len([]rune(v.PremisesNumber)) > 50 {
		return v, ErrPremisesRequired
	}
	switch v.OwnerType {
	case models.OwnerPerson:
		v.CompanyBIN, v.CompanyName = "", ""
	case models.OwnerCompany:
		if !ValidIIN(v.CompanyBIN) { // БИН has the same 12-digit format as ИИН
			return v, ErrInvalidBIN
		}
		if v.CompanyName == "" || len([]rune(v.CompanyName)) > 255 {
			return v, ErrCompanyName
		}
	default:
		return v, ErrMissingFields
	}
	return v, nil
}

// requireApproved blocks booking and renewal for users the ОСИ has not confirmed.
func (s *Service) requireApproved(ctx context.Context, userID string) error {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.Role == models.RoleResident && u.Status != models.UserActive {
		return ErrNotApproved
	}
	return nil
}

func (s *Service) ListResidents(ctx context.Context, jkID, status string) ([]*models.User, error) {
	switch models.UserStatus(status) {
	case "", models.UserPending, models.UserActive, models.UserRejected, models.UserBlocked:
	default:
		return nil, ErrInvalid
	}
	return s.repo.ListResidents(ctx, jkID, status)
}

func (s *Service) CountPendingResidents(ctx context.Context, jkID string) (int64, error) {
	return s.repo.CountPendingResidents(ctx, jkID)
}

// ReviewResident applies an ОСИ decision: approve (also unblocks), reject (reason required) or block.
func (s *Service) ReviewResident(ctx context.Context, jkID, adminID, userID, action, comment string) error {
	comment = strings.TrimSpace(comment)
	var status models.UserStatus
	var msg notify.Message
	switch action {
	case "approve":
		status, comment = models.UserActive, ""
		msg = notify.Message{
			Title: "Доступ подтверждён",
			Body:  "ОСИ подтвердил ваш аккаунт. Теперь вы можете бронировать и арендовать кладовки.",
			Link:  "/app/storages",
		}
	case "reject":
		if comment == "" {
			return ErrReasonRequired
		}
		status = models.UserRejected
		msg = notify.Message{
			Title: "Заявка на доступ отклонена",
			Body:  "ОСИ отклонил заявку: " + comment + ". Исправьте данные в профиле и отправьте заново.",
			Link:  "/app/profile",
		}
	case "block":
		status = models.UserBlocked
		body := "ОСИ ограничил доступ к бронированию кладовок."
		if comment != "" {
			body += " Причина: " + comment + "."
		}
		msg = notify.Message{Title: "Доступ ограничен", Body: body, Link: "/app/profile"}
	default:
		return ErrInvalid
	}

	if err := s.repo.SetResidentStatus(ctx, userID, jkID, status, comment, adminID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	s.Notify(ctx, userID, KindAccountStatus, msg, fmt.Sprintf("account:%s:%s:%d", userID, status, time.Now().UnixNano()))
	return nil
}

// ResubmitVerification lets a pending or rejected user correct their data and ask again.
func (s *Service) ResubmitVerification(ctx context.Context, userID string, v models.Verification) error {
	v, err := normalizeVerification(v)
	if err != nil {
		return err
	}
	if err := s.repo.ResubmitVerification(ctx, userID, v); err != nil {
		if errors.Is(err, repository.ErrLocked) {
			return ErrAlreadyReviewed
		}
		return err
	}
	if u, err := s.repo.GetUserByID(ctx, userID); err == nil {
		s.notifyResidentNew(ctx, u)
	}
	return nil
}

// notifyResidentNew tells the ЖК's ОСИ that someone is waiting for confirmation.
func (s *Service) notifyResidentNew(ctx context.Context, u *models.User) {
	if u.JKID == nil {
		return
	}
	who := u.FullName
	if u.OwnerType == models.OwnerCompany {
		who = fmt.Sprintf("%s (представитель %s, БИН %s)", u.FullName, u.CompanyName, u.CompanyBIN)
	}
	place := "кв. " + u.PremisesNumber
	if u.PremisesKind == models.PremisesCommercial {
		place = "помещение " + u.PremisesNumber
	}
	s.notifyJKAdmins(ctx, *u.JKID, KindResidentNew, notify.Message{
		Title: "Новый житель ждёт подтверждения",
		Body:  fmt.Sprintf("%s, %s. Проверьте, что это собственник, и подтвердите доступ.", who, place),
		Link:  "/admin/residents",
	}, fmt.Sprintf("resident:%s:%d", u.ID, time.Now().UnixNano()))
}
