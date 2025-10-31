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

type sendResponse struct {
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
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
	Send(ctx context.Context, req SendRequest) (providerID string, err error)
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
func (s *httpSender) Send(ctx context.Context, req SendRequest) (string, error) {

	var lastErr error
	var sleepOnRetry = func(attempt int) {
		if s.cfg.MaxRetries > 1 {
			time.Sleep(time.Duration(attempt) * DefaultRetryDelay)
		}
	}

	for attempt := 1; attempt <= s.cfg.MaxRetries; attempt++ {

		req, err := s.buildReq(ctx, req)
		if err != nil {
			lastErr = err
			sleepOnRetry(attempt)
			continue
		}

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			sleepOnRetry(attempt)
			continue
		}

		msgId, err := s.parseMessageId(resp)
		if err != nil {
			lastErr = err
			sleepOnRetry(attempt)
			continue
		}

		err = resp.Body.Close()
		if err != nil {
			s.log.Error("send: close response body error", zap.Error(err))
		}
		return msgId, nil

	}
	if lastErr == nil {
		lastErr = errors.New("send failed")
	}
	return "", lastErr
}

func (s *httpSender) buildReq(ctx context.Context, sendReq SendRequest) (*http.Request, error) {
	b, err := json.Marshal(sendReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.URL, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if s.cfg.AuthHeader != "" && s.cfg.AuthValue != "" {
		req.Header.Set(s.cfg.AuthHeader, s.cfg.AuthValue)
	}

	return req, nil
}

func (s *httpSender) parseMessageId(resp *http.Response) (string, error) {
	if resp.StatusCode != s.cfg.ExpectStatus {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var out sendResponse
	decErr := json.NewDecoder(resp.Body).Decode(&out)
	if decErr != nil {
		return "", fmt.Errorf("decode response: %w", decErr)
	}

	msgId := out.MessageID
	if msgId == "" {
		return "", errors.New("empty messageId in response")
	}

	return msgId, nil
}
