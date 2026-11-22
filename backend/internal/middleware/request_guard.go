package middleware

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes: no API request needs more (the largest are JSON forms and
// ApiPay webhooks of a few KB).
const MaxBodyBytes int64 = 1 << 20

// BodyLimit answers 413 when Content-Length is over limit and caps the body
// reader otherwise (a chunked body then fails to parse: 400).
func BodyLimit(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > limit {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Слишком большой запрос"})
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

// ParamGuard stops values Postgres can't store (NUL, invalid UTF-8) before
// they reach SQL: a bad path id is 404 (no such record), a bad query value 400.
func ParamGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, p := range c.Params {
			if !cleanValue(p.Value) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Не найдено"})
				return
			}
		}
		for _, vals := range c.Request.URL.Query() {
			for _, v := range vals {
				if !cleanValue(v) {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Недопустимые символы в запросе"})
					return
				}
			}
		}
		c.Next()
	}
}

func cleanValue(v string) bool {
	return utf8.ValidString(v) && !strings.ContainsRune(v, 0)
}
