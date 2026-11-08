package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/service"
)

type SetTariffRequest struct {
	Amount int64 `json:"amount"`
}

func (h *Handler) GetTariff(c *gin.Context) {
	jkID := c.Param("id")
	if !middleware.SameJK(c, jkID) {
		middleware.Forbidden(c)
		return
	}

	amount, err := h.svc.GetTariff(c.Request.Context(), jkID)
	if err != nil {
		log.Printf("GetTariff: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить тариф"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"amount": amount})
}

func (h *Handler) GetTariffPreview(c *gin.Context) {
	jkID := c.Param("id")
	if !middleware.SameJK(c, jkID) {
		middleware.Forbidden(c)
		return
	}

	amount, err := strconv.ParseInt(c.Query("amount"), 10, 64)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите новый тариф"})
		return
	}

	preview, err := h.svc.GetTariffPreview(c.Request.Context(), jkID, amount)
	if err != nil {
		log.Printf("GetTariffPreview: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось рассчитать перерасчёт"})
		return
	}
	c.JSON(http.StatusOK, preview)
}

func (h *Handler) SetTariff(c *gin.Context) {
	jkID := c.Param("id")
	if !middleware.SameJK(c, jkID) {
		middleware.Forbidden(c)
		return
	}

	var req SetTariffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
		return
	}

	err := h.svc.SetTariff(c.Request.Context(), service.SetTariffInput{JKID: jkID, Amount: req.Amount})
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Тариф должен быть больше нуля"})
			return
		}
		log.Printf("SetTariff: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить тариф"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tariff updated", "amount": req.Amount})
}
