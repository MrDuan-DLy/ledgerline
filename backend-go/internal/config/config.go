package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	GeminiAPIKey          string
	GeminiModel           string
	DataDir               string
	UploadsDir            string
	ReceiptsDir           string
	PageImagesDir         string
	Port                  string
	Host                  string
	DefaultCurrency       string
	DefaultCurrencySymbol string
	DefaultLocale         string
	DatabaseURL           string
	CORSOrigins           []string
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	dataDir := envOrDefault("DATA_DIR", "./data")
	uploadsDir := filepath.Join(dataDir, "uploads")
	receiptsDir := filepath.Join(dataDir, "receipts")
	pageImagesDir := filepath.Join(dataDir, "page_images")

	// Ensure directories exist
	for _, dir := range []string{dataDir, uploadsDir, receiptsDir, pageImagesDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	corsOrigins := envOrDefault("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")

	cfg := &Config{
		GeminiAPIKey:          os.Getenv("GEMINI_API_KEY"),
		GeminiModel:           envOrDefault("GEMINI_MODEL", "gemini-2.5-flash"),
		DataDir:               dataDir,
		UploadsDir:            uploadsDir,
		ReceiptsDir:           receiptsDir,
		PageImagesDir:         pageImagesDir,
		Port:                  envOrDefault("PORT", "8000"),
		Host:                  envOrDefault("HOST", "0.0.0.0"),
		DefaultCurrency:       envOrDefault("DEFAULT_CURRENCY", "GBP"),
		DefaultCurrencySymbol: envOrDefault("DEFAULT_CURRENCY_SYMBOL", "£"),
		DefaultLocale:         envOrDefault("DEFAULT_LOCALE", "en-GB"),
		DatabaseURL:           "file:" + filepath.Join(dataDir, "accounting.db") + "?_foreign_keys=on&_busy_timeout=5000",
		CORSOrigins:           strings.Split(corsOrigins, ","),
	}

	// Validate
	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return nil, fmt.Errorf("PORT must be numeric, got %q", cfg.Port)
	}
	if cfg.GeminiAPIKey == "" {
		log.Println("WARN: GEMINI_API_KEY not set — PDF and receipt AI features will be unavailable")
	}
	testFile := filepath.Join(dataDir, ".write-test")
	if err := os.WriteFile(testFile, []byte("ok"), 0o600); err != nil {
		return nil, fmt.Errorf("DATA_DIR %q is not writable: %w", dataDir, err)
	}
	os.Remove(testFile)

	return cfg, nil
}
