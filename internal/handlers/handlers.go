package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/auth"
	"github.com/kaekkr/kladovki/internal/config"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/service"
)

type Handler struct {
	svc    *service.Service
	tokens *auth.TokenService
	cfg    config.Config
	mw     *middleware.Auth
}

func New(svc *service.Service, tokens *auth.TokenService, cfg config.Config, mw *middleware.Auth) *Handler {
	return &Handler{svc: svc, tokens: tokens, cfg: cfg, mw: mw}
}

// --- Cookie & Token Helpers ---

func (h *Handler) setAuthCookie(c *gin.Context, token string, expires time.Time) {
	maxAge := int(time.Until(expires).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
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

// --- Render Helpers ---

func (h *Handler) renderToastError(c *gin.Context, message string) {
	c.HTML(http.StatusOK, "partials/toast.html", gin.H{"Message": message, "Type": "error"})
}

func (h *Handler) renderToastSuccess(c *gin.Context, message string) {
	c.HTML(http.StatusOK, "partials/toast.html", gin.H{"Message": message, "Type": "success"})
}
