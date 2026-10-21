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

	"uptime_app/backend/internal/auth"
	"uptime_app/backend/internal/avatar"
	"uptime_app/backend/internal/config"
	"uptime_app/backend/internal/httpapi"
	"uptime_app/backend/internal/mail"
	"uptime_app/backend/internal/monitor"
	"uptime_app/backend/internal/storage"
)

// @title Uptime API
// @version 1.0
// @description API for authentication, profiles, avatars and uptime monitors.
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter Bearer followed by a space and the access token.
func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := storage.Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	cipher, err := auth.NewCipher(cfg.MailKey)
	if err != nil {
		return err
	}
	service, err := auth.NewService(&storage.AuthRepository{DB: db}, auth.Signer{Key: cfg.JWTKey, Issuer: cfg.Issuer, Audience: cfg.Audience}, cipher, cfg.PublicURL)
	if err != nil {
		return err
	}
	service.Avatars = avatar.LocalStore{Dir: cfg.AvatarDir}
	worker := &storage.MailWorker{DB: db, Cipher: cipher, Sender: mail.SMTP{Addr: cfg.SMTPAddr, User: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.MailFrom, AllowInsecure: cfg.SMTPInsecure}}
	workerCtx, workerCancel := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() { defer close(workerDone); worker.Run(workerCtx) }()
	maintenanceDone := make(chan struct{})
	go func() { defer close(maintenanceDone); storage.RunMaintenance(workerCtx, db, cfg.AvatarDir) }()
	monitorRepo := &storage.MonitorRepository{DB: db}
	monitorDone := make(chan struct{})
	go func() { defer close(monitorDone); monitorRepo.RunChecks(workerCtx) }()
	defer func() { workerCancel(); <-workerDone; <-maintenanceDone; <-monitorDone }()
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewLimitedApplicationRouter(service, &monitor.Service{Repo: monitorRepo}, cfg, &storage.RateLimiter{DB: db}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("API starting", "addr", cfg.HTTPAddr)
		errCh <- server.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		slog.Info("API shutting down")
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
