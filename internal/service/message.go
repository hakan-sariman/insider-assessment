package service

import (
	"context"

	"github.com/hakan-sariman/insider-assessment/internal/model"
	"github.com/hakan-sariman/insider-assessment/internal/outbound"
	"github.com/hakan-sariman/insider-assessment/internal/scheduler"
	"github.com/hakan-sariman/insider-assessment/internal/storage"

	"go.uber.org/zap"
)

// Message is the message service interface
type Message interface {
	CreateMessage(ctx context.Context, to, content string) (*model.Message, error)
	ListSentMessages(ctx context.Context, limit, offset int) ([]model.Message, error)
}

// message is the message service implementation
type message struct {
	store  storage.Storage
	logger *zap.Logger
	sched  *scheduler.Scheduler
	sender outbound.Sender
}

// NewMessageService creates a new message service
func NewMessageService(store storage.Storage, logger *zap.Logger, sched *scheduler.Scheduler, sender outbound.Sender) Message {
	return &message{
		store:  store,
		logger: logger,
		sched:  sched,
		sender: sender,
	}
}

// CreateMessage creates a new message
func (s *message) CreateMessage(ctx context.Context, to, content string) (*model.Message, error) {
	s.logger.Debug("CreateMessage", zap.String("to", to), zap.String("content", content))
	msg, err := model.NewMessage(to, content)
	if err != nil {
		s.logger.Error("CreateMessage: validation error", zap.Error(err))
		return nil, err
	}
	if err := s.store.InsertMessage(ctx, msg); err != nil {
		s.logger.Error("CreateMessage: db error", zap.Error(err))
		return nil, err
	}
	s.logger.Info("CreateMessage: stored", zap.String("id", msg.ID.String()))
	return msg, nil
}

// ListSentMessages lists sent messages
func (s *message) ListSentMessages(ctx context.Context, limit, offset int) ([]model.Message, error) {
	s.logger.Debug("ListSentMessages", zap.Int("limit", limit), zap.Int("offset", offset))
	msgs, err := s.store.ListSent(ctx, limit, offset)
	if err != nil {
		s.logger.Error("ListSentMessages: db error", zap.Error(err))
	}
	s.logger.Info("ListSentMessages: fetched", zap.Int("count", len(msgs)))
	return msgs, err
}
