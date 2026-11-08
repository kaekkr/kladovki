package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/kaekkr/kladovki/internal/config"
	"github.com/kaekkr/kladovki/internal/database"
	"github.com/kaekkr/kladovki/internal/handlers"
	"github.com/kaekkr/kladovki/internal/middleware"
	"github.com/kaekkr/kladovki/internal/notify"
	"github.com/kaekkr/kladovki/internal/repository"
	"github.com/kaekkr/kladovki/internal/service"
)

func main() {
	cfg := config.Load()

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 1. Database & Repositories
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	repo := repository.New(db)
	svc := service.New(repo)

	notifyCfg := notify.ConfigFromEnv()
	svc.SetNotifier(notifyCfg)
	for _, ch := range notifyCfg.Channels() {
		log.Printf("notifications: %s channel enabled", ch.Name())
	}

	// 2. Auth & Middleware
	tokens := service.NewTokenService(
		cfg.JWTSecret,
		cfg.JWTAccessTTL,
	)

	mw := middleware.NewAuth(tokens, cfg)

	// 3. Router & Handlers
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return origin == "http://localhost:4200" ||
				origin == "http://127.0.0.1:4200" || // local cross-site testing (like the mobile app)
				origin == "https://localhost" || // Capacitor Android
				origin == "capacitor://localhost" || // Capacitor iOS
				strings.HasSuffix(origin, ".onrender.com")
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Requested-With",
			"X-Kladovki-Client", // mobile app asks for a bearer token
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	h := handlers.New(svc, tokens, cfg, mw)
	h.Register(r)

	staticPath := "./frontend/dist/frontend/browser"

	r.NoRoute(func(c *gin.Context) {
		// Pass through API 404s cleanly
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
			return
		}

		// Check if requested file exists (js, css, images, etc.)
		filePath := filepath.Join(staticPath, filepath.Clean(c.Request.URL.Path))
		if _, err := os.Stat(filePath); err == nil {
			c.File(filePath)
			return
		}

		// Fallback to index.html for Single Page Application (SPA) client-side routing
		c.File(filepath.Join(staticPath, "index.html"))
	})

	// 4. HTTP Server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// 5. Background workers
	workerCtx, workerCancel := context.WithCancel(
		context.Background(),
	)
	defer workerCancel()

	go svc.StartRentalWorker(workerCtx)
	go svc.StartNotificationWorker(workerCtx)

	if notifyCfg.TelegramToken != "" && notifyCfg.AppBaseURL != "" && notifyCfg.TelegramWebhookSecret != "" {
		hook := notifyCfg.AppBaseURL + cfg.APIPrefix + "/telegram/webhook"
		if err := notify.NewTelegram(notifyCfg).SetWebhook(context.Background(), hook); err != nil {
			log.Printf("telegram webhook: %v", err)
		} else {
			log.Printf("telegram webhook registered: %s", hook)
		}
	}

	// 6. Routes
	for _, route := range r.Routes() {
		log.Printf(
			"%-6s %s",
			route.Method,
			route.Path,
		)
	}

	// 7. HTTP Server
	go func() {
		log.Printf(
			"Server listening on http://localhost:%s (%s mode)",
			cfg.Port,
			cfg.Env,
		)

		if err := srv.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Fatalf(
				"listen error: %s\n",
				err,
			)
		}
	}()

	// 8. Graceful Shutdown
	quit := make(chan os.Signal, 1)

	signal.Notify(
		quit,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-quit

	log.Println("Shutting down server...")

	// Stop background workers.
	workerCancel()

	// Shutdown HTTP server.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(
			"Server forced to shutdown:",
			err,
		)
	}

	log.Println("Server exiting")
}
