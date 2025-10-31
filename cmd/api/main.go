package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/hakan-sariman/insider-assessment/internal/api"
	"github.com/hakan-sariman/insider-assessment/internal/cache"
	"github.com/hakan-sariman/insider-assessment/internal/config"
	"github.com/hakan-sariman/insider-assessment/internal/logx"
	"github.com/hakan-sariman/insider-assessment/internal/outbound"
	"github.com/hakan-sariman/insider-assessment/internal/scheduler"
	"github.com/hakan-sariman/insider-assessment/internal/service"
	postgresstorage "github.com/hakan-sariman/insider-assessment/internal/storage/postgres"

	"go.uber.org/zap"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// run migrations before db pool is used
func runMigrations(dbUrl string, logger *zap.Logger) error {
	m, err := migrate.New(
		"file:///app/internal/storage/migrations",
		dbUrl,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration error: %w", err)
	}
	return nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	logger, err := logx.New(cfg.App.Env)
	if err != nil {
		panic(fmt.Errorf("new logger: %w", err))
	}

	logger.Info("application starting...", zap.String("env", cfg.App.Env))

	// run DB migrations before db pool is used
	if err := runMigrations(cfg.Postgres.URL, logger); err != nil {
		logger.Error("failed to run migrations", zap.Error(err))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// postgres
	db, err := postgresstorage.New(ctx, cfg.Postgres.URL, cfg.Postgres.MaxOpenConns, logger)
	if err != nil {
		logger.Fatal("postgres connect", zap.Error(err))
	}
	defer db.Close()

	// redis
	redisClient := cache.NewRedis(cfg.Redis.Addr, cfg.Redis.DB)
	defer redisClient.Close()

	// outbound sender
	sender := outbound.NewHTTP(outbound.Config{
		URL:          cfg.Outbound.URL,
		Timeout:      cfg.Outbound.Timeout,
		MaxRetries:   cfg.Outbound.MaxRetries,
		ExpectStatus: cfg.Outbound.ExpectStatus,
		AuthHeader:   cfg.Outbound.AuthHeader,
		AuthValue:    cfg.Outbound.AuthValue,
	}, logger)

	// scheduler
	sched := scheduler.New(scheduler.Config{
		Enabled:   cfg.Scheduler.Enabled,
		Interval:  cfg.Scheduler.Interval,
		BatchSize: cfg.Scheduler.BatchSize,
	}, db, redisClient, sender, logger)

	msgSvc := service.NewMessageService(db, logger, sched, sender)
	schedSvc := service.NewScheduler(sched, logger)

	// HTTP server
	srv := api.NewServer(api.ServerCfg{
		Port:         cfg.Server.Port,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		IsProd:       cfg.App.Env == "prod",
	}, msgSvc, schedSvc, logger)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server start", zap.Error(err))
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		logger.Info("http server shutdown", zap.Error(err))
	} else {
		logger.Info("http server shutdown successfully")
	}
	time.Sleep(100 * time.Millisecond)
	err = logger.Sync()
	if err != nil {
		logger.Error("logger sync", zap.Error(err))
	}
}
