package service

import (
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/kaekkr/kladovki/internal/cadastral"
)

// ValidationError is bad operator input; handlers answer 400 with its Russian text.
type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }

func invalid(msg string) error { return &ValidationError{msg: msg} }

// IsValidationError reports whether err is (or wraps) a ValidationError.
func IsValidationError(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}

// Conflicts the operator can fix in the form (409).
var (
	ErrEmailTaken     = errors.New("Этот email уже используется.")
	ErrCadastralTaken = errors.New("ЖК с таким кадастровым номером уже есть.")
)

const (
	maxNameLen    = 255
	maxAddressLen = 500
	maxEmailLen   = 255
	maxPhoneLen   = 50
	// Operator-chosen bootstrap password; temporary ones are generated.
	minSuperadminPasswordLen = 10
	tempPasswordLen          = 10
)

// JKDetails is what the operator types for a ЖК.
type JKDetails struct {
	Name            string
	Address         string
	CadastralNumber string
}

// AccountInput is what the operator types for an ОСИ account.
type AccountInput struct {
	FullName string
	Email    string
	Phone    string
}

// normalizeJKDetails trims, checks lengths and normalizes the cadastral number.
func normalizeJKDetails(in JKDetails) (JKDetails, error) {
	out := JKDetails{
		Name:            strings.TrimSpace(in.Name),
		Address:         strings.TrimSpace(in.Address),
		CadastralNumber: cadastral.Normalize(in.CadastralNumber),
	}
	switch {
	case out.Name == "":
		return out, invalid("Укажите название ЖК.")
	case utf8.RuneCountInString(out.Name) > maxNameLen:
		return out, invalid("Название ЖК слишком длинное.")
	case out.Address == "":
		return out, invalid("Укажите адрес ЖК.")
	case utf8.RuneCountInString(out.Address) > maxAddressLen:
		return out, invalid("Адрес ЖК слишком длинный.")
	case out.CadastralNumber != "" && !cadastral.ValidComplex(out.CadastralNumber):
		return out, invalid("Кадастровый номер ЖК должен быть вида 21:319:021:848:8.")
	}
	return out, nil
}

// normalizeAccount trims the fields, lower-cases the email and validates them.
func normalizeAccount(in AccountInput) (AccountInput, error) {
	out := AccountInput{
		FullName: strings.TrimSpace(in.FullName),
		Email:    strings.ToLower(strings.TrimSpace(in.Email)),
		Phone:    strings.TrimSpace(in.Phone),
	}
	switch {
	case out.FullName == "":
		return out, invalid("Укажите ФИО администратора ОСИ.")
	case utf8.RuneCountInString(out.FullName) > maxNameLen:
		return out, invalid("ФИО слишком длинное.")
	case !validEmail(out.Email):
		return out, invalid("Укажите корректный email.")
	case out.Phone == "":
		return out, invalid("Укажите номер телефона.")
	case utf8.RuneCountInString(out.Phone) > maxPhoneLen:
		return out, invalid("Номер телефона слишком длинный.")
	}
	return out, nil
}

// validEmail accepts a bare address like name@example.kz (no display name).
func validEmail(s string) bool {
	if s == "" || len(s) > maxEmailLen {
		return false
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return false
	}
	at := strings.LastIndex(s, "@")
	return at > 0 && strings.Contains(s[at+1:], ".")
}
