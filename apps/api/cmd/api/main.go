package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/ai"
	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/config"
	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/database"
	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/httpapi"
	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/resume"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		healthcheck()
		return
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startupCtx, cancel := context.WithTimeout(rootCtx, 20*time.Second)
	defer cancel()

	pool, err := database.Open(startupCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database startup failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := database.Migrate(startupCtx, pool); err != nil {
		logger.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	resumeStore := resume.NewPostgresStore(pool)
	sessions := ai.NewSessionStore(cfg.AISessionTTL)
	defer sessions.Close()
	providerClient := &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	aiService := ai.NewService(providerClient)

	handler := httpapi.New(httpapi.Dependencies{
		Logger:       logger,
		DB:           pool,
		Resumes:      resumeStore,
		Sessions:     sessions,
		AI:           aiService,
		CORSOrigin:   cfg.CORSOrigin,
		CookieSecure: cfg.CookieSecure,
		MaxBodyBytes: cfg.MaxBodyBytes,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      75 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("api listening", "address", cfg.HTTPAddr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-rootCtx.Done():
		logger.Info("shutdown requested")
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		_ = server.Close()
	}
}

func healthcheck() {
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = ":8080"
	}
	if _, port, err := net.SplitHostPort(address); err == nil {
		address = net.JoinHostPort("127.0.0.1", port)
	} else {
		address = "127.0.0.1:8080"
	}
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://" + address + "/api/v1/health")
	if err != nil {
		os.Exit(1)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
