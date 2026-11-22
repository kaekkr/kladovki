package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/service"
)

// Platform operator routes (/platform/..., superadmin only — see router.go).

type jkDetailsRequest struct {
	Name            string `json:"name"`
	Address         string `json:"address"`
	CadastralNumber string `json:"cadastral_number"`
}

type accountRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

type createJKRequest struct {
	jkDetailsRequest
	Admin accountRequest `json:"admin"`
}

func (r jkDetailsRequest) details() service.JKDetails {
	return service.JKDetails{Name: r.Name, Address: r.Address, CadastralNumber: r.CadastralNumber}
}

func (r accountRequest) account() service.AccountInput {
	return service.AccountInput{FullName: r.FullName, Email: r.Email, Phone: r.Phone}
}

// adminJSON is an ОСИ account as the operator sees it (no password hash, no ИИН).
func adminJSON(u *models.User) gin.H {
	return gin.H{
		"id":                   u.ID,
		"full_name":            u.FullName,
		"email":                u.Email,
		"phone":                u.Phone,
		"jk_id":                u.JKID,
		"created_at":           u.CreatedAt,
		"must_change_password": u.MustChangePassword,
	}
}

func adminsJSON(list []*models.User) []gin.H {
	out := make([]gin.H, 0, len(list))
	for _, u := range list {
		out = append(out, adminJSON(u))
	}
	return out
}

func (h *Handler) PlatformListJKs(c *gin.Context) {
	list, err := h.svc.ListJKSummaries(c.Request.Context())
	if err != nil {
		platformError(c, "PlatformListJKs", err, "Не удалось загрузить список ЖК")
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) PlatformGetJK(c *gin.Context) {
	jk, err := h.svc.GetJKSummary(c.Request.Context(), c.Param("id"))
	if err != nil {
		platformError(c, "PlatformGetJK", err, "Не удалось загрузить ЖК")
		return
	}
	c.JSON(http.StatusOK, jk)
}

func (h *Handler) PlatformCreateJK(c *gin.Context) {
	var req createJKRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
		return
	}
	jk, acc, err := h.svc.CreateJKWithAdmin(c.Request.Context(), req.details(), req.Admin.account())
	if err != nil {
		platformError(c, "PlatformCreateJK", err, "Не удалось создать ЖК")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"jk":                 jk,
		"admin":              adminJSON(acc.User),
		"temporary_password": acc.TemporaryPassword,
	})
}

func (h *Handler) PlatformUpdateJK(c *gin.Context) {
	var req jkDetailsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
		return
	}
	jk, err := h.svc.UpdateJKDetails(c.Request.Context(), c.Param("id"), req.details())
	if err != nil {
		platformError(c, "PlatformUpdateJK", err, "Не удалось сохранить ЖК")
		return
	}
	c.JSON(http.StatusOK, jk)
}

func (h *Handler) PlatformListAdmins(c *gin.Context) {
	list, err := h.svc.ListJKAdmins(c.Request.Context(), c.Param("id"))
	if err != nil {
		platformError(c, "PlatformListAdmins", err, "Не удалось загрузить администраторов")
		return
	}
	c.JSON(http.StatusOK, adminsJSON(list))
}

func (h *Handler) PlatformAddAdmin(c *gin.Context) {
	var req accountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Некорректные данные"})
		return
	}
	acc, err := h.svc.AddJKAdmin(c.Request.Context(), c.Param("id"), req.account())
	if err != nil {
		platformError(c, "PlatformAddAdmin", err, "Не удалось добавить администратора")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"admin": adminJSON(acc.User), "temporary_password": acc.TemporaryPassword})
}

func (h *Handler) PlatformResetAdminPassword(c *gin.Context) {
	temp, err := h.svc.ResetAdminPassword(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Администратор ОСИ не найден"})
			return
		}
		platformError(c, "PlatformResetAdminPassword", err, "Не удалось сбросить пароль")
		return
	}
	c.JSON(http.StatusOK, gin.H{"temporary_password": temp})
}

// platformError maps service errors to status codes; unknown errors are logged, not shown.
func platformError(c *gin.Context, op string, err error, fallback string) {
	switch {
	case service.IsValidationError(err):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrEmailTaken), errors.Is(err, service.ErrPhoneTaken),
		errors.Is(err, service.ErrCadastralTaken):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "ЖК не найден"})
	default:
		log.Printf("%s: %v", op, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fallback})
	}
}
