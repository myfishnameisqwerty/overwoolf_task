package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"tracker/internal/config"
	"tracker/internal/db"
	"tracker/internal/handler"
	"tracker/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()

	if _, err := os.Stat(cfg.WidgetPath); err != nil {
		slog.Warn("widget file not found at startup", "path", cfg.WidgetPath)
	}
	if _, err := os.Stat(cfg.DashboardPath); err != nil {
		slog.Warn("dashboard file not found at startup", "path", cfg.DashboardPath)
	}

	store, err := db.New(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "path", cfg.DBPath, "error", err)
		os.Exit(1)
	}
	defer store.Close()

	h := &handler.Handler{
		Store:         store,
		WidgetPath:    cfg.WidgetPath,
		DashboardPath: cfg.DashboardPath,
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: server.New(h),
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		slog.Info("server started", "port", cfg.Port, "db", cfg.DBPath)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
