package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/models"
)

func (h *Handler) Register(r *gin.Engine) {
	r.Use(h.mw.Optional())

	r.GET("/", h.Home)
	r.GET("/logout", h.Logout)
	r.POST("/logout", h.Logout)

	// --- Client HTML Routes ---
	r.GET("/client/register", h.ClientRegisterPage)
	r.POST("/client/register", h.ClientRegister)
	r.GET("/client/login", h.ClientLoginPage)
	r.POST("/client/login", h.ClientLogin)

	client := r.Group("/client")
	client.Use(h.mw.Require())
	{
		client.GET("/chessboard", h.Chessboard)
		client.POST("/occupy/:id", h.Occupy)
		client.POST("/pay/:id", h.Pay)
	}

	// --- Admin HTML Routes ---
	r.GET("/admin/register", h.AdminRegisterPage)
	r.POST("/admin/register", h.AdminRegister)
	r.GET("/admin/login", h.AdminLoginPage)
	r.POST("/admin/login", h.AdminLogin)

	admin := r.Group("/admin")
	admin.Use(h.mw.RequireRole(string(models.RoleAdmin)))
	{
		admin.GET("/dashboard", h.Dashboard)
		admin.POST("/storages/add-row", h.AddStorageRow)
		admin.POST("/storages/save", h.SaveStorages)
		admin.GET("/tariff", h.TariffPage)
		admin.POST("/tariff", h.UpdateTariff)
		admin.GET("/reports", h.Reports)
	}

	// --- JSON API Routes ---
	api := r.Group("/api")
	{
		api.POST("/auth/register/resident", h.APIRegisterResident)
		api.POST("/auth/register/jk", h.APIRegisterJK)
		api.POST("/auth/login", h.APILogin)
		api.GET("/auth/me", h.mw.Require(), h.APIMe)
	}
}

func (h *Handler) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "layouts/home.html", gin.H{"Title": "Кладовки ЖК"})
}

func (h *Handler) Logout(c *gin.Context) {
	h.clearAuthCookie(c)
	c.Redirect(http.StatusFound, "/")
}
