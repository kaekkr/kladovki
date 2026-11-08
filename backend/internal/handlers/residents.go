package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/service"
)

// ListResidents: ОСИ view of residents of their ЖК (?status=pending|active|rejected|blocked).
func (h *Handler) ListResidents(c *gin.Context) {
	jkID := c.Param("jk_id")
	if !middleware.SameJK(c, jkID) {
		middleware.Forbidden(c)
		return
	}
	list, err := h.svc.ListResidents(c.Request.Context(), jkID, c.Query("status"))
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неизвестный статус"})
			return
		}
		log.Printf("ListResidents: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить жителей"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) CountPendingResidents(c *gin.Context) {
	n, err := h.svc.CountPendingResidents(c.Request.Context(), middleware.JKID(c))
	if err != nil {
		log.Printf("CountPendingResidents: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось посчитать заявки"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pending": n})
}

type reviewRequest struct {
	Comment string `json:"comment"`
}

// ReviewResident: PATCH /residents/:id/approve|reject|block.
func (h *Handler) ReviewResident(c *gin.Context) {
	var req reviewRequest
	_ = c.ShouldBindJSON(&req)
	err := h.svc.ReviewResident(c.Request.Context(), middleware.JKID(c), middleware.UserID(c),
		c.Param("id"), c.Param("action"), req.Comment)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"message": "Сохранено"})
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Житель не найден"})
	case errors.Is(err, service.ErrReasonRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalid):
		c.JSON(http.StatusNotFound, gin.H{"error": "Неизвестное действие"})
	default:
		log.Printf("ReviewResident: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить решение"})
	}
}

// ResubmitVerification: a pending or rejected user corrects their property data.
func (h *Handler) ResubmitVerification(c *gin.Context) {
	var req models.Verification
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
		return
	}
	err := h.svc.ResubmitVerification(c.Request.Context(), middleware.UserID(c), req)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"message": "Заявка отправлена ОСИ"})
	case errors.Is(err, service.ErrAlreadyReviewed):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case isVerificationError(err), errors.Is(err, service.ErrMissingFields):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		log.Printf("ResubmitVerification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось отправить заявку"})
	}
}
