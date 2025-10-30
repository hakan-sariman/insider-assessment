package service

import (
	"context"
	"errors"

	"github.com/hakan-sariman/insider-assessment/internal/model"
	"github.com/hakan-sariman/insider-assessment/internal/outbound"
	"github.com/hakan-sariman/insider-assessment/internal/scheduler"
	"github.com/hakan-sariman/insider-assessment/internal/storage"
	"go.uber.org/zap"
)

type MessageService struct {
	store  storage.Storage
	logger *zap.Logger
	sched  *scheduler.Scheduler
	sender outbound.Sender
}

func NewMessageService(store storage.Storage, logger *zap.Logger, sched *scheduler.Scheduler, sender outbound.Sender) *MessageService {
	return &MessageService{
		store:  store,
		logger: logger,
		sched:  sched,
		sender: sender,
	}
}

func (s *MessageService) Start(ctx context.Context) {
	s.StartScheduler(ctx)
}

func (s *MessageService) Stop() {
	s.StopScheduler(errors.New("service stopped"))
}

func (s *MessageService) CreateMessage(ctx context.Context, to, content string) (*model.Message, error) {
	s.logger.Info("CreateMessage", zap.String("to", to), zap.String("content", content))
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

func (s *MessageService) ListSentMessages(ctx context.Context, limit, offset int) ([]model.Message, error) {
	s.logger.Info("ListSentMessages", zap.Int("limit", limit), zap.Int("offset", offset))
	msgs, err := s.store.ListSent(ctx, limit, offset)
	if err != nil {
		s.logger.Error("ListSentMessages: db error", zap.Error(err))
	}
	s.logger.Info("ListSentMessages: fetched", zap.Int("count", len(msgs)))
	return msgs, err
}

func (s *MessageService) StartScheduler(ctx context.Context) {
	if s.sched != nil {
		s.logger.Debug("Starting scheduler")
		s.sched.Start(ctx)
	}
}

func (s *MessageService) StopScheduler(reason error) {
	if s.sched != nil {
		s.logger.Debug("Stopping scheduler")
		s.sched.Stop(reason)
	}
}
