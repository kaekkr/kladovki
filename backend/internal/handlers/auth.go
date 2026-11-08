package handlers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/service"
)

type loginRequest struct {
	Login    string `json:"login"` // IIN, phone or email
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password" binding:"required"`
}

type registerRequest struct {
	FullName string  `json:"full_name" binding:"required"`
	Phone    string  `json:"phone" binding:"required"`
	Password string  `json:"password" binding:"required"`
	JKID     *string `json:"jk_id"`
	IIN      string  `json:"iin" binding:"required"`
	models.Verification
}

func (h *Handler) RegisterResident(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Заполните обязательные поля для регистрации"})
		return
	}

	user, err := h.svc.Register(c.Request.Context(), service.RegisterInput{
		FullName: req.FullName,
		Phone:    req.Phone,
		Password: req.Password,
		JKID:     req.JKID,
		IIN:      req.IIN,

		Verification: req.Verification,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrIINTaken), errors.Is(err, service.ErrPhoneTaken):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrInvalidIIN), errors.Is(err, service.ErrWeakPassword),
			errors.Is(err, service.ErrJKRequired), errors.Is(err, service.ErrMissingFields),
			isVerificationError(err):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			log.Printf("RegisterResident: %v", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Сервер временно недоступен. Попробуйте позже."})
		}
		return
	}

	// Сразу авторизуем пользователя (выпускаем токен и устанавливаем куку)
	_, err = h.issueToken(c, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сгенерировать токен"})
		return
	}

	c.JSON(http.StatusOK, userJSON(user))
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Заполните обязательные поля"})
		return
	}

	loginIdentifier := req.Login
	if loginIdentifier == "" {
		loginIdentifier = req.Email
	}
	if loginIdentifier == "" {
		loginIdentifier = req.Phone
	}

	user, err := h.svc.Login(c.Request.Context(), service.LoginInput{
		Login:    loginIdentifier,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль"})
			return
		}
		log.Printf("Login: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Сервер временно недоступен. Попробуйте позже."})
		return
	}

	// issueToken internally invokes h.setAuthCookie using h.cfg properties
	_, err = h.issueToken(c, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сгенерировать токен"})
		return
	}

	c.JSON(http.StatusOK, userJSON(user))
}

func (h *Handler) Me(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Пользователь не авторизован",
		})
		return
	}

	user, err := h.svc.Me(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Пользователь не найден",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось получить профиль пользователя",
		})
		return
	}

	c.JSON(http.StatusOK, userJSON(user))
}

func (h *Handler) Logout(c *gin.Context) {
	h.clearAuthCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "Успешный выход"})
}

// ---------- Cookie & Token Helpers ----------

func (h *Handler) setAuthCookie(c *gin.Context, token string, expires time.Time) {
	maxAge := max(int(time.Until(expires).Seconds()), 0)

	// Standard Lax mode works perfectly for same-origin requests
	c.SetSameSite(http.SameSiteLaxMode)

	c.SetCookie(
		h.cfg.CookieName,
		token,
		maxAge,
		"/",
		"",                 // Empty string defaults to host
		h.cfg.CookieSecure, // true on HTTPS in prod
		true,               // HttpOnly = true
	)
}

func (h *Handler) clearAuthCookie(c *gin.Context) {
	if h.cfg.Env == "production" {
		c.SetSameSite(http.SameSiteNoneMode)
	} else {
		c.SetSameSite(http.SameSiteLaxMode)
	}

	c.SetCookie(
		h.cfg.CookieName,
		"",
		-1,
		"/",
		"",
		h.cfg.CookieSecure,
		true,
	)
}

func (h *Handler) issueToken(c *gin.Context, u *models.User) (string, error) {
	jkID := ""
	if u.JKID != nil {
		jkID = *u.JKID
	}

	// Pass u.Role directly as a string
	token, exp, err := h.tokens.Generate(u.ID, string(u.Role), jkID, u.Email)
	if err != nil {
		return "", err
	}
	h.setAuthCookie(c, token, exp)
	return token, nil
}

type iinStatusRequest struct {
	IIN string `json:"iin"`
}

// IINStatus tells the login screen whether to ask for the password once
// (registered) or show first-login registration (new).
func (h *Handler) IINStatus(c *gin.Context) {
	var req iinStatusRequest
	_ = c.ShouldBindJSON(&req)
	registered, err := h.svc.IINRegistered(c.Request.Context(), req.IIN)
	if err != nil {
		if errors.Is(err, service.ErrInvalidIIN) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Сервер временно недоступен. Попробуйте позже."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"registered": registered})
}

// userJSON is the profile returned by register, login and /auth/me.
func userJSON(user *models.User) gin.H {
	return gin.H{
		"id":              user.ID,
		"full_name":       user.FullName,
		"phone":           user.Phone,
		"email":           user.Email,
		"role":            user.Role,
		"jk_id":           user.JKID,
		"iin":             user.IIN,
		"status":          user.Status,
		"review_comment":  user.ReviewComment,
		"owner_type":      user.OwnerType,
		"premises_kind":   user.PremisesKind,
		"premises_number": user.PremisesNumber,
		"company_bin":     user.CompanyBIN,
		"company_name":    user.CompanyName,
	}
}

func isVerificationError(err error) bool {
	for _, e := range []error{service.ErrPremisesRequired, service.ErrPremisesKind, service.ErrInvalidBIN, service.ErrCompanyName} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}
