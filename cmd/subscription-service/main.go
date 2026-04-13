// @title Subscriptions REST API
// @version 1.0
// @description CRUDL for user subscriptions and monthly cost aggregation (PostgreSQL).
// @termsOfService http://swagger.io/terms/

// @contact.name Support

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1
//
//go:generate sh -c "cd ../.. && go run github.com/swaggo/swag/cmd/swag@v1.16.3 init -g cmd/subscription-service/main.go -o docs --parseInternal --parseDependency"
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wassiliy/subscriptions-service/internal/config"
	"github.com/wassiliy/subscriptions-service/internal/db"
	"github.com/wassiliy/subscriptions-service/internal/handler"
	"github.com/wassiliy/subscriptions-service/internal/repository"
	"github.com/wassiliy/subscriptions-service/internal/service"

	_ "github.com/wassiliy/subscriptions-service/docs"
)

func main() {
	ctx := context.Background()

	yamlPath := os.Getenv("CONFIG_YAML")
	if _, set := os.LookupEnv("CONFIG_YAML"); !set {
		yamlPath = "config.yaml"
	}
	cfg, err := config.Load(yamlPath)
	if err != nil {
		slog.Error("load config", slog.Any("err", err))
		os.Exit(1)
	}

	log := newLogger(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(log)

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, cfg.DBRetry, cfg.DBRetryWait)
	if err != nil {
		log.Error("database pool", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("database connected")

	if err := db.RunMigrations(cfg.Migrations, cfg.DatabaseURL); err != nil {
		log.Error("migrations", slog.Any("err", err))
		os.Exit(1)
	}
	log.Info("migrations applied", slog.String("source", cfg.Migrations))

	repo := repository.NewSubscription(pool)
	svc := service.NewSubscription(repo, log)
	h := handler.New(svc)
	router := handler.NewRouter(log, h)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("http server listening", slog.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown", slog.Any("err", err))
	}
	log.Info("server stopped")
}

func newLogger(level, format string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lvl}
	var h slog.Handler
	if format == "json" {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(h)
}
