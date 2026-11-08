package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/service"
)

func requestError(c *gin.Context, err error, conflictMsg string) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Заявка не найдена"})
	case errors.Is(err, service.ErrAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": conflictMsg})
	case errors.Is(err, service.ErrInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные"})
	default:
		log.Printf("requests: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обработать заявку"})
	}
}

func (h *Handler) ListMyRequests(c *gin.Context) {
	list, err := h.svc.ListRequestsByUser(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		requestError(c, err, "")
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) ListRequestsByJK(c *gin.Context) {
	jkID := c.Param("jk_id")
	if !middleware.SameJK(c, jkID) {
		middleware.Forbidden(c)
		return
	}
	list, err := h.svc.ListRequestsByJK(c.Request.Context(), jkID)
	if err != nil {
		requestError(c, err, "")
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) CountOpenRequests(c *gin.Context) {
	n, err := h.svc.CountOpenRequests(c.Request.Context(), middleware.JKID(c))
	if err != nil {
		requestError(c, err, "")
		return
	}
	c.JSON(http.StatusOK, gin.H{"open": n})
}

type choiceRequest struct {
	Choice string `json:"choice" binding:"required"`
}

func (h *Handler) ChooseTariffOption(c *gin.Context) {
	var req choiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Выберите вариант перерасчёта"})
		return
	}
	err := h.svc.ChooseTariffOption(c.Request.Context(), c.Param("id"), middleware.UserID(c), req.Choice)
	if err != nil {
		requestError(c, err, "Вы уже выбрали вариант перерасчёта")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Выбор сохранён"})
}

type terminateRequest struct {
	Comment string `json:"comment"`
}

func (h *Handler) RequestEarlyTermination(c *gin.Context) {
	var req terminateRequest
	_ = c.ShouldBindJSON(&req) // body is optional
	q, err := h.svc.RequestEarlyTermination(c.Request.Context(), c.Param("id"), middleware.UserID(c), req.Comment)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Активная аренда не найдена"})
			return
		}
		requestError(c, err, "Заявка на досрочное завершение уже отправлена")
		return
	}
	c.JSON(http.StatusCreated, q)
}

type resolveRequest struct {
	Action  string `json:"action" binding:"required"` // approve | reject
	Comment string `json:"comment"`
}

func (h *Handler) ResolveRequest(c *gin.Context) {
	var req resolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите решение"})
		return
	}
	err := h.svc.ResolveRequest(c.Request.Context(), c.Param("id"), middleware.JKID(c),
		middleware.UserID(c), req.Action, req.Comment)
	if err != nil {
		requestError(c, err, "Заявка уже обработана")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Заявка обработана"})
}
