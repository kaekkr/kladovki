package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims — payload JWT
type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	JKID   string `json:"jk_id,omitempty"`
	Email  string `json:"email,omitempty"`
	jwt.RegisteredClaims
}

type TokenService struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewTokenService(secret string, ttl time.Duration) *TokenService {
	return &TokenService{
		secret: []byte(secret),
		ttl:    ttl,
		issuer: "kladovki",
	}
}

func (ts *TokenService) Generate(userID, role, jkID, email string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(ts.ttl)

	claims := Claims{
		UserID: userID,
		Role:   role,
		JKID:   jkID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			// Отнимаем 5 секунд buffer-time на случай минимальной рассинхронизации часов
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			Issuer:    ts.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(ts.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}

	return signed, expiresAt, nil
}

func (ts *TokenService) Parse(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return ts.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Проверяем наличие ключевых полей
	if claims.UserID == "" || claims.Role == "" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
