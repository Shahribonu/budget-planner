// Command server runs the HTTP API, the Telegram bot (webhook or long polling) and the
// once-a-minute scheduler in one process.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/api"
	"github.com/Shahribonu/budget-planner/backend/internal/bot"
	"github.com/Shahribonu/budget-planner/backend/internal/capture"
	"github.com/Shahribonu/budget-planner/backend/internal/capture/llm"
	"github.com/Shahribonu/budget-planner/backend/internal/capture/stt"
	"github.com/Shahribonu/budget-planner/backend/internal/db"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
	"github.com/Shahribonu/budget-planner/backend/internal/telegram"
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

	l := ledger.New(pool)
	ai := llm.New(os.Getenv("ANTHROPIC_API_KEY"))
	if ai == nil {
		slog.Info("ANTHROPIC_API_KEY not set: category fallback uses rules only")
	}
	transcriber, err := stt.New(os.Getenv("STT_PROVIDER"))
	if err != nil {
		return err
	}
	cfg := api.Config{BotToken: botToken, DevAuth: env == "development"}

	if botToken != "" {
		tg := telegram.New(botToken)
		b := bot.New(bot.Config{MiniAppURL: os.Getenv("MINIAPP_URL")}, tg, pool, capture.New(pool, l, ai), transcriber)
		mode := getenv("BOT_MODE", map[bool]string{true: "polling", false: "webhook"}[env == "development"])
		switch mode {
		case "polling":
			go func() {
				if err := b.Poll(ctx); err != nil {
					slog.Error("bot polling stopped", "err", err)
				}
			}()
		case "webhook":
			secret, base := os.Getenv("TELEGRAM_WEBHOOK_SECRET"), os.Getenv("PUBLIC_URL")
			if secret == "" || base == "" {
				return errors.New("BOT_MODE=webhook needs TELEGRAM_WEBHOOK_SECRET and PUBLIC_URL")
			}
			cfg.TelegramWebhook = b.Webhook(secret)
			if err := tg.SetWebhook(ctx, base+"/telegram/webhook", secret); err != nil {
				return fmt.Errorf("set webhook: %w", err)
			}
		case "off":
		default:
			return fmt.Errorf("unknown BOT_MODE %q", mode)
		}
		if mode != "off" {
			b.SetCommands(ctx)
			go b.RunScheduler(ctx)
		}
		slog.Info("bot started", "mode", mode)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.New(cfg, pool),
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
