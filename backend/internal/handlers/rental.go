package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/repository"
	"github.com/kaekkr/kladovki/internal/service"
)

func (h *Handler) ListActiveRentalsByJK(c *gin.Context) {
	jkID := c.Param("jk_id")
	if !middleware.SameJK(c, jkID) {
		middleware.Forbidden(c)
		return
	}

	rentals, err := h.svc.ListActiveRentalsByJK(c.Request.Context(), jkID)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Некорректный ID ЖК",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось получить список аренд",
		})
		return
	}

	c.JSON(http.StatusOK, rentals)
}

func (h *Handler) GetRentalByID(c *gin.Context) {
	id := c.Param("id")

	rental, err := h.svc.GetRentalByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) || errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Аренда не найдена",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось получить аренду",
		})
		return
	}

	if !canSeeRental(c, rental.UserID, rental.JKID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Аренда не найдена"})
		return
	}

	if rental.Status == models.RentalStatusExpired || rental.Status == models.RentalStatusCancelled {
		if t, err := h.svc.GetTermination(c.Request.Context(), rental.ID); err == nil {
			rental.Termination = t
		}
	}
	c.JSON(http.StatusOK, rental)
}

// canSeeRental: the tenant themself, or an admin of the rental's JK.
func canSeeRental(c *gin.Context, ownerID, jkID string) bool {
	if middleware.UserID(c) == ownerID {
		return true
	}
	return middleware.IsAdmin(c) && middleware.SameJK(c, jkID)
}

func (h *Handler) ListMyRentals(c *gin.Context) {
	rentals, err := h.svc.ListRentalsByUser(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить список аренд"})
		return
	}
	if rentals == nil {
		rentals = []*models.Rental{}
	}
	c.JSON(http.StatusOK, rentals)
}

func (h *Handler) ListRentalsByJK(c *gin.Context) {
	jkID := c.Param("jk_id")
	if !middleware.SameJK(c, jkID) {
		middleware.Forbidden(c)
		return
	}

	rentals, err := h.svc.ListRentalsByJK(c.Request.Context(), jkID)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Некорректный ID ЖК",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось получить список аренд",
		})
		return
	}

	c.JSON(http.StatusOK, rentals)
}

func (h *Handler) GetActiveRentalByStorage(c *gin.Context) {
	storageID := c.Param("storage_id")

	rental, err := h.svc.GetActiveRentalByStorage(
		c.Request.Context(),
		storageID,
	)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Активная аренда не найдена",
			})
			return
		}

		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Некорректный ID кладовой",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось получить аренду",
		})
		return
	}

	if !middleware.SameJK(c, rental.JKID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Активная аренда не найдена"})
		return
	}

	c.JSON(http.StatusOK, rental)
}

func (h *Handler) CancelRental(c *gin.Context) {
	id := c.Param("id")
	jkID := middleware.JKID(c)

	rental, err := h.svc.CancelRental(
		c.Request.Context(),
		id,
		jkID,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Некорректный ID аренды",
			})
			return
		}

		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Аренда не найдена",
			})
			return
		}

		c.JSON(http.StatusConflict, gin.H{
			"error": "Аренду нельзя завершить",
		})
		return
	}

	c.JSON(http.StatusOK, rental)
}

type releaseOverdueRequest struct {
	Reason string `json:"reason"`
	Note   string `json:"note"`
	Police bool   `json:"police"`
	Costs  int64  `json:"costs"`
}

// ReleaseOverdueRental: ОСИ ends an overdue rental for debt, with a termination record.
func (h *Handler) ReleaseOverdueRental(c *gin.Context) {
	var req releaseOverdueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите причину расторжения"})
		return
	}
	rental, err := h.svc.ReleaseOverdueRental(c.Request.Context(), c.Param("id"), middleware.JKID(c), middleware.UserID(c),
		service.TerminationInput{Reason: req.Reason, Note: req.Note, Police: req.Police, Costs: req.Costs})
	switch {
	case err == nil:
		c.JSON(http.StatusOK, rental)
	case errors.Is(err, service.ErrReasonRequiredTermination):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Аренда не найдена"})
	default:
		// Not overdue long enough / not active — never expose internal error text.
		c.JSON(http.StatusConflict, gin.H{"error": "Освободить можно только аренду с длительной просрочкой"})
	}
}

func (h *Handler) ForceReleaseLocked(c *gin.Context) {
	id := c.Param("id")
	jkID := middleware.JKID(c)

	rental, err := h.svc.ForceReleaseLocked(c.Request.Context(), id, jkID)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Некорректный ID аренды",
			})
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Аренда не найдена",
			})
			return
		}
		// status != locked (or any other business conflict)
		c.JSON(http.StatusConflict, gin.H{
			"error": "Можно снять только заблокированную (неоплаченную) аренду",
		})
		return
	}

	c.JSON(http.StatusOK, rental)
}

func (h *Handler) LockStorage(c *gin.Context) {
	var input service.LockStorageInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректные данные",
		})
		return
	}

	// UserID should come from authenticated middleware,
	// not from the client request.
	input.UserID = middleware.UserID(c)
	if _, ok := h.storageInMyJK(c, input.StorageID); !ok {
		return
	}

	rental, err := h.svc.LockStorage(
		c.Request.Context(),
		input,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Некорректные данные аренды",
			})
			return
		}

		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Кладовка не найдена",
			})
			return
		}

		if errors.Is(err, service.ErrNotApproved) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, repository.ErrStorageAlreadyLocked) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Кладовка уже занята или забронирована",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Не удалось забронировать кладовку",
		})
		return
	}

	c.JSON(http.StatusCreated, rental)
}

func (h *Handler) ConfirmPayment(c *gin.Context) {
	var input service.ConfirmPaymentInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректные данные",
		})
		return
	}

	input.UserID = middleware.UserID(c)

	rental, err := h.svc.ConfirmPayment(
		c.Request.Context(),
		input,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Некорректные данные оплаты",
			})
			return
		}

		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Аренда не найдена",
			})
			return
		}

		if errors.Is(err, service.ErrLocked) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Срок бронирования истёк",
			})
			return
		}

		c.JSON(http.StatusConflict, gin.H{
			"error": "Не удалось подтвердить оплату",
		})
		return
	}

	c.JSON(http.StatusOK, rental)
}
