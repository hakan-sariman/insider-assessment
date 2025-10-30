package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/hakan-sariman/insider-assessment/internal/service"
	"go.uber.org/zap"
)

// Server is the API server
type Server struct {
	cfg    ServerCfg
	msgSvc *service.MessageService
	log    *zap.Logger
	http   *http.Server
}

// ServerCfg is the configuration for the API server
type ServerCfg struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// Scheduler is the scheduler interface for API controls
type Scheduler interface {
	Start(ctx context.Context)
	Stop()
}

// NewServer creates a new API server
// and registers the routes
func NewServer(cfg ServerCfg, msgSvc *service.MessageService, log *zap.Logger) *Server {
	r := mux.NewRouter()
	s := &Server{
		cfg:    cfg,
		msgSvc: msgSvc,
		log:    log,
	}

	// health check
	r.HandleFunc("/healthz", s.healthz).Methods("GET")

	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/scheduler/start", s.startScheduler).Methods("POST")
	api.HandleFunc("/scheduler/stop", s.stopScheduler).Methods("POST")
	api.HandleFunc("/messages", s.createMessage).Methods("POST")
	api.HandleFunc("/messages", s.listMessages).Methods("GET")

	// swagger is registered via build tag in swagger_enabled.go; noop otherwise.
	registerSwagger(r)

	s.http = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	return s
}

// Start starts the API server
// and the message service
func (s *Server) Start() error {
	s.log.Info("API server starting...")
	s.msgSvc.Start(context.Background())
	s.log.Info("http server listening", zap.String("addr", s.http.Addr))
	return s.http.ListenAndServe()
}

// Shutdown shuts down the API server
// and the message service
func (s *Server) Shutdown(ctx context.Context) error {
	s.msgSvc.Stop()
	return s.http.Shutdown(ctx)
}
