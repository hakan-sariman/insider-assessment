package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/hakan-sariman/insider-assessment/internal/service"

	"go.uber.org/zap"
)

type createMessageReq struct {
	To      string `json:"to"`
	Content string `json:"content"`
}

const (
	DefaultLimitListMessages = 50
)

// healthz godoc
// @Summary Health check
// @Description Returns OK if the service is healthy
// @Tags Health
// @Produce plain
// @Success 200 {string} string "ok"
// @Router /healthz [get]
func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("ok"))
	if err != nil {
		s.log.Error("healthz: write error", zap.Error(err))
	}
}

// createMessage godoc
// @Summary Create a message
// @Description Creates a new message to be sent by the scheduler
// @Tags Messages
// @Accept json
// @Produce json
// @Param request body createMessageReq true "Create message payload"
// @Success 201 {object} model.Message
// @Failure 400 {string} string "invalid json or validation error"
// @Router /api/v1/messages [post]
func (s *Server) createMessage(w http.ResponseWriter, r *http.Request) {
	s.log.Debug("createMessage API called")
	var req createMessageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.log.Error("createMessage: invalid json", zap.Error(err))
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	msg, err := s.msgSvc.CreateMessage(r.Context(), service.CreateMessageRequest{
		To:      req.To,
		Content: req.Content,
	})
	if err != nil {
		s.log.Error("createMessage: failed", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.log.Info("createMessage: success", zap.String("id", msg.ID.String()))
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(msg)
	if err != nil {
		s.log.Error("createMessage: encode error", zap.Error(err))
	}
}

// listMessages godoc
// @Summary List sent messages
// @Description Returns a paginated list of sent messages
// @Tags Messages
// @Produce json
// @Param limit query int false "Max number of records" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} model.Message
// @Failure 500 {string} string "db error"
// @Router /api/v1/messages [get]
func (s *Server) listMessages(w http.ResponseWriter, r *http.Request) {
	s.log.Debug("listMessages API called")

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 {
		limit = DefaultLimitListMessages
	}
	if offset < 0 {
		offset = 0
	}

	msgs, err := s.msgSvc.ListSentMessages(r.Context(), limit, offset)
	if err != nil {
		s.log.Error("listMessages: db error", zap.Error(err))
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.log.Debug("listMessages: success", zap.Int("count", len(msgs)))
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(msgs)
	if err != nil {
		s.log.Error("listMessages: encode error", zap.Error(err))
	}
}

// startScheduler godoc
// @Summary Start scheduler
// @Description Starts the background scheduler that sends messages
// @Tags Scheduler
// @Produce plain
// @Success 200 {string} string "scheduler started"
// @Router /api/v1/scheduler/start [post]
func (s *Server) startScheduler(w http.ResponseWriter, r *http.Request) {
	s.log.Debug("startScheduler API called")
	s.schedSvc.Start(r.Context())
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("scheduler started"))
	if err != nil {
		s.log.Error("startScheduler: write error", zap.Error(err))
	}
}

// stopScheduler godoc
// @Summary Stop scheduler
// @Description Stops the background scheduler
// @Tags Scheduler
// @Produce plain
// @Success 200 {string} string "scheduler stopped"
// @Router /api/v1/scheduler/stop [post]
func (s *Server) stopScheduler(w http.ResponseWriter, r *http.Request) {
	s.log.Debug("stopScheduler API called")
	s.schedSvc.Stop(errors.New("scheduler stopped by API"))
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("scheduler stopped"))
	if err != nil {
		s.log.Error("stopScheduler: write error", zap.Error(err))
	}
}
