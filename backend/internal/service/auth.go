package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type LoginInput struct {
	Login    string
	Password string
}

type RegisterInput struct {
	FullName     string
	Phone        string
	IIN          string
	Password     string
	JKID         *string
	Verification models.Verification
}

// Registration errors with user-facing meaning.
var (
	ErrIINTaken      = errors.New("Этот ИИН уже зарегистрирован. Войдите с паролем.")
	ErrPhoneTaken    = errors.New("Этот номер телефона уже используется.")
	ErrWeakPassword  = errors.New("Пароль должен быть не короче 6 символов.")
	ErrJKRequired    = errors.New("Откройте ссылку или QR-код, который выдал ваш ОСИ.")
	ErrInvalidIIN    = errors.New("ИИН должен состоять из 12 цифр.")
	ErrMissingFields = errors.New("Заполните ФИО и номер телефона.")
)

const minPasswordLen = 6

func ValidIIN(iin string) bool {
	if len(iin) != 12 {
		return false
	}
	for _, ch := range iin {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func (s *Service) IINRegistered(ctx context.Context, iin string) (bool, error) {
	iin = strings.TrimSpace(iin)
	if !ValidIIN(iin) {
		return false, ErrInvalidIIN
	}
	return s.repo.IINRegistered(ctx, iin)
}

// Register creates a resident on first login. The JK comes from the OSI's
// QR link; when the system has exactly one JK it is used automatically.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*models.User, error) {
	fullName := strings.TrimSpace(in.FullName)
	phone := strings.TrimSpace(in.Phone)
	iin := strings.TrimSpace(in.IIN)
	password := in.Password

	if !ValidIIN(iin) {
		return nil, ErrInvalidIIN
	}
	if fullName == "" || phone == "" {
		return nil, ErrMissingFields
	}
	if len([]rune(password)) < minPasswordLen {
		return nil, ErrWeakPassword
	}
	verification, err := normalizeVerification(in.Verification)
	if err != nil {
		return nil, err
	}

	jkID, err := s.resolveRegistrationJK(ctx, in.JKID)
	if err != nil {
		return nil, err
	}

	if taken, err := s.repo.IINRegistered(ctx, iin); err != nil {
		return nil, err
	} else if taken {
		return nil, ErrIINTaken
	}
	if taken, err := s.repo.PhoneRegistered(ctx, phone); err != nil {
		return nil, err
	} else if taken {
		return nil, ErrPhoneTaken
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		FullName:     fullName,
		Phone:        phone,
		PasswordHash: hashedPassword,
		Role:         models.RoleResident,
		JKID:         &jkID,
		IIN:          iin,

		Status:         models.UserPending,
		OwnerType:      verification.OwnerType,
		PremisesKind:   verification.PremisesKind,
		PremisesNumber: verification.PremisesNumber,
		CompanyBIN:     verification.CompanyBIN,
		CompanyName:    verification.CompanyName,
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	s.notifyResidentNew(ctx, user)
	return user, nil
}

func (s *Service) resolveRegistrationJK(ctx context.Context, requested *string) (string, error) {
	if requested != nil && strings.TrimSpace(*requested) != "" {
		jk, err := s.repo.GetJKByID(ctx, strings.TrimSpace(*requested))
		if err != nil {
			return "", ErrJKRequired
		}
		return jk.ID, nil
	}
	jks, err := s.repo.ListJKs(ctx)
	if err != nil {
		return "", err
	}
	if len(jks) != 1 {
		return "", ErrJKRequired
	}
	return jks[0].ID, nil
}

func (s *Service) Login(ctx context.Context, in LoginInput) (*models.User, error) {
	login := strings.TrimSpace(in.Login)
	password := in.Password // not trimmed: registration keeps spaces too

	if login == "" || password == "" {
		return nil, ErrInvalid
	}

	user, err := s.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !CheckPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (s *Service) Me(ctx context.Context, userID string) (*models.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalid
	}

	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return user, nil
}

// ---------- Password Helpers ----------

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
