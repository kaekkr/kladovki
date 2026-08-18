package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/service"
)

type loginRequest struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Заполните обязательные поля",
		})
		return
	}

	// Use email or phone as the login identifier
	loginIdentifier := req.Email
	if loginIdentifier == "" {
		loginIdentifier = req.Phone
	}

	if loginIdentifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Укажите Email или номер телефона",
		})
		return
	}

	user, err := h.svc.Login(c.Request.Context(), service.LoginInput{
		Login:    loginIdentifier,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Неверный логин или пароль",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось выполнить вход",
		})
		return
	}

	token, err := h.issueToken(c, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось сгенерировать токен авторизации",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"id":           user.ID,
		"full_name":    user.FullName,
		"phone":        user.Phone,
		"email":        user.Email,
		"role":         user.Role,
		"jk_id":        user.JKID,
	})
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

	c.JSON(http.StatusOK, gin.H{
		"id":        user.ID,
		"full_name": user.FullName,
		"phone":     user.Phone,
		"email":     user.Email,
		"role":      user.Role,
		"jk_id":     user.JKID,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	h.clearAuthCookie(c)
	c.Redirect(http.StatusFound, "/")
}

// ---------- Cookie & Token Helpers ----------

func (h *Handler) setAuthCookie(c *gin.Context, token string, expires time.Time) {
	maxAge := max(int(time.Until(expires).Seconds()), 0)
	c.SetCookie(h.cfg.CookieName, token, maxAge, "/", "", h.cfg.CookieSecure, true)
}

func (h *Handler) clearAuthCookie(c *gin.Context) {
	c.SetCookie(h.cfg.CookieName, "", -1, "/", "", h.cfg.CookieSecure, true)
}

func (h *Handler) issueToken(c *gin.Context, u *models.User) (string, error) {
	jkID := ""
	if u.JKID != nil {
		jkID = *u.JKID
	}

	token, exp, err := h.tokens.Generate(u.ID, string(u.Role), jkID, u.Email)
	if err != nil {
		return "", err
	}
	h.setAuthCookie(c, token, exp)
	return token, nil
}
