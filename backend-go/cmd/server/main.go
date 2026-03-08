package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/anthropics/accounting-tool/backend-go/internal/config"
	"github.com/anthropics/accounting-tool/backend-go/internal/database"
	"github.com/anthropics/accounting-tool/backend-go/internal/handlers"
	"github.com/anthropics/accounting-tool/backend-go/internal/middleware"
	"github.com/anthropics/accounting-tool/backend-go/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := database.Init(cfg.DatabaseURL); err != nil {
		log.Fatalf("database: %v", err)
	}

	db := database.DB()
	gemini := services.NewGeminiExtractService(cfg.GeminiAPIKey, cfg.GeminiModel)

	r := chi.NewRouter()
	r.Use(middleware.CORS(cfg.CORSOrigins))
	r.Use(middleware.Logging)
	r.Use(middleware.BodyLimit(50 << 20)) // 50MB global limit

	// Health with DB check
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		if err := database.DB().Ping(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": "database unreachable"})
			return
		}
		handlers.WriteJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	})

	// Config endpoint
	r.Get("/api/config", func(w http.ResponseWriter, _ *http.Request) {
		handlers.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"currency_symbol":   cfg.DefaultCurrencySymbol,
			"locale":            cfg.DefaultLocale,
			"supported_formats": []string{"pdf", "csv"},
			"app_name":          "Personal Accounting System",
		})
	})

	// Register all handlers
	handlers.NewCategoryHandler(db).Routes(r)
	handlers.NewRuleHandler(db).Routes(r)
	handlers.NewBudgetHandler(db).Routes(r)
	handlers.NewTransactionHandler(db).Routes(r)
	handlers.NewStatementHandler(db, cfg, gemini).Routes(r)
	handlers.NewReceiptHandler(db, cfg, gemini).Routes(r)
	handlers.NewImportHandler(db, cfg, gemini).Routes(r)
	handlers.NewMerchantHandler(db).Routes(r)

	// Static file serving (no directory listing, path traversal safe)
	r.Get("/files/pages/*", func(w http.ResponseWriter, r *http.Request) {
		path := chi.URLParam(r, "*")
		serveFileNoDir(w, r, cfg.PageImagesDir, path)
	})
	r.Get("/files/receipts/*", func(w http.ResponseWriter, r *http.Request) {
		path := chi.URLParam(r, "*")
		serveFileNoDir(w, r, cfg.ReceiptsDir, path)
	})

	// SPA catch-all: serve frontend static files with index.html fallback
	staticDir := envOrDefault("STATIC_DIR", "./static")
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
		})
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(staticDir, r.URL.Path)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				http.ServeFile(w, r, path)
				return
			}
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
		})
	} else {
		r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
			handlers.WriteJSON(w, http.StatusOK, map[string]string{
				"status":  "ok",
				"message": "Personal Accounting System API",
			})
		})
	}

	addr := net.JoinHostPort(cfg.Host, cfg.Port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	if err := database.Close(); err != nil {
		log.Printf("database close error: %v", err)
	}
	log.Println("stopped")
}

func serveFileNoDir(w http.ResponseWriter, r *http.Request, baseDir, requestPath string) {
	if requestPath == "" || strings.HasSuffix(requestPath, "/") {
		http.NotFound(w, r)
		return
	}
	fullPath := filepath.Join(baseDir, filepath.Base(requestPath))
	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, fullPath)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
