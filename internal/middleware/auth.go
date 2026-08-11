package middleware

import (
	"net/http"
	"strings"

	"github.com/kaekkr/kladovki/internal/auth"
	"github.com/kaekkr/kladovki/internal/config"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserID = "user_id"
	ContextRole   = "role"
	ContextJKID   = "jk_id"
	ContextEmail  = "email"
	ContextClaims = "claims"
)

type Auth struct {
	tokens *auth.TokenService
	cfg    config.Config
}

func NewAuth(tokens *auth.TokenService, cfg config.Config) *Auth {
	return &Auth{tokens: tokens, cfg: cfg}
}

// extractToken: Authorization: Bearer <token> или cookie
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

// Optional — парсит JWT если есть, не блокирует
func (a *Auth) Optional() gin.HandlerFunc {
	return func(c *gin.Context) {
		if tok := a.extractToken(c); tok != "" {
			if claims, err := a.tokens.Parse(tok); err == nil {
				setClaims(c, claims)
			}
		}
		c.Next()
	}
}

// Require — обязательный JWT
func (a *Auth) Require() gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := a.extractToken(c)
		if tok == "" {
			abortAuth(c)
			return
		}
		claims, err := a.tokens.Parse(tok)
		if err != nil {
			abortAuth(c)
			return
		}
		setClaims(c, claims)
		c.Next()
	}
}

// RequireRole — JWT + роль
func (a *Auth) RequireRole(roles ...string) gin.HandlerFunc {
	set := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		set[r] = struct{}{}
	}
	return func(c *gin.Context) {
		tok := a.extractToken(c)
		if tok == "" {
			abortAuth(c)
			return
		}
		claims, err := a.tokens.Parse(tok)
		if err != nil {
			abortAuth(c)
			return
		}
		if _, ok := set[claims.Role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		setClaims(c, claims)
		c.Next()
	}
}

func setClaims(c *gin.Context, claims *auth.Claims) {
	c.Set(ContextUserID, claims.UserID)
	c.Set(ContextRole, claims.Role)
	c.Set(ContextJKID, claims.JKID)
	c.Set(ContextEmail, claims.Email)
	c.Set(ContextClaims, claims)
}

func abortAuth(c *gin.Context) {
	// HTMX / browser: redirect; API: JSON
	if c.GetHeader("HX-Request") == "true" || strings.Contains(c.GetHeader("Accept"), "text/html") {
		c.Redirect(http.StatusFound, "/client/login")
		c.Abort()
		return
	}
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
}

// Helpers из контекста
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
