package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Status is the status of a message
type Status string

const (
	StatusUnsent Status = "unsent"
	StatusSent   Status = "sent"
)

const (
	// MaxContentLength is the maximum length of a message content
	MaxContentLength = 140
)

// Message is the message model
type Message struct {
	ID                uuid.UUID  `json:"id"`
	To                string     `json:"to"`
	Content           string     `json:"content"`
	Status            Status     `json:"status"`
	ProviderMessageID *string    `json:"provider_message_id,omitempty"`
	AttemptCount      int        `json:"attempt_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	SentAt            *time.Time `json:"sent_at,omitempty"`
	LastError         *string    `json:"last_error,omitempty"`
}

// NewMessage creates a new message
func NewMessage(to, content string) (*Message, error) {
	if len(content) > MaxContentLength {
		return nil, fmt.Errorf("content exceeds %d characters", MaxContentLength)
	}
	id := uuid.New()
	now := time.Now().UTC()
	return &Message{
		ID:           id,
		To:           to,
		Content:      content,
		Status:       StatusUnsent,
		AttemptCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}
