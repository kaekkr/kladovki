package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/auth"
	"github.com/kaekkr/kladovki/internal/config"
	"github.com/kaekkr/kladovki/internal/database"
	"github.com/kaekkr/kladovki/internal/handlers"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/repository"
	"github.com/kaekkr/kladovki/internal/service"
)

func main() {
	cfg := config.Load()

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	repo := repository.New(db)
	svc := service.New(repo)
	if err := svc.EnsureSeed(context.Background()); err != nil {
		log.Println("seed info:", err)
	}

	tokens := auth.NewTokenService(cfg.JWTSecret, cfg.JWTAccessTTL)
	mw := middleware.NewAuth(tokens, cfg)

	r := gin.Default()
	r.SetHTMLTemplate(loadTemplates())
	r.Static("/static", "./static")

	h := handlers.New(svc, tokens, cfg, mw)
	h.Register(r)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Запуск сервера в отдельной горутине для Graceful Shutdown
	go func() {
		log.Printf("Server listening on http://localhost:%s (%s mode)", cfg.Port, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %s\n", err)
		}
	}()

	// Ожидание сигнала завершения (Ctrl+C, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func loadTemplates() *template.Template {
	// Создаем FuncMap для форматирования сумм int64 и дат в шаблонах HTMX/HTML
	funcMap := template.FuncMap{
		"formatMoney": func(amount int64) string {
			return fmt.Sprintf("%d ₸", amount)
		},
		"formatDate": func(t *time.Time) string {
			if t == nil {
				return "-"
			}
			return t.Format("02.01.2006")
		},
	}

	t := template.New("").Funcs(funcMap)
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
			if _, err := t.New(name).Parse(string(content)); err != nil {
				log.Printf("parse %s: %v", name, err)
			}
		}
	}
	return t
}
