package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/auth"
)

func (h *Handler) ClientRegisterPage(c *gin.Context) {
	jks, _ := h.svc.ListJKs(c.Request.Context())
	selectedJK := c.Query("jk") // Captures ?jk=jk_id from QR code/invite link

	c.HTML(http.StatusOK, "pages/client/register", gin.H{
		"JKs":        jks,
		"SelectedJK": selectedJK,
	})
}

func (h *Handler) ClientRegister(c *gin.Context) {
	fullName := c.PostForm("full_name")
	phone := c.PostForm("phone")
	email := c.PostForm("email")
	iin := c.PostForm("iin")
	password := c.PostForm("password")
	jkID := c.PostForm("jk_id")

	u, err := h.svc.RegisterResident(c.Request.Context(), fullName, phone, email, iin, password, jkID)
	if err != nil {
		log.Printf("[Register Error]: %v", err)

		jks, _ := h.svc.ListJKs(c.Request.Context())

		errMsg := "Ошибка при регистрации. Проверьте введенные данные."
		if strings.Contains(err.Error(), "email") {
			errMsg = "Пользователь с таким Email уже зарегистрирован."
		} else if strings.Contains(err.Error(), "phone") {
			errMsg = "Пользователь с таким номером телефона уже зарегистрирован."
		} else if strings.Contains(err.Error(), "IIN") || strings.Contains(err.Error(), "iin") {
			errMsg = "Пользователь с таким ИИН уже зарегистрирован."
		}

		c.HTML(http.StatusBadRequest, "pages/client/register", gin.H{
			"Title":      "Регистрация жителя",
			"Error":      errMsg,
			"FullName":   fullName,
			"Phone":      phone,
			"Email":      email,
			"IIN":        iin,
			"JKs":        jks,
			"SelectedJK": jkID,
		})
		return
	}

	// Resolve active JK: explicit dropdown selection -> eGov stub match
	finalJKID := jkID
	if finalJKID == "" && u.IIN != nil {
		if egovJKs, _ := h.svc.EgovStub(c.Request.Context(), *u.IIN); len(egovJKs) > 0 {
			finalJKID = egovJKs[0].ID
		}
	}

	// Attach resolved JKID to user model for token issuance
	if finalJKID != "" {
		u.JKID = &finalJKID
	}
	if _, err := h.issueToken(c, u); err != nil {
		log.Printf("[Register Token Error]: %v", err)
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/client/chessboard")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusSeeOther, "/client/chessboard")
}

func (h *Handler) ClientLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/client/login", nil)
}

func (h *Handler) ClientLogin(c *gin.Context) {
	login := c.PostForm("login")
	password := c.PostForm("password")

	u, err := h.svc.Login(c.Request.Context(), login, password)
	if err != nil {
		log.Printf("[Login Error]: %v", err)
		c.HTML(http.StatusOK, "pages/client/login", gin.H{
			"Title": "Вход для жителей",
			"Error": "Неверный логин или пароль",
		})
		return
	}

	_, err = h.issueToken(c, u)
	if err != nil {
		log.Printf("[Login Error] Token generation failed: %v", err)
		c.HTML(http.StatusOK, "pages/client/login", gin.H{
			"Title": "Вход для жителей",
			"Error": "Ошибка авторизации. Попробуйте позже.",
		})
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/client/chessboard")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusSeeOther, "/client/chessboard")
}

/* TODO: Uncomment these interactive client handlers as we build their views
func (h *Handler) Chessboard(c *gin.Context) { ... }
func (h *Handler) Occupy(c *gin.Context) { ... }
func (h *Handler) StoragePreview(c *gin.Context) { ... }
func (h *Handler) Pay(c *gin.Context) { ... }
*/

// ---------- Context Helpers ----------

func getClaimsFromCtx(c *gin.Context) *auth.Claims {
	if val, ok := c.Get("claims"); ok {
		if claims, ok := val.(*auth.Claims); ok {
			return claims
		}
	}
	return nil
}
