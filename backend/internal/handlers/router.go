package handlers

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) Register(r *gin.Engine) {
	r.Use(h.mw.Optional())

	api := r.Group(h.cfg.APIPrefix)
	{
		api.POST("/lead", h.CreateLead)
		api.GET("/jks/:id", h.mw.Require(), h.GetJKByID)

		// === Auth ===
		auth := api.Group("/auth")
		{
			auth.POST("/login", h.Login)
			auth.POST("/logout", h.Logout)
			auth.GET("/me", h.mw.Require(), h.Me)
		}

		dashboard := api.Group("/dashboard")
		dashboard.Use(h.mw.Require())
		{
			dashboard.GET("/stats", h.GetDashboardStats)
			dashboard.GET("/revenue", h.GetDashboardRevenue)
			dashboard.GET("/attention", h.GetDashboardAttention)
			dashboard.GET("/activity", h.GetDashboardActivity)
			dashboard.GET("/occupancy", h.GetDashboardOccupancy)
		}

		// === Storages (Flat Routes) ===
		storages := api.Group("/storages")
		storages.Use(h.mw.Require())
		{
			storages.GET("/jk/:jk_id", h.ListStoragesByJK)
			storages.POST("", h.CreateStorage)
			storages.GET("/:id", h.GetStorageByID)
			storages.PUT("/:id", h.UpdateStorage)
			storages.DELETE("/:id", h.DeleteStorage)
		}

		// === Rentals ===
		rentals := api.Group("/rentals")
		rentals.Use(h.mw.Require())
		{
			rentals.GET("/jk/:jk_id", h.ListRentalsByJK)
			rentals.GET("/storage/:storage_id", h.GetActiveRentalByStorage)
			rentals.PATCH("/:id/cancel", h.CancelRental)
			rentals.PATCH("/:id/force-release", h.ForceReleaseLocked)
			rentals.GET("/:id", h.GetRentalByID)
		}
	}
}
