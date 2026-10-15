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
	"uptime_app/backend/internal/config"
	"uptime_app/backend/internal/httpapi"
	"uptime_app/backend/internal/mail"
	"uptime_app/backend/internal/monitor"
	"uptime_app/backend/internal/storage"
)

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
	worker := &storage.MailWorker{DB: db, Cipher: cipher, Sender: mail.SMTP{Addr: cfg.SMTPAddr, User: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.MailFrom, AllowInsecure: cfg.SMTPInsecure}}
	workerCtx, workerCancel := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() { defer close(workerDone); worker.Run(workerCtx) }()
	defer func() { workerCancel(); <-workerDone }()
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewApplicationRouter(service, &monitor.Service{Repo: &storage.MonitorRepository{DB: db}}, cfg),
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
