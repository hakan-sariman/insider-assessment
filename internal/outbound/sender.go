package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type SendRequest struct {
	To      string `json:"to"`
	Content string `json:"content"`
}

// Config is the configuration for the outbound sender
type Config struct {
	URL        string
	Timeout    time.Duration
	MaxRetries int
	// ExpectStatus is the expected HTTP status code from the outbound provider
	// Right now only one status code is supported,
	// might need to be extended in the future
	ExpectStatus int
	AuthHeader   string
	AuthValue    string
}

// Sender is the outbound sender interface
type Sender interface {
	// Send sends a message to the outbound provider
	Send(ctx context.Context, req SendRequest) (providerID *string, err error)
}

const (
	DefaultRetryDelay = 200 * time.Millisecond
)

// httpSender is the HTTP outbound sender
type httpSender struct {
	cfg    Config
	client *http.Client
	log    *zap.Logger
}

// NewHTTP creates a new HTTP outbound sender
func NewHTTP(cfg Config, log *zap.Logger) Sender {
	return &httpSender{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout},
		log:    log,
	}
}

// Send sends a message to the outbound provider
// and returns the provider message id
func (s *httpSender) Send(ctx context.Context, req SendRequest) (*string, error) {
	// req body
	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	var sleepOnRetry = func(attempt int) {
		if s.cfg.MaxRetries > 1 {
			time.Sleep(time.Duration(attempt) * DefaultRetryDelay)
		}
	}
	var lastErr error
	for attempt := 1; attempt <= s.cfg.MaxRetries; attempt++ {
		rCtx, rCtxCancel := context.WithTimeout(ctx, s.cfg.Timeout)
		defer rCtxCancel()

		req, _ := http.NewRequestWithContext(rCtx, http.MethodPost, s.cfg.URL, bytes.NewReader(b))

		req.Header.Set("Content-Type", "application/json")
		if s.cfg.AuthHeader != "" && s.cfg.AuthValue != "" {
			req.Header.Set(s.cfg.AuthHeader, s.cfg.AuthValue)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			sleepOnRetry(attempt)
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode == s.cfg.ExpectStatus {
			// provider message id unknown, placeholder
			ok := fmt.Sprintf("accepted-%d", time.Now().UnixNano())
			return &ok, nil
		}
		lastErr = fmt.Errorf("unexpected status %d", resp.StatusCode)
		sleepOnRetry(attempt)
	}
	if lastErr == nil {
		lastErr = errors.New("send failed")
	}
	return nil, lastErr
}
