package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hakan-sariman/insider-assessment/internal/model"
	"github.com/hakan-sariman/insider-assessment/internal/service"
	"go.uber.org/zap"
)

type fakeMsgSvc struct {
	createResp *model.Message
	createErr  error
	listResp   []model.Message
	listErr    error
}

func (f *fakeMsgSvc) CreateMessage(ctx context.Context, req service.CreateMessageRequest) (*model.Message, error) {
	return f.createResp, f.createErr
}
func (f *fakeMsgSvc) ListSentMessages(ctx context.Context, limit, offset int) ([]model.Message, error) {
	return f.listResp, f.listErr
}

type fakeSchedSvc struct{ started, stopped bool }

func (f *fakeSchedSvc) Start(ctx context.Context) { f.started = true }
func (f *fakeSchedSvc) Stop(reason error)         { f.stopped = true }

func newTestServer(m service.Message, s service.Scheduler) *Server {
	cfg := ServerCfg{Port: 0, ReadTimeout: time.Second, WriteTimeout: time.Second, IdleTimeout: time.Second, IsProd: true}
	return NewServer(cfg, m, s, zap.NewNop())
}

func TestHealthz(t *testing.T) {
	s := newTestServer(&fakeMsgSvc{}, &fakeSchedSvc{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	s.healthz(rr, req)
	if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != "ok" {
		t.Fatalf("unexpected: %d %s", rr.Code, rr.Body.String())
	}
}

func TestCreateMessage(t *testing.T) {
	msg := &model.Message{To: "a", Content: "b"}
	s := newTestServer(&fakeMsgSvc{createResp: msg}, &fakeSchedSvc{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(`{"to":"a","content":"b"}`))
	rr := httptest.NewRecorder()
	s.createMessage(rr, req)
	if rr.Code != 201 {
		t.Fatalf("unexpected code: %d", rr.Code)
	}
	var out model.Message
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.To != "a" || out.Content != "b" {
		t.Fatalf("unexpected body: %#v", out)
	}
}

func TestCreateMessage_InvalidJSON(t *testing.T) {
	s := newTestServer(&fakeMsgSvc{}, &fakeSchedSvc{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader("{"))
	rr := httptest.NewRecorder()
	s.createMessage(rr, req)
	if rr.Code != 400 {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCreateMessage_ServiceError(t *testing.T) {
	s := newTestServer(&fakeMsgSvc{createErr: errors.New("bad")}, &fakeSchedSvc{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(`{"to":"a","content":"b"}`))
	rr := httptest.NewRecorder()
	s.createMessage(rr, req)
	if rr.Code != 400 {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestListMessages_Success(t *testing.T) {
	msgs := []model.Message{{To: "x"}}
	s := newTestServer(&fakeMsgSvc{listResp: msgs}, &fakeSchedSvc{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?limit=10&offset=0", nil)
	rr := httptest.NewRecorder()
	s.listMessages(rr, req)
	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestListMessages_Error(t *testing.T) {
	s := newTestServer(&fakeMsgSvc{listErr: errors.New("db")}, &fakeSchedSvc{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?limit=10&offset=0", nil)
	rr := httptest.NewRecorder()
	s.listMessages(rr, req)
	if rr.Code != 500 {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestSchedulerControls(t *testing.T) {
	fs := &fakeSchedSvc{}
	s := newTestServer(&fakeMsgSvc{}, fs)
	rr := httptest.NewRecorder()
	s.startScheduler(rr, httptest.NewRequest(http.MethodPost, "/api/v1/scheduler/start", nil))
	if rr.Code != 200 || !fs.started {
		t.Fatalf("start failed")
	}
	rr = httptest.NewRecorder()
	s.stopScheduler(rr, httptest.NewRequest(http.MethodPost, "/api/v1/scheduler/stop", nil))
	if rr.Code != 200 || !fs.stopped {
		t.Fatalf("stop failed")
	}
}
