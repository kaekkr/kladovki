package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/legal"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/repository"
	"github.com/kaekkr/kladovki/internal/service"
)

// GetAgreement: public text of an agreement (?lang=ru|kk).
func (h *Handler) GetAgreement(c *gin.Context) {
	d, ok := legal.Get(c.Param("doc"), c.Query("lang"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Документ не найден"})
		return
	}
	c.JSON(http.StatusOK, d)
}

func (h *Handler) AgreementStatus(c *gin.Context) {
	st, err := h.svc.LegalStatus(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		log.Printf("AgreementStatus: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось проверить соглашение"})
		return
	}
	c.JSON(http.StatusOK, st)
}

type acceptRequest struct {
	Doc     string `json:"doc" binding:"required"`
	Version string `json:"version" binding:"required"`
}

// AcceptAgreement: explicit consent (e.g. the ОСИ regulation prompt in the admin panel).
func (h *Handler) AcceptAgreement(c *gin.Context) {
	var req acceptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
		return
	}
	if req.Doc != service.DocumentForRole(models.Role(middleware.Role(c))) {
		middleware.Forbidden(c)
		return
	}
	if !service.TermsAccepted(req.Doc, req.Version) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Текст соглашения обновился. Обновите страницу."})
		return
	}
	h.recordConsent(c, middleware.UserID(c), req.Doc, "prompt", "")
	c.JSON(http.StatusOK, gin.H{"message": "Принято"})
}

// recordConsent stores who accepted the current version of `doc`, with request metadata as evidence.
func (h *Handler) recordConsent(c *gin.Context, userID, doc, context, rentalID string) {
	h.svc.RecordAcceptance(c.Request.Context(), repository.Acceptance{
		UserID:    userID,
		Document:  doc,
		Version:   legal.Version(doc),
		Context:   context,
		RentalID:  rentalID,
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
}

// termsRequired answers 400 unless the request carries the current resident agreement version.
func termsRequired(c *gin.Context, version string) bool {
	if service.TermsAccepted(legal.ResidentTerms, version) {
		return false
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": service.ErrTermsNotAccepted.Error(), "terms_version": legal.Version(legal.ResidentTerms)})
	return true
}
