package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.SetHTMLTemplate(loadTemplates())
	r.Static("/static", "./static")

	// ========== PUBLIC ==========
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "layouts/home.html", gin.H{"Title": "Кладовки ЖК"})
	})

	// ========== CLIENT ==========
	r.GET("/client/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "client/register.html", gin.H{"Title": "Регистрация жителя"})
	})

	r.POST("/client/register", func(c *gin.Context) {
		// Stub eGov: всегда ЖК Сыганак → сразу схема
		c.HTML(http.StatusOK, "client/chessboard.html", gin.H{
			"Title":    "Кладовки — ЖК Сыганак",
			"JKName":   "ЖК Сыганак",
			"Storages": mockStorages(),
		})
	})

	r.GET("/client/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "client/login.html", gin.H{"Title": "Вход для жителей"})
	})

	r.POST("/client/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "client/chessboard.html", gin.H{
			"Title":    "Кладовки — ЖК Сыганак",
			"JKName":   "ЖК Сыганак",
			"Storages": mockStorages(),
		})
	})

	r.POST("/client/occupy/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.HTML(http.StatusOK, "partials/occupy_modal.html", gin.H{
			"StorageID": id,
			"ExpiresAt": time.Now().Add(2 * time.Minute).Format("15:04:05"),
		})
	})

	r.POST("/client/pay/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.HTML(http.StatusOK, "partials/payment_success.html", gin.H{"StorageID": id})
	})

	// ========== ADMIN ==========
	r.GET("/admin/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin/register.html", gin.H{"Title": "Регистрация ЖК (платно)"})
	})

	r.POST("/admin/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin/storages_setup.html", gin.H{
			"Title":  "Настройка кладовок",
			"JKName": c.PostForm("jk_name"),
		})
	})

	r.GET("/admin/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin/login.html", gin.H{"Title": "Вход для управляющей компании"})
	})

	r.POST("/admin/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin/dashboard.html", gin.H{
			"Title":    "Панель ЖК Сыганак",
			"JKName":   "ЖК Сыганак",
			"Storages": mockStoragesAdmin(),
			"Tariff":   15000,
		})
	})

	r.GET("/admin/dashboard", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin/dashboard.html", gin.H{
			"Title":    "Панель ЖК Сыганак",
			"JKName":   "ЖК Сыганак",
			"Storages": mockStoragesAdmin(),
			"Tariff":   15000,
		})
	})

	r.POST("/admin/storages/add-row", func(c *gin.Context) {
		c.HTML(http.StatusOK, "partials/storage_row.html", gin.H{
			"Index": time.Now().UnixNano() % 10000,
		})
	})

	r.POST("/admin/storages/save", func(c *gin.Context) {
		c.HTML(http.StatusOK, "partials/toast.html", gin.H{
			"Message": "Кладовки сохранены (stub)",
			"Type":    "success",
		})
	})

	r.GET("/admin/tariff", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin/tariff.html", gin.H{
			"Title":         "Тарифы",
			"JKName":        "ЖК Сыганак",
			"CurrentTariff": 15000,
			"History": []map[string]interface{}{
				{"Date": time.Now().AddDate(0, -2, 0).Format("02.01.2006 15:04"), "Amount": 12000},
				{"Date": time.Now().AddDate(0, -1, 0).Format("02.01.2006 15:04"), "Amount": 14000},
				{"Date": time.Now().Format("02.01.2006 15:04"), "Amount": 15000},
			},
		})
	})

	r.POST("/admin/tariff", func(c *gin.Context) {
		c.HTML(http.StatusOK, "partials/toast.html", gin.H{
			"Message": "Тариф обновлён. Перерасчёт будет выполнен (stub)",
			"Type":    "success",
		})
	})

	r.GET("/admin/reports", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin/reports.html", gin.H{
			"Title":  "Отчёты",
			"JKName": "ЖК Сыганак",
			"Debtors": []map[string]interface{}{
				{"Storage": "A-12", "Owner": "Иванов И.И.", "Debt": 15000, "Days": 12},
				{"Storage": "B-05", "Owner": "Петрова А.С.", "Debt": 30000, "Days": 45},
			},
			"Free":     8,
			"Occupied": 24,
			"Total":    32,
		})
	})

	log.Println("Server starting on :8080")
	r.Run(":8080")
}

func loadTemplates() *template.Template {
	t := template.New("").Funcs(template.FuncMap{
		"formatDate": func(tt time.Time) string {
			return tt.Format("02.01.2006 15:04")
		},
	})

	patterns := []string{
		"templates/layouts/*.html",
		"templates/client/*.html",
		"templates/admin/*.html",
		"templates/partials/*.html",
	}

	for _, p := range patterns {
		files, err := filepath.Glob(p)
		if err != nil {
			continue
		}
		for _, f := range files {
			name := strings.TrimPrefix(f, "templates/")
			name = filepath.ToSlash(name)
			content, err := os.ReadFile(f)
			if err != nil {
				log.Printf("read %s: %v", f, err)
				continue
			}
			_, err = t.New(name).Parse(string(content))
			if err != nil {
				log.Printf("parse %s: %v", name, err)
			}
		}
	}
	return t
}

func mockStorages() []map[string]interface{} {
	return []map[string]interface{}{
		{"ID": "A-01", "Number": "A-01", "Area": 4.5, "Floor": -1, "Entrance": 1, "Status": "free"},
		{"ID": "A-02", "Number": "A-02", "Area": 5.0, "Floor": -1, "Entrance": 1, "Status": "occupied"},
		{"ID": "A-03", "Number": "A-03", "Area": 3.8, "Floor": -1, "Entrance": 1, "Status": "free"},
		{"ID": "A-04", "Number": "A-04", "Area": 6.2, "Floor": -1, "Entrance": 1, "Status": "free"},
		{"ID": "B-01", "Number": "B-01", "Area": 4.0, "Floor": -1, "Entrance": 2, "Status": "occupied"},
		{"ID": "B-02", "Number": "B-02", "Area": 5.5, "Floor": -1, "Entrance": 2, "Status": "free"},
		{"ID": "B-03", "Number": "B-03", "Area": 4.8, "Floor": -1, "Entrance": 2, "Status": "free"},
		{"ID": "C-01", "Number": "C-01", "Area": 7.0, "Floor": -1, "Entrance": 3, "Status": "occupied"},
		{"ID": "C-02", "Number": "C-02", "Area": 3.5, "Floor": -1, "Entrance": 3, "Status": "free"},
		{"ID": "C-03", "Number": "C-03", "Area": 5.2, "Floor": -1, "Entrance": 3, "Status": "free"},
		{"ID": "D-01", "Number": "D-01", "Area": 4.2, "Floor": -1, "Entrance": 4, "Status": "occupied"},
		{"ID": "D-02", "Number": "D-02", "Area": 6.0, "Floor": -1, "Entrance": 4, "Status": "free"},
	}
}

func mockStoragesAdmin() []map[string]interface{} {
	return []map[string]interface{}{
		{"ID": "A-01", "Number": "A-01", "Area": 4.5, "Floor": -1, "Entrance": 1, "Status": "free", "Period": "-", "Paid": 0, "Debt": 0},
		{"ID": "A-02", "Number": "A-02", "Area": 5.0, "Floor": -1, "Entrance": 1, "Status": "occupied", "Period": "01.06.2026 — 31.08.2026", "Paid": 45000, "Debt": 0},
		{"ID": "A-03", "Number": "A-03", "Area": 3.8, "Floor": -1, "Entrance": 1, "Status": "free", "Period": "-", "Paid": 0, "Debt": 0},
		{"ID": "A-04", "Number": "A-04", "Area": 6.2, "Floor": -1, "Entrance": 1, "Status": "free", "Period": "-", "Paid": 0, "Debt": 0},
		{"ID": "B-01", "Number": "B-01", "Area": 4.0, "Floor": -1, "Entrance": 2, "Status": "occupied", "Period": "15.05.2026 — 14.07.2026", "Paid": 30000, "Debt": 15000},
		{"ID": "B-02", "Number": "B-02", "Area": 5.5, "Floor": -1, "Entrance": 2, "Status": "free", "Period": "-", "Paid": 0, "Debt": 0},
		{"ID": "B-03", "Number": "B-03", "Area": 4.8, "Floor": -1, "Entrance": 2, "Status": "free", "Period": "-", "Paid": 0, "Debt": 0},
		{"ID": "C-01", "Number": "C-01", "Area": 7.0, "Floor": -1, "Entrance": 3, "Status": "occupied", "Period": "01.07.2026 — 30.09.2026", "Paid": 45000, "Debt": 0},
		{"ID": "C-02", "Number": "C-02", "Area": 3.5, "Floor": -1, "Entrance": 3, "Status": "free", "Period": "-", "Paid": 0, "Debt": 0},
		{"ID": "C-03", "Number": "C-03", "Area": 5.2, "Floor": -1, "Entrance": 3, "Status": "free", "Period": "-", "Paid": 0, "Debt": 0},
		{"ID": "D-01", "Number": "D-01", "Area": 4.2, "Floor": -1, "Entrance": 4, "Status": "occupied", "Period": "10.04.2026 — 09.06.2026", "Paid": 30000, "Debt": 30000},
		{"ID": "D-02", "Number": "D-02", "Area": 6.0, "Floor": -1, "Entrance": 4, "Status": "free", "Period": "-", "Paid": 0, "Debt": 0},
	}
}
