package service

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// Input errors answered with 400 (Postgres would otherwise fail with 500).
var (
	ErrInvalidText     = errors.New("Поля содержат недопустимые символы.")
	ErrNameTooLong     = errors.New("ФИО слишком длинное (не больше 255 символов).")
	ErrPasswordTooLong = errors.New("Пароль слишком длинный (не больше 72 латинских символов).")
)

// maxPasswordBytes is bcrypt's input limit; longer passwords fail to hash.
const maxPasswordBytes = 72

// cleanText reports whether every value is valid UTF-8 without NUL bytes
// (Postgres text columns reject both).
func cleanText(values ...string) bool {
	for _, v := range values {
		if !utf8.ValidString(v) || strings.ContainsRune(v, 0) {
			return false
		}
	}
	return true
}

// checkPassword enforces the length rules shared by registration and password change.
func checkPassword(pw string) error {
	if utf8.RuneCountInString(pw) < minPasswordLen {
		return ErrWeakPassword
	}
	if len(pw) > maxPasswordBytes {
		return ErrPasswordTooLong
	}
	return nil
}
