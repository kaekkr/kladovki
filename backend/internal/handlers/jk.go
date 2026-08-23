package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/service"
)

func (h *Handler) GetJKByID(c *gin.Context) {
	id := c.Param("id")

	jk, err := h.svc.GetJKByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) || errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "ЖК не найден",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось получить информацию о ЖК",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         jk.ID,
		"name":       jk.Name,
		"created_at": jk.CreatedAt,
	})
}
