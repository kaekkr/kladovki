package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/service"
)

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

// ChangePassword: PUT /me/password.
func (h *Handler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите текущий и новый пароль"})
		return
	}
	err := h.svc.ChangePassword(c.Request.Context(), middleware.UserID(c), req.CurrentPassword, req.NewPassword)
	switch {
	case err == nil:
		h.freshSessionAfterPasswordChange(c)
	case errors.Is(err, service.ErrWrongPassword), errors.Is(err, service.ErrWeakPassword),
		errors.Is(err, service.ErrPasswordTooLong):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		log.Printf("ChangePassword: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось изменить пароль"})
	}
}

// freshSessionAfterPasswordChange: the change revoked every session (other
// devices are logged out); the caller gets a new cookie, and the mobile app
// (X-Kladovki-Client: native) a new access_token, so it stays logged in.
func (h *Handler) freshSessionAfterPasswordChange(c *gin.Context) {
	out := gin.H{"message": "Пароль изменён"}
	user, err := h.svc.Me(c.Request.Context(), middleware.UserID(c))
	if err == nil {
		var token string
		if token, err = h.issueToken(c, user); err == nil && c.GetHeader("X-Kladovki-Client") == "native" {
			out["access_token"] = token
		}
	}
	if err != nil {
		// The password did change; the user just has to log in again.
		log.Printf("ChangePassword: new session: %v", err)
	}
	c.JSON(http.StatusOK, out)
}

// ResetResidentPassword: POST /residents/:id/reset-password (ОСИ, own ЖК).
func (h *Handler) ResetResidentPassword(c *gin.Context) {
	temp, err := h.svc.ResetResidentPassword(c.Request.Context(), middleware.JKID(c), c.Param("id"))
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"temporary_password": temp})
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Житель не найден"})
	default:
		log.Printf("ResetResidentPassword: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сбросить пароль"})
	}
}
