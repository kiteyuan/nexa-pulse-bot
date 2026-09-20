package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kiteyuan/nexa-pulse-bot/internal/app"
	"github.com/kiteyuan/nexa-pulse-bot/internal/collect/rss"
	"github.com/kiteyuan/nexa-pulse-bot/internal/collect/telegram"
	"github.com/kiteyuan/nexa-pulse-bot/internal/config"
	"github.com/kiteyuan/nexa-pulse-bot/internal/httpapi"
	"github.com/kiteyuan/nexa-pulse-bot/internal/imgbed"
	"github.com/kiteyuan/nexa-pulse-bot/internal/ports"
	"github.com/kiteyuan/nexa-pulse-bot/internal/store"
	"github.com/kiteyuan/nexa-pulse-bot/internal/webui"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	var bed ports.ImageUploader
	if cfg.ImageBaseURL != "" {
		bed = &imgbed.Client{BaseURL: cfg.ImageBaseURL, AuthCode: cfg.ImageAuth}
	}
	tg := &telegram.Service{DB: db, Sessions: cfg.SessionDir, Proxy: cfg.TGProxy, Bed: bed}
	feeds := &rss.Service{DB: db, Bed: bed}
	engine := &app.Engine{
		Pipe:    db,
		Sources: []ports.Intake{tg, feeds},
	}
	engine.Run(ctx)

	var publicFS, adminFS fs.FS
	if !cfg.Dev {
		publicFS, _ = fs.Sub(webui.Public(), "public")
		adminFS, _ = fs.Sub(webui.Admin(), "admin")
	}
	api := &httpapi.Server{
		Accounts: db,
		Feeds:    db,
		Themes:   db,
		Content:  db,
		Cfg:      cfg,
		TG:       tg,
		Public:   publicFS,
		Admin:    adminFS,
	}

	publicSrv := &http.Server{
		Addr: cfg.PublicAddr, Handler: api.PublicHandler(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	adminSrv := &http.Server{
		Addr: cfg.AdminAddr, Handler: api.AdminHandler(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	go func() {
		slog.Info("public", "addr", cfg.PublicAddr)
		if err := publicSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("public", "err", err)
			stop()
		}
	}()
	go func() {
		slog.Info("admin", "addr", cfg.AdminAddr)
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("admin", "err", err)
			stop()
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = publicSrv.Shutdown(shutdown)
	_ = adminSrv.Shutdown(shutdown)
}
