package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/The-Christopher-Robin/pulse-front/backend/internal/analytics"
	"github.com/The-Christopher-Robin/pulse-front/backend/internal/cache"
	"github.com/The-Christopher-Robin/pulse-front/backend/internal/catalog"
	"github.com/The-Christopher-Robin/pulse-front/backend/internal/config"
	"github.com/The-Christopher-Robin/pulse-front/backend/internal/db"
	"github.com/The-Christopher-Robin/pulse-front/backend/internal/experiments"
	"github.com/The-Christopher-Robin/pulse-front/backend/internal/grpcapi"
	"github.com/The-Christopher-Robin/pulse-front/backend/internal/httpapi"
	"github.com/The-Christopher-Robin/pulse-front/backend/internal/seed"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := db.Open(ctx, cfg.PostgresURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Migrate(ctx); err != nil {
		return err
	}
	if err := seed.Run(ctx, pool.Pool); err != nil {
		return err
	}

	rdb, err := cache.Open(ctx, cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		return err
	}
	defer rdb.Close()

	writer := analytics.NewWriter(pool.Pool, cfg.ExposureBuffer, cfg.ExposureFlushDur)
	writer.Start(ctx)
	defer writer.Stop()

	registry := experiments.NewRegistry(pool.Pool)
	if err := registry.Load(ctx); err != nil {
		return err
	}
	go registry.Watch(ctx, 30*time.Second, func(err error) {
		log.Printf("registry refresh: %v", err)
	})

	expService := experiments.NewService(registry, rdb, writer)
	catService := catalog.NewService(pool.Pool)

	router := httpapi.NewRouter(httpapi.Deps{
		Catalog:       catService,
		Experiments:   expService,
		Writer:        writer,
		AllowedOrigin: cfg.AllowedOrigin,
	})

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	grpcSrv, err := grpcapi.NewServer(cfg.GRPCAddr, writer)
	if err != nil {
		return err
	}

	errCh := make(chan error, 2)
	go func() {
		log.Printf("http listening on %s", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		log.Printf("grpc listening on %s", grpcSrv.Addr())
		if err := grpcSrv.Start(); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Printf("shutdown signal received")
	case err := <-errCh:
		log.Printf("server error: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	grpcSrv.GracefulStop()
	return nil
}
