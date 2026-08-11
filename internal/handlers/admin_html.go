package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/models"
)

func (h *Handler) AdminRegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "admin/register.html", gin.H{"Title": "Регистрация ЖК (платно)"})
}

func (h *Handler) AdminRegister(c *gin.Context) {
	password := c.PostForm("password")
	if password == "" || len(password) < 6 {
		c.HTML(http.StatusOK, "admin/register.html", gin.H{"Title": "Регистрация ЖК", "Error": "Пароль минимум 6 символов"})
		return
	}

	jk, u, err := h.svc.RegisterJK(
		c.Request.Context(),
		c.PostForm("jk_name"), c.PostForm("bin"), c.PostForm("contact"),
		c.PostForm("phone"), c.PostForm("email"), password,
	)
	if err != nil {
		c.HTML(http.StatusOK, "admin/register.html", gin.H{"Title": "Регистрация ЖК", "Error": err.Error()})
		return
	}

	u.JKID = &jk.ID
	if _, err := h.issueToken(c, u); err != nil {
		c.HTML(http.StatusOK, "admin/register.html", gin.H{"Title": "Регистрация ЖК", "Error": "Ошибка создания сессии"})
		return
	}

	c.HTML(http.StatusOK, "admin/storages_setup.html", gin.H{
		"Title": "Настройка кладовок", "JKName": jk.Name, "JKID": jk.ID,
	})
}

func (h *Handler) AdminLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "admin/login.html", gin.H{"Title": "Вход для УК"})
}

func (h *Handler) AdminLogin(c *gin.Context) {
	u, err := h.svc.Login(c.Request.Context(), c.PostForm("login"), c.PostForm("password"))
	if err != nil || u.Role != models.RoleAdmin {
		c.HTML(http.StatusOK, "admin/login.html", gin.H{"Title": "Вход", "Error": "Неверный логин или пароль"})
		return
	}

	var jkID string
	if u.JKID != nil {
		jkID = *u.JKID
	}

	if jkID == "" {
		jks, _ := h.svc.Repo().ListJKs(c.Request.Context())
		for _, j := range jks {
			if j.OwnerID == u.ID {
				jkID = j.ID
				_ = h.svc.Repo().UpdateUserJKID(c.Request.Context(), u.ID, jkID)
				u.JKID = &jkID
				break
			}
		}
	}

	if _, err := h.issueToken(c, u); err != nil {
		c.HTML(http.StatusOK, "admin/login.html", gin.H{"Title": "Вход", "Error": "Ошибка сессии"})
		return
	}

	h.renderDashboard(c, jkID)
}

func (h *Handler) Dashboard(c *gin.Context) {
	jkID := middleware.JKID(c)
	if jkID == "" {
		c.Redirect(http.StatusFound, "/admin/login")
		return
	}
	h.renderDashboard(c, jkID)
}

func (h *Handler) renderDashboard(c *gin.Context, jkID string) {
	jk, _ := h.svc.Repo().GetJKByID(c.Request.Context(), jkID)
	name := "ЖК"
	if jk != nil {
		name = jk.Name
	}

	views, _ := h.svc.StorageViews(c.Request.Context(), jkID)
	tariff, _ := h.svc.Repo().GetTariff(c.Request.Context(), jkID)

	c.HTML(http.StatusOK, "admin/dashboard.html", gin.H{
		"Title": "Панель " + name, "JKName": name, "Storages": views, "Tariff": tariff,
	})
}

func (h *Handler) AddStorageRow(c *gin.Context) {
	c.HTML(http.StatusOK, "partials/storage_row.html", gin.H{"Index": time.Now().UnixNano() % 10000})
}

func (h *Handler) SaveStorages(c *gin.Context) {
	jkID := middleware.JKID(c)
	if jkID == "" {
		h.renderToastError(c, "Нет сессии")
		return
	}

	numbers := c.PostFormArray("number[]")
	areas := c.PostFormArray("area[]")
	floors := c.PostFormArray("floor[]")
	entrances := c.PostFormArray("entrance[]")

	var items []models.Storage
	for i := range numbers {
		if numbers[i] == "" {
			continue
		}
		area, _ := strconv.ParseFloat(areas[i], 64)
		floor, _ := strconv.Atoi(floors[i])
		ent, _ := strconv.Atoi(entrances[i])
		items = append(items, models.Storage{
			Number: numbers[i], Area: area, Floor: floor, Entrance: ent,
		})
	}

	if err := h.svc.AddStorages(c.Request.Context(), jkID, items); err != nil {
		h.renderToastError(c, err.Error())
		return
	}

	h.renderToastSuccess(c, "Кладовки сохранены")
}

func (h *Handler) TariffPage(c *gin.Context) {
	jkID := middleware.JKID(c)
	jk, _ := h.svc.Repo().GetJKByID(c.Request.Context(), jkID)
	name := "ЖК"
	if jk != nil {
		name = jk.Name
	}

	tariff, _ := h.svc.Repo().GetTariff(c.Request.Context(), jkID)
	hist, _ := h.svc.Repo().TariffHistory(c.Request.Context(), jkID)

	var history []map[string]any
	for _, hitem := range hist {
		history = append(history, map[string]any{
			"Date":   hitem.ChangedAt.Format("02.01.2006 15:04"),
			"Amount": hitem.Amount,
		})
	}

	c.HTML(http.StatusOK, "admin/tariff.html", gin.H{
		"Title": "Тарифы", "JKName": name, "CurrentTariff": tariff, "History": history,
	})
}

func (h *Handler) UpdateTariff(c *gin.Context) {
	jkID := middleware.JKID(c)

	amount, err := strconv.ParseInt(c.PostForm("tariff"), 10, 64)
	if err != nil || amount < 0 {
		h.renderToastError(c, "Некорректная сумма")
		return
	}

	old, rentals, err := h.svc.ChangeTariff(c.Request.Context(), jkID, amount)
	if err != nil {
		h.renderToastError(c, err.Error())
		return
	}

	msg := "Тариф обновлён"
	if amount < old && len(rentals) > 0 {
		msg += ". Есть предоплатившие — доступен перерасчёт"
	}

	h.renderToastSuccess(c, msg)
}

func (h *Handler) Reports(c *gin.Context) {
	jkID := middleware.JKID(c)
	jk, _ := h.svc.Repo().GetJKByID(c.Request.Context(), jkID)
	name := "ЖК"
	if jk != nil {
		name = jk.Name
	}

	views, _ := h.svc.StorageViews(c.Request.Context(), jkID)
	free, occupied := 0, 0
	var debtors []map[string]any

	for _, v := range views {
		if v.Status == models.StatusFree {
			free++
		} else {
			occupied++
		}
		if v.Debt > 0 {
			debtors = append(debtors, map[string]any{
				"Storage": v.Number, "Owner": v.Owner, "Debt": v.Debt, "Days": v.DaysOver,
			})
		}
	}

	c.HTML(http.StatusOK, "admin/reports.html", gin.H{
		"Title": "Отчёты", "JKName": name,
		"Free": free, "Occupied": occupied, "Total": free + occupied, "Debtors": debtors,
	})
}
