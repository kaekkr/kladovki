package handlers

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/kaekkr/kladovki/internal/middleware"
)

func (h *Handler) ListNotifications(c *gin.Context) {
	list, err := h.svc.ListNotifications(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить уведомления"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) UnreadNotifications(c *gin.Context) {
	n, err := h.svc.UnreadNotifications(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось загрузить уведомления"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unread": n})
}

func (h *Handler) MarkNotificationRead(c *gin.Context) {
	h.markRead(c, c.Param("id"))
}

func (h *Handler) MarkAllNotificationsRead(c *gin.Context) {
	h.markRead(c, "")
}

func (h *Handler) markRead(c *gin.Context, id string) {
	if err := h.svc.MarkNotificationsRead(c.Request.Context(), middleware.UserID(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить уведомления"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *Handler) MyChannels(c *gin.Context) {
	st, err := h.svc.UserChannels(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить настройки уведомлений"})
		return
	}
	c.JSON(http.StatusOK, st)
}

func (h *Handler) TelegramLink(c *gin.Context) {
	url, err := h.svc.TelegramLinkURL(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Telegram-уведомления пока не настроены"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *Handler) TelegramUnlink(c *gin.Context) {
	if err := h.svc.UnlinkTelegram(c.Request.Context(), middleware.UserID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось отключить Telegram"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

type telegramUpdate struct {
	Message *struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

// TelegramWebhook receives bot updates. Telegram sends the secret configured in
// setWebhook as X-Telegram-Bot-Api-Secret-Token.
func (h *Handler) TelegramWebhook(c *gin.Context) {
	secret := h.svc.NotifyConfig().TelegramWebhookSecret
	got := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
	if secret == "" || subtle.ConstantTimeCompare([]byte(got), []byte(secret)) != 1 {
		c.Status(http.StatusUnauthorized)
		return
	}
	var u telegramUpdate
	if err := c.ShouldBindJSON(&u); err != nil || u.Message == nil {
		c.Status(http.StatusOK)
		return
	}
	text := strings.TrimSpace(u.Message.Text)
	if text == "/start" || strings.HasPrefix(text, "/start ") {
		token := strings.TrimSpace(strings.TrimPrefix(text, "/start"))
		h.svc.HandleTelegramStart(c.Request.Context(), fmt.Sprint(u.Message.Chat.ID), token)
	}
	c.Status(http.StatusOK)
}
