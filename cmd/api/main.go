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
func runMigrations(dbUrl string, logger *zap.Logger) {
	m, err := migrate.New(
		"file:///app/internal/storage/migrations",
		dbUrl,
	)
	if err != nil {
		logger.Fatal("failed to initialize migrations", zap.Error(err))
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		logger.Fatal("migration error", zap.Error(err))
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("load config: %w", err))
	}

	logger := logx.New(cfg.App.Env)

	// run DB migrations before db pool is used
	runMigrations(cfg.Postgres.URL, logger)

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
		ExpectStatus: cfg.Outbound.ExpectStatus,
		AuthHeader:   cfg.Outbound.AuthHeader,
		AuthValue:    cfg.Outbound.AuthValue,
	}, logger)

	// scheduler
	sched := scheduler.New(scheduler.Config{
		Enabled:   cfg.Scheduler.Enabled,
		Interval:  cfg.Scheduler.Interval,
		BatchSize: cfg.Scheduler.BatchSize, // EXACT 2 per tick
	}, db, redisClient, sender, logger)

	msgSvc := service.NewMessageService(db, logger, sched, sender)

	// HTTP server
	srv := api.NewServer(api.ServerCfg{
		Port:         cfg.Server.Port,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}, msgSvc, logger)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server start", zap.Error(err))
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	_ = srv.Shutdown(shutdownCtx)
	time.Sleep(100 * time.Millisecond)
	logger.Sync()
}
