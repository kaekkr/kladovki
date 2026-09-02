package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/middleware"
)

func (h *Handler) ListPaymentsByJK(c *gin.Context) {
	jkID := c.Param("jk_id")
	if jkID == "" {
		jkID = middleware.JKID(c)
	}

	if jkID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Доступ запрещен: ЖК не определен"})
		return
	}

	payments, err := h.svc.ListPaymentsByJK(c.Request.Context(), jkID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить список платежей"})
		return
	}

	c.JSON(http.StatusOK, payments)
}
