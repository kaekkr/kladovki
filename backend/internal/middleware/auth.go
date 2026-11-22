package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/config"
	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/repository"
	"github.com/kaekkr/kladovki/internal/service"
)

const (
	ContextUserID = "user_id"
	ContextRole   = "role"
	ContextJKID   = "jk_id"
	ContextEmail  = "email"
	ContextClaims = "claims"
	ContextStatus = "user_status"
)

// SessionStore re-reads the user behind a token (see service.SessionUser).
// It returns repository.ErrNotFound when the user no longer exists.
type SessionStore interface {
	SessionUser(ctx context.Context, userID string) (*models.SessionUser, error)
}

type Auth struct {
	tokens   *service.TokenService
	cfg      config.Config
	sessions SessionStore
}

func NewAuth(tokens *service.TokenService, cfg config.Config, sessions SessionStore) *Auth {
	return &Auth{tokens: tokens, cfg: cfg, sessions: sessions}
}

func (a *Auth) extractToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if t, err := c.Cookie(a.cfg.CookieName); err == nil && t != "" {
		return t
	}
	return ""
}

// ValidClaims returns the request's token claims when the signature and
// expiry are valid. It does NOT check the database: use it only where a
// revoked token can do no harm (logout), never for authorization.
func (a *Auth) ValidClaims(c *gin.Context) (*service.Claims, bool) {
	tok := a.extractToken(c)
	if tok == "" {
		return nil, false
	}
	claims, err := a.tokens.Parse(tok)
	return claims, err == nil
}

var errSessionGone = errors.New("session revoked")

// authenticate validates the token and re-reads the user (one primary-key
// query): the session is rejected when the user is gone or token_version
// changed (logout, password change or reset). Role and ЖК come from the DB.
func (a *Auth) authenticate(c *gin.Context) (*models.SessionUser, error) {
	claims, ok := a.ValidClaims(c)
	if !ok {
		return nil, errSessionGone
	}
	user, err := a.sessions.SessionUser(c.Request.Context(), claims.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, errSessionGone
	}
	if err != nil {
		return nil, err
	}
	if user.TokenVersion != claims.TokenVersion {
		return nil, errSessionGone
	}
	setSession(c, claims, user)
	return user, nil
}

// Optional sets the session when a valid one is present; it never rejects.
func (a *Auth) Optional() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, _ = a.authenticate(c)
		c.Next()
	}
}

func (a *Auth) Require() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Bypass preflight checks
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		if a.check(c) {
			c.Next()
		}
	}
}

// check validates the session and the temporary-password rule; it aborts and
// returns false when the request must not continue.
func (a *Auth) check(c *gin.Context) bool {
	user, ok := a.requireSession(c)
	if !ok {
		return false
	}
	if user.MustChangePassword && !a.allowedBeforePasswordChange(c) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Сначала смените временный пароль",
			"code":  "password_change_required",
		})
		return false
	}
	return true
}

// requireSession aborts with 401 (or 503 when the DB is down) unless the session is valid.
func (a *Auth) requireSession(c *gin.Context) (*models.SessionUser, bool) {
	user, err := a.authenticate(c)
	switch {
	case errors.Is(err, errSessionGone):
		abortAuth(c)
		return nil, false
	case err != nil:
		log.Printf("auth: session lookup: %v", err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Сервер временно недоступен. Попробуйте позже."})
		return nil, false
	}
	return user, true
}

// allowedBeforePasswordChange: while must_change_password is set, the user may
// only load their profile, change the password and see/accept the agreements
// (the change-password overlay and the ОСИ rules prompt). Logout is public.
func (a *Auth) allowedBeforePasswordChange(c *gin.Context) bool {
	switch c.Request.Method + " " + strings.TrimPrefix(c.FullPath(), a.cfg.APIPrefix) {
	case "GET /auth/me", "PUT /me/password", "GET /legal/status", "POST /legal/accept":
		return true
	}
	return false
}

func (a *Auth) RequireRole(allowedRoles ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowedSet[r] = struct{}{}
	}

	return func(c *gin.Context) {
		// Bypass preflight checks
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		if !a.check(c) {
			return
		}
		if _, ok := allowedSet[Role(c)]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

// setSession stores the DB view of the user (role, ЖК) for downstream checks.
func setSession(c *gin.Context, claims *service.Claims, user *models.SessionUser) {
	c.Set(ContextUserID, user.ID)
	c.Set(ContextRole, string(user.Role))
	c.Set(ContextJKID, user.JKID)
	c.Set(ContextStatus, string(user.Status))
	c.Set(ContextEmail, claims.Email)
	c.Set(ContextClaims, claims)
}

func abortAuth(c *gin.Context) {
	if c.GetHeader("HX-Request") == "true" || strings.Contains(c.GetHeader("Accept"), "text/html") {
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
		return
	}
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
}

// Context Context Helpers

func UserID(c *gin.Context) string {
	v, _ := c.Get(ContextUserID)
	s, _ := v.(string)
	return s
}

func Role(c *gin.Context) string {
	v, _ := c.Get(ContextRole)
	s, _ := v.(string)
	return s
}

func JKID(c *gin.Context) string {
	v, _ := c.Get(ContextJKID)
	s, _ := v.(string)
	return s
}

// RequireAdmin must run after Require(); it rejects non-admin users with 403.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if Role(c) != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Доступ только для администратора ОСИ"})
			return
		}
		c.Next()
	}
}

// RequireSuperadmin must run after Require(); it lets only the platform
// operator through. A superadmin has no ЖК and does NOT pass RequireAdmin.
func RequireSuperadmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if Role(c) != "superadmin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Доступ только для оператора платформы"})
			return
		}
		c.Next()
	}
}

// IsAdmin reports whether the authenticated user is an OSI admin.
func IsAdmin(c *gin.Context) bool { return Role(c) == "admin" }

// SameJK reports whether jkID is the authenticated user's own JK.
func SameJK(c *gin.Context, jkID string) bool {
	own := JKID(c)
	return own != "" && own == jkID
}

// Forbidden writes a 403 and aborts.
func Forbidden(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Нет доступа к данным другого ЖК"})
}
