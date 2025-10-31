package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hakan-sariman/insider-assessment/internal/service"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Server is the API server
type Server struct {
	cfg      ServerCfg
	msgSvc   service.Message
	schedSvc service.Scheduler
	log      *zap.Logger
	http     *http.Server
}

// ServerCfg is the configuration for the API server
type ServerCfg struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	IsProd       bool
}

// NewServer creates a new API server
// and registers the routes
func NewServer(cfg ServerCfg, msgSvc service.Message, schedSvc service.Scheduler, log *zap.Logger) *Server {
	r := mux.NewRouter()
	s := &Server{
		cfg:      cfg,
		msgSvc:   msgSvc,
		schedSvc: schedSvc,
		log:      log,
	}

	// health check
	r.HandleFunc("/healthz", s.healthz).Methods("GET")

	api := r.PathPrefix("/api/v1").Subrouter()

	// api/v1/scheduler
	api.HandleFunc("/scheduler/start", s.startScheduler).Methods("POST")
	api.HandleFunc("/scheduler/stop", s.stopScheduler).Methods("POST")

	// api/v1/messages
	api.HandleFunc("/messages", s.createMessage).Methods("POST")
	api.HandleFunc("/messages", s.listMessages).Methods("GET")

	// if not production, register swagger
	if !cfg.IsProd {
		registerSwagger(r)
		s.log.Info("swagger endpoints registered")
	}

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
	s.schedSvc.Start(context.Background())
	s.log.Info("http server listening", zap.String("addr", s.http.Addr))
	return s.http.ListenAndServe()
}

// Shutdown shuts down the API server
// and the message service
func (s *Server) Shutdown(ctx context.Context) error {
	s.schedSvc.Stop(errors.New("server shutdown"))
	return s.http.Shutdown(ctx)
}
