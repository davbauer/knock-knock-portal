package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/davbauer/knock-knock-portal/internal/api"
	"github.com/davbauer/knock-knock-portal/internal/auth"
	"github.com/davbauer/knock-knock-portal/internal/config"
	"github.com/davbauer/knock-knock-portal/internal/ipallowlist"
	"github.com/davbauer/knock-knock-portal/internal/session"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const Version = "1.0.0"

func main() {
	// Load .env file
	_ = godotenv.Load() // Ignore error if .env doesn't exist
	
	// Setup logging
	setupLogging()

	log.Info().Str("version", Version).Msg("Starting Knock-Knock Portal")

	// Load configuration
	configPath := os.Getenv("CONFIG_FILE_PATH")
	if configPath == "" {
		configPath = "./config.yml"
	}

	configLoader, err := config.NewLoader(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}
	defer configLoader.Close()

	cfg := configLoader.GetConfig()

	// Initialize JWT manager
	jwtManager, err := auth.NewJWTManager()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize JWT manager")
	}

	// Initialize password verifier
	passwordVerifier, err := auth.NewPasswordVerifier()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize password verifier")
	}

	// Initialize session manager
	var maxDuration *time.Duration
	if cfg.SessionConfig.MaximumSessionDurationSeconds != nil {
		d := time.Duration(*cfg.SessionConfig.MaximumSessionDurationSeconds) * time.Second
		maxDuration = &d
	}

	sessionManager := session.NewManager(
		time.Duration(cfg.SessionConfig.DefaultSessionDurationSeconds)*time.Second,
		maxDuration,
		cfg.SessionConfig.AutoExtendSessionOnConnection,
		time.Duration(cfg.SessionConfig.SessionCleanupIntervalSeconds)*time.Second,
	)
	defer sessionManager.Close()

	// Initialize IP allowlist manager
	allowlistManager := ipallowlist.NewManager(&cfg.NetworkAccessControl)
	defer allowlistManager.Close()

	// Setup API router
	router := api.NewRouter(
		configLoader,
		jwtManager,
		passwordVerifier,
		sessionManager,
		allowlistManager,
	)

	// Start HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ProxyServerConfig.AdminAPIPort),
		Handler: router.GetEngine(),
	}

	// Graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	log.Info().
		Int("port", cfg.ProxyServerConfig.AdminAPIPort).
		Msg("HTTP API server started")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped")
}

func setupLogging() {
	// Configure zerolog
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Configure output format
	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "text" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}
}
