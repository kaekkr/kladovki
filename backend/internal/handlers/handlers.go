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

// --- Auth Handlers ---

// Logout clears auth cookie and redirects home
func (h *Handler) Logout(c *gin.Context) {
	h.clearAuthCookie(c)
	c.Redirect(http.StatusFound, "/")
}

// --- Cookie & Token Helpers ---

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

	roles := make([]string, len(u.Roles))
	for i, r := range u.Roles {
		roles[i] = string(r)
	}

	token, exp, err := h.tokens.Generate(u.ID, roles, jkID, u.Email)
	if err != nil {
		return "", err
	}
	h.setAuthCookie(c, token, exp)
	return token, nil
}

// --- Render Helpers ---

func (h *Handler) renderToastError(c *gin.Context, message string) {
	c.HTML(http.StatusOK, "components/ui/toast.html", gin.H{"Message": message, "Type": "error"})
}

func (h *Handler) renderToastSuccess(c *gin.Context, message string) {
	c.HTML(http.StatusOK, "components/ui/toast.html", gin.H{"Message": message, "Type": "success"})
}
