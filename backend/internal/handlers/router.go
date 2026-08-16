package handlers

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) Register(r *gin.Engine) {
	r.Use(h.mw.Optional())

	api := r.Group(h.cfg.APIPrefix)
	{
		// === Auth (shared) ===
		auth := api.Group("/auth")
		{
			auth.POST("/login", h.Login) // Handles both admin & resident authentication
			// auth.POST("/register", h.Register)    // Resident registration (accepts JK ID from QR code)
			auth.POST("/logout", h.Logout)
			auth.GET("/me", h.mw.Require(), h.Me)
		}

		//
		// // === Client area ===
		// client := api.Group("/client")
		// client.Use(h.mw.Require()) // only authenticated residents
		// {
		// 	client.GET("/dashboard", h.ClientDashboard)
		// 	client.GET("/storages", h.ClientStorages)
		// 	// ...
		// }
		//
		// // === Admin area ===
		// admin := api.Group("/admin")
		// admin.Use(h.mw.RequireRole("admin"))
		// {
		// 	admin.GET("/dashboard", h.AdminDashboard)
		// 	admin.GET("/storages", h.AdminStorages)
		// 	admin.POST("/storages", h.CreateStorage)
		// 	// ...
		// }
	}
}
