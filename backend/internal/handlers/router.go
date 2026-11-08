package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/middleware"
)

// Access rules:
//   - admin  = OSI administrator, may manage only their own JK (token jk_id).
//   - resident may read their own JK's storages and only their own rentals/payments.
//
// JK ownership of path/body IDs is checked inside the handlers.
func (h *Handler) Register(r *gin.Engine) {
	r.Use(h.mw.Optional())

	auth := h.mw.Require()
	admin := middleware.RequireAdmin()

	api := r.Group(h.cfg.APIPrefix)
	{
		api.POST("/lead", h.CreateLead)
		api.GET("/jks/:id", auth, h.GetJKByID)
		api.GET("/jks/:id/tariff", auth, h.GetTariff)
		api.PUT("/jks/:id/tariff", auth, admin, h.SetTariff)
		api.GET("/jks/:id/tariff/preview", auth, admin, h.GetTariffPreview)

		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", h.RegisterResident)
			authGroup.POST("/login", h.Login)
			authGroup.POST("/logout", h.Logout)
			authGroup.GET("/me", auth, h.Me)
		}

		dashboard := api.Group("/dashboard", auth, admin)
		{
			dashboard.GET("/stats", h.GetDashboardStats)
			dashboard.GET("/revenue", h.GetDashboardRevenue)
			dashboard.GET("/attention", h.GetDashboardAttention)
			dashboard.GET("/activity", h.GetDashboardActivity)
			dashboard.GET("/occupancy", h.GetDashboardOccupancy)
		}

		storages := api.Group("/storages", auth)
		{
			storages.GET("/jk/:jk_id", h.ListStoragesByJK)
			storages.GET("/:id", h.GetStorageByID)
			storages.POST("", admin, h.CreateStorage)
			storages.PUT("/:id", admin, h.UpdateStorage)
			storages.DELETE("/:id", admin, h.DeleteStorage)
			storages.PATCH("/:id/transfer", h.SetStorageTransfer)
		}

		rentals := api.Group("/rentals", auth)
		{
			rentals.GET("/my", h.ListMyRentals)
			rentals.GET("/jk/:jk_id", admin, h.ListRentalsByJK)
			rentals.GET("/storage/:storage_id", admin, h.GetActiveRentalByStorage)

			rentals.POST("/lock", h.LockStorage)
			// Activates a rental without payment — local development only.
			if h.cfg.Env == "development" {
				rentals.POST("/confirm-payment", h.ConfirmPayment)
			}

			rentals.PATCH("/:id/cancel", admin, h.CancelRental)
			rentals.PATCH("/:id/force-release", admin, h.ForceReleaseLocked)
			rentals.PATCH("/:id/release-overdue", admin, h.ReleaseOverdueRental)

			rentals.POST("/:id/terminate", h.RequestEarlyTermination)
			rentals.GET("/:id", h.GetRentalByID)
		}

		requests := api.Group("/requests", auth)
		{
			requests.GET("/my", h.ListMyRequests)
			requests.POST("/:id/choice", h.ChooseTariffOption)
			requests.GET("/jk/:jk_id", admin, h.ListRequestsByJK)
			requests.GET("/open-count", admin, h.CountOpenRequests)
			requests.PATCH("/:id/resolve", admin, h.ResolveRequest)
		}

		notifications := api.Group("/notifications", auth)
		{
			notifications.GET("", h.ListNotifications)
			notifications.GET("/unread-count", h.UnreadNotifications)
			notifications.POST("/read-all", h.MarkAllNotificationsRead)
			notifications.POST("/:id/read", h.MarkNotificationRead)
		}

		me := api.Group("/me", auth)
		{
			me.GET("/channels", h.MyChannels)
			me.POST("/telegram/link", h.TelegramLink)
			me.DELETE("/telegram", h.TelegramUnlink)
		}

		// Telegram bot updates; authenticated by the secret-token header.
		api.POST("/telegram/webhook", h.TelegramWebhook)

		payments := api.Group("/payments")
		{
			// Called by ApiPay; authenticated by HMAC signature, not by user token.
			payments.POST("/webhook", h.PaymentWebhook)
			payments.POST("", auth, h.CreatePayment)
			payments.GET("/my", auth, h.ListMyPayments)
			payments.GET("/jk/:jk_id", auth, admin, h.ListPaymentsByJK)
		}
	}
}
