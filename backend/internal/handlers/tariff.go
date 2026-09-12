package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/service"
)

type SetTariffRequest struct {
	Amount int64 `json:"amount"`
}

func (h *Handler) GetTariff(c *gin.Context) {
	jkID := c.Param("id")

	amount, err := h.svc.GetTariff(c.Request.Context(), jkID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"amount": amount,
	})
}

func (h *Handler) SetTariff(c *gin.Context) {
	jkID := c.Param("id")

	// Пока проверяем, что пользователь вообще авторизован.
	// Для полноценной защиты ниже можно добавить проверку admin role.
	if middleware.UserID(c) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	var req SetTariffRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	err := h.svc.SetTariff(c.Request.Context(), service.SetTariffInput{
		JKID:   jkID,
		Amount: req.Amount,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "tariff updated",
		"amount":  req.Amount,
	})
}
