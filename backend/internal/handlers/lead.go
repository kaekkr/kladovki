package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/service"
)

type createLeadRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Email    string `json:"email"`
}

func (h *Handler) CreateLead(c *gin.Context) {
	var req createLeadRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Заполните обязательные поля (Имя и Телефон)",
		})
		return
	}

	lead, err := h.svc.CreateLead(c.Request.Context(), service.CreateLeadInput{
		FullName: req.FullName,
		Phone:    req.Phone,
		Email:    req.Email,
	})
	if err != nil {
		if err == service.ErrInvalid {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Заполните обязательные поля (Имя и Телефон)",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось сохранить заявку",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Заявка успешно принята",
		"lead":    lead,
	})
}
