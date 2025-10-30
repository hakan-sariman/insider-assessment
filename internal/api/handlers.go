package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

type createMessageReq struct {
	To      string `json:"to"`
	Content string `json:"content"`
}

const (
	DefaultLimitListMessages = 50
)

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) createMessage(w http.ResponseWriter, r *http.Request) {
	s.log.Debug("createMessage API called")
	var req createMessageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.log.Error("createMessage: invalid json", zap.Error(err))
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	msg, err := s.msgSvc.CreateMessage(r.Context(), req.To, req.Content)
	if err != nil {
		s.log.Error("createMessage: failed", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.log.Info("createMessage: success", zap.String("id", msg.ID.String()))
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(msg)
}

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
	_ = json.NewEncoder(w).Encode(msgs)
}

func (s *Server) startScheduler(w http.ResponseWriter, r *http.Request) {
	s.log.Debug("startScheduler API called")
	s.msgSvc.StartScheduler(r.Context())
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("scheduler started"))
}

func (s *Server) stopScheduler(w http.ResponseWriter, r *http.Request) {
	s.log.Debug("stopScheduler API called")
	s.msgSvc.StopScheduler(errors.New("scheduler stopped by API"))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("scheduler stopped"))
}
