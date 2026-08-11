package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/auth"
	"github.com/kaekkr/kladovki/internal/service"
)

func (h *Handler) ShowIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "client/index.html", gin.H{
		"User": getUserFromCtx(c),
	})
}

func (h *Handler) ClientRegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "client/register.html", nil)
}

func (h *Handler) ClientRegister(c *gin.Context) {
	fullName := c.PostForm("full_name")
	phone := c.PostForm("phone")
	email := c.PostForm("email")
	iin := c.PostForm("iin")
	password := c.PostForm("password")

	u, err := h.svc.RegisterResident(c.Request.Context(), fullName, phone, email, iin, password)
	if err != nil {
		log.Printf("[Register Error]: %v", err)

		// Map raw errors to clean, user-facing UI messages
		errMsg := "Ошибка при регистрации. Проверьте введенные данные."
		if errors.Is(err, service.ErrAlreadyExists) || strings.Contains(err.Error(), "email") {
			errMsg = "Пользователь с таким Email уже зарегистрирован."
		} else if strings.Contains(err.Error(), "users_pkey") || strings.Contains(err.Error(), "duplicate key") {
			errMsg = "Ошибка базы данных: пользователь уже существует."
		}

		// Use http.StatusOK (200) so HTMX updates the form with the error message
		c.HTML(http.StatusOK, "client/register.html", gin.H{
			"Title": "Регистрация жителя",
			"Error": errMsg,
		})
		return
	}

	jks, _ := h.svc.EgovStub(c.Request.Context(), iin)
	var jkID string
	if len(jks) > 0 {
		jkID = jks[0].ID
	}

	token, _, err := h.tokens.Generate(u.ID, string(u.Role), jkID, u.Email)
	if err == nil {
		setCookie(c, h.cfg, token)
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/client/chessboard")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusSeeOther, "/client/chessboard")
}

func (h *Handler) ClientLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "client/login.html", nil)
}

func (h *Handler) ClientLogin(c *gin.Context) {
	login := c.PostForm("login")
	password := c.PostForm("password")

	u, err := h.svc.Login(c.Request.Context(), login, password)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "client/login.html", gin.H{"Error": "Неверный логин или пароль"})
		return
	}

	var jkID string
	if u.IIN != nil {
		jks, _ := h.svc.EgovStub(c.Request.Context(), *u.IIN)
		if len(jks) > 0 {
			jkID = jks[0].ID
		}
	}

	token, _, err := h.tokens.Generate(u.ID, string(u.Role), jkID, u.Email)

	if err == nil {
		setCookie(c, h.cfg, token)
	}

	c.Redirect(http.StatusSeeOther, "/client/chessboard")
}

func (h *Handler) Chessboard(c *gin.Context) {
	claims := getClaimsFromCtx(c)
	jkID := ""
	if claims != nil {
		jkID = claims.JKID
	}

	if jkID == "" {
		jks, err := h.svc.Repo().ListJKs(c.Request.Context())
		if err != nil {
			log.Printf("[Chessboard Error] ListJKs DB Error: %v", err)
		} else if len(jks) > 0 {
			jkID = jks[0].ID
		}
	}

	// 1. Try EgovStub if claims didn't supply a JKID
	if jkID == "" {
		jks, _ := h.svc.EgovStub(c.Request.Context(), "")
		if len(jks) > 0 {
			jkID = jks[0].ID
		}
	}

	// 2. Fallback: Query all JKs directly from the DB if EgovStub returns empty
	if jkID == "" {
		jks, err := h.svc.Repo().ListJKs(c.Request.Context())
		if err == nil && len(jks) > 0 {
			jkID = jks[0].ID
		}
	}

	// 3. Fail gracefully only if the database has 0 JKs configured
	if jkID == "" {
		log.Printf("[Chessboard Error] No JKs found in database")
		c.HTML(http.StatusOK, "client/chessboard.html", gin.H{
			"Error": "В системе нет доступных ЖК. Добавьте ЖК в базу данных.",
		})
		return
	}

	jk, _ := h.svc.Repo().GetJKByID(c.Request.Context(), jkID)
	jkName := "ЖК"
	if jk != nil {
		jkName = jk.Name
	}

	views, err := h.svc.StorageViews(c.Request.Context(), jkID)
	if err != nil {
		log.Printf("[Chessboard Error] StorageViews failed: %v", err)
		c.HTML(http.StatusOK, "client/chessboard.html", gin.H{
			"Error": err.Error(),
		})
		return
	}

	tariff, _ := h.svc.Repo().GetTariff(c.Request.Context(), jkID)

	c.HTML(http.StatusOK, "client/chessboard.html", gin.H{
		"User":     getUserFromCtx(c),
		"Storages": views,
		"Tariff":   tariff,
		"JKID":     jkID,
		"JKName":   jkName,
	})
}

func (h *Handler) Occupy(c *gin.Context) {
	claims := getClaimsFromCtx(c)
	if claims == nil {
		c.Redirect(http.StatusSeeOther, "/client/login")
		return
	}

	storageID := c.Param("id")
	if storageID == "" {
		storageID = c.PostForm("storage_id")
	}
	months, _ := strconv.Atoi(c.PostForm("months"))

	rt, quote, err := h.svc.LockStorage(c.Request.Context(), storageID, claims.UserID, months)
	if err != nil {
		c.HTML(http.StatusBadRequest, "client/catalog.html", gin.H{"Error": err.Error()})
		return
	}

	c.HTML(http.StatusOK, "partials/payment_modal.html", gin.H{
		"Rental": rt,
		"Quote":  quote,
	})
}

func (h *Handler) Pay(c *gin.Context) {
	claims := getClaimsFromCtx(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	rentalID := c.Param("id")
	if rentalID == "" {
		rentalID = c.PostForm("rental_id")
	}
	_, err := h.svc.ConfirmPayment(c.Request.Context(), rentalID, claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("HX-Redirect", "/client/chessboard")
	c.Status(http.StatusOK)
}

// ---------- Helper functions ----------

func getUserFromCtx(c *gin.Context) *auth.Claims {
	return getClaimsFromCtx(c)
}

func getClaimsFromCtx(c *gin.Context) *auth.Claims {
	if val, ok := c.Get("claims"); ok {
		if claims, ok := val.(*auth.Claims); ok {
			return claims
		}
	}
	return nil
}

func setCookie(c *gin.Context, cfg interface{}, token string) {
	c.SetCookie("access_token", token, 86400, "/", "", false, true)
}

func clearCookie(c *gin.Context, cfg interface{}) {
	c.SetCookie("access_token", "", -1, "/", "", false, true)
}
