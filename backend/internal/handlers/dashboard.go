package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/service"
)

func (h *Handler) GetDashboardStats(c *gin.Context) {
	period := models.DashboardPeriod(
		c.DefaultQuery(
			"period",
			string(models.DashboardPeriod30Days),
		),
	)

	jkID := middleware.JKID(c)

	stats, err := h.svc.GetDashboardStats(
		c.Request.Context(),
		jkID,
		period,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Некорректные параметры",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось получить статистику",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *Handler) GetDashboardRevenue(c *gin.Context) {
	period := models.DashboardPeriod(
		c.DefaultQuery(
			"period",
			string(models.DashboardPeriod30Days),
		),
	)

	jkID := middleware.JKID(c)

	revenue, err := h.svc.GetDashboardRevenue(
		c.Request.Context(),
		jkID,
		period,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Некорректные параметры",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось получить данные выручки",
		})
		return
	}

	c.JSON(http.StatusOK, revenue)
}

func (h *Handler) GetDashboardAttention(c *gin.Context) {
	jkID := middleware.JKID(c)

	attention, err := h.svc.GetDashboardAttention(
		c.Request.Context(),
		jkID,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Некорректные параметры",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось получить данные для внимания",
		})
		return
	}

	c.JSON(http.StatusOK, attention)
}

func (h *Handler) GetDashboardActivity(c *gin.Context) {
	jkID := middleware.JKID(c)

	activity, err := h.svc.GetDashboardActivity(
		c.Request.Context(),
		jkID,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Некорректные параметры",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось получить активность",
		})
		return
	}

	c.JSON(http.StatusOK, activity)
}

func (h *Handler) GetDashboardOccupancy(c *gin.Context) {
	jkID := middleware.JKID(c)

	occupancy, err := h.svc.GetDashboardOccupancy(
		c.Request.Context(),
		jkID,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Некорректные параметры",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось получить данные по кладовым",
		})
		return
	}

	c.JSON(http.StatusOK, occupancy)
}
