package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/models"
)

type loginReq struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerResidentReq struct {
	FullName string `json:"full_name" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	IIN      string `json:"iin" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type registerJKReq struct {
	Name     string `json:"jk_name" binding:"required"`
	BIN      string `json:"bin" binding:"required"`
	Contact  string `json:"contact" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func (h *Handler) APIRegisterResident(c *gin.Context) {
	var req registerResidentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.svc.RegisterResident(c.Request.Context(), req.FullName, req.Phone, req.Email, req.IIN, req.Password)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	jks, _ := h.svc.EgovStub(c.Request.Context(), req.IIN)
	if len(jks) == 1 {
		_ = h.svc.Repo().UpdateUserJKID(c.Request.Context(), u.ID, jks[0].ID)
		u.JKID = &jks[0].ID
	}

	token, err := h.issueToken(c, u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"user": gin.H{
			"id":        u.ID,
			"roles":     u.Roles,
			"email":     u.Email,
			"full_name": u.FullName,
			"jk_id":     u.JKID,
		},
	})
}

func (h *Handler) APIRegisterJK(c *gin.Context) {
	var req registerJKReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jk, u, err := h.svc.RegisterJK(c.Request.Context(), req.Name, req.BIN, req.Contact, req.Phone, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	u.JKID = &jk.ID
	token, err := h.issueToken(c, u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"user":         gin.H{"id": u.ID, "roles": u.Roles, "email": u.Email, "jk_id": jk.ID},
		"jk":           gin.H{"id": jk.ID, "name": jk.Name},
	})
}

func (h *Handler) APILogin(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.svc.Login(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if u.HasRole(models.RoleAdmin) && (u.JKID == nil || *u.JKID == "") {
		jks, _ := h.svc.Repo().ListJKs(c.Request.Context())
		for _, j := range jks {
			if j.OwnerID == u.ID {
				u.JKID = &j.ID
				_ = h.svc.Repo().UpdateUserJKID(c.Request.Context(), u.ID, j.ID)
				break
			}
		}
	}

	if u.HasRole(models.RoleResident) && (u.JKID == nil || *u.JKID == "") && u.IIN != nil {
		jks, _ := h.svc.EgovStub(c.Request.Context(), *u.IIN)
		if len(jks) == 1 {
			u.JKID = &jks[0].ID
			_ = h.svc.Repo().UpdateUserJKID(c.Request.Context(), u.ID, *u.JKID)
		}
	}

	token, err := h.issueToken(c, u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"user": gin.H{
			"id":        u.ID,
			"roles":     u.Roles,
			"email":     u.Email,
			"full_name": u.FullName,
			"jk_id":     u.JKID,
		},
	})
}

func (h *Handler) APIMe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"id":    middleware.UserID(c),
		"roles": middleware.Roles(c),
		"jk_id": middleware.JKID(c),
		"email": c.GetString(middleware.ContextEmail),
	})
}
