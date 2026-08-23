package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/service"
)

type createStorageRequest struct {
	JKID     string  `json:"jk_id"`
	Number   string  `json:"number" binding:"required"`
	Area     float64 `json:"area" binding:"required,gt=0"`
	Floor    int     `json:"floor"`
	Entrance int     `json:"entrance"`
}

func (h *Handler) CreateStorage(c *gin.Context) {
	var req createStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные параметры запроса"})
		return
	}

	// If JKID isn't passed in payload, fall back to user's assigned JKID from context
	jkID := req.JKID
	if jkID == "" {
		jkID = middleware.JKID(c)
	}

	storage, err := h.svc.CreateStorage(c.Request.Context(), service.CreateStorageInput{
		JKID:     jkID,
		Number:   req.Number,
		Area:     req.Area,
		Floor:    req.Floor,
		Entrance: req.Entrance,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные кладовки"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать кладовку"})
		return
	}

	c.JSON(http.StatusCreated, storage)
}

func (h *Handler) ListStoragesByJK(c *gin.Context) {
	jkID := c.Param("jk_id")
	if jkID == "" {
		jkID = middleware.JKID(c)
	}

	storages, err := h.svc.ListStoragesByJK(c.Request.Context(), jkID)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректный ID ЖК"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить список кладовок"})
		return
	}

	c.JSON(http.StatusOK, storages)
}

func (h *Handler) GetStorageByID(c *gin.Context) {
	id := c.Param("id")

	storage, err := h.svc.GetStorageByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) || errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Кладовка не найдена"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить кладовку"})
		return
	}

	c.JSON(http.StatusOK, storage)
}

type updateStorageRequest struct {
	Number   string  `json:"number" binding:"required"`
	Area     float64 `json:"area" binding:"required,gt=0"`
	Floor    int     `json:"floor"`
	Entrance int     `json:"entrance"`
	Status   string  `json:"status"`
}

func (h *Handler) UpdateStorage(c *gin.Context) {
	id := c.Param("id")

	var req updateStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные параметры запроса"})
		return
	}

	storage, err := h.svc.UpdateStorage(c.Request.Context(), service.UpdateStorageInput{
		ID:       id,
		Number:   req.Number,
		Area:     req.Area,
		Floor:    req.Floor,
		Entrance: req.Entrance,
		Status:   models.StorageStatus(req.Status),
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Кладовка не найдена"})
			return
		}
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные данные кладовки"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить кладовку"})
		return
	}

	c.JSON(http.StatusOK, storage)
}

func (h *Handler) DeleteStorage(c *gin.Context) {
	id := c.Param("id")

	if err := h.svc.DeleteStorage(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrNotFound) || errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Кладовка не найдена"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось удалить кладовку"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Кладовка успешно удалена"})
}
