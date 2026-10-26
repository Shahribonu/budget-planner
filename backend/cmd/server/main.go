// Command server runs the HTTP API. The bot webhook and job worker will join it as
// additional entry points in the same binary.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/api"
	"github.com/Shahribonu/budget-planner/backend/internal/db"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	dbURL := os.Getenv("DATABASE_URL")
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	env := getenv("APP_ENV", "production")
	addr := ":" + getenv("PORT", "8080")
	if dbURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	if botToken == "" && env != "development" {
		return errors.New("TELEGRAM_BOT_TOKEN is required")
	}

	if getenv("MIGRATE", "true") == "true" {
		if err := db.Migrate(dbURL); err != nil {
			return err
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.New(api.Config{BotToken: botToken, DevAuth: env == "development"}, pool),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", addr, "env", env)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdown)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
