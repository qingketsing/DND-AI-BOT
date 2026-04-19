package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/observability"
)

func TestSessionMemoryRefreshServiceSkipsBelowThreshold(t *testing.T) {
	sessionRepo := &fakeSessionRepository{
		session: &model.Session{
			ID:      "session-1",
			History: make([]model.HistoryRecord, 10),
		},
	}
	memoryRepo := &fakeSessionMemoryRepository{}
	memoryService := NewSessionMemoryService(memoryRepo)
	refresh := NewSessionMemoryRefreshService(sessionRepo, memoryService, &fakeSessionSummarizer{}, 30, 40)

	if err := refresh.RefreshIfNeeded(context.Background(), "session-1", time.Now().UTC()); err != nil {
		t.Fatalf("expected no error below threshold, got %v", err)
	}
	if len(memoryRepo.saved) != 0 {
		t.Fatalf("expected no summary save below threshold, got %+v", memoryRepo.saved)
	}
}

func TestSessionMemoryRefreshServiceSummarizesOlderMessages(t *testing.T) {
	history := make([]model.HistoryRecord, 0, 45)
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 45; i++ {
		source := model.MessageSourceUser
		user := model.SessionUser{ID: "user-1", Name: "Qingke"}
		if i%2 == 1 {
			source = model.MessageSourceAgent
			user = model.SessionUser{ID: "agent", Name: "DM Agent"}
		}
		history = append(history, model.HistoryRecord{
			ID:        user.ID,
			User:      user,
			Message:   model.Message{Content: "message"},
			Sequence:  int64(i + 1),
			Source:    source,
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		})
	}

	sessionRepo := &fakeSessionRepository{
		session: &model.Session{
			ID:      "session-1",
			History: history,
		},
	}
	memoryRepo := &fakeSessionMemoryRepository{}
	memoryService := NewSessionMemoryService(memoryRepo)
	summarizer := &fakeSessionSummarizer{
		result: SummaryResult{
			SceneSummary:     "the city 广场",
			CurrentObjective: "寻找格伦",
			RecentKeyEvents:  []string{"查看布告栏"},
		},
	}
	refresh := NewSessionMemoryRefreshService(sessionRepo, memoryService, summarizer, 30, 40)

	if err := refresh.RefreshIfNeeded(context.Background(), "session-1", now); err != nil {
		t.Fatalf("expected refresh to succeed, got %v", err)
	}
	if summarizer.callCount != 1 {
		t.Fatalf("expected summarizer to be called once, got %d", summarizer.callCount)
	}
	if len(summarizer.messages) != 15 {
		t.Fatalf("expected only older messages to be summarized, got %d", len(summarizer.messages))
	}
	got := memoryRepo.saved["session-1"]
	if got == nil || got.SceneSummary != "the city 广场" || got.CurrentObjective != "寻找格伦" {
		t.Fatalf("expected summary merge to persist memory, got %+v", got)
	}
}

func TestSessionMemoryRefreshServiceRecordsSkippedMetric(t *testing.T) {
	metrics := observability.NewMetrics(prometheus.NewRegistry())
	sessionRepo := &fakeSessionRepository{
		session: &model.Session{
			ID:      "session-1",
			History: make([]model.HistoryRecord, 10),
		},
	}
	refresh := NewSessionMemoryRefreshService(
		sessionRepo,
		NewSessionMemoryService(&fakeSessionMemoryRepository{}),
		&fakeSessionSummarizer{},
		30,
		40,
		WithSessionMemoryRefreshMetrics(metrics),
	)

	if err := refresh.RefreshIfNeeded(context.Background(), "session-1", time.Now().UTC()); err != nil {
		t.Fatalf("expected no error below threshold, got %v", err)
	}

	body := scrapeMetrics(t, metrics)
	if !strings.Contains(body, `session_memory_refresh_total{status="skipped"} 1`) {
		t.Fatalf("expected skipped session memory metric, got %s", body)
	}
}

func TestSessionMemoryRefreshServiceRecordsSuccessMetric(t *testing.T) {
	metrics := observability.NewMetrics(prometheus.NewRegistry())
	history := make([]model.HistoryRecord, 45)
	sessionRepo := &fakeSessionRepository{
		session: &model.Session{
			ID:      "session-1",
			History: history,
		},
	}
	refresh := NewSessionMemoryRefreshService(
		sessionRepo,
		NewSessionMemoryService(&fakeSessionMemoryRepository{}),
		&fakeSessionSummarizer{},
		30,
		40,
		WithSessionMemoryRefreshMetrics(metrics),
	)

	if err := refresh.RefreshIfNeeded(context.Background(), "session-1", time.Now().UTC()); err != nil {
		t.Fatalf("expected refresh to succeed, got %v", err)
	}

	body := scrapeMetrics(t, metrics)
	if !strings.Contains(body, `session_memory_refresh_total{status="success"} 1`) {
		t.Fatalf("expected success session memory metric, got %s", body)
	}
}

type fakeSessionRepository struct {
	session *model.Session
	err     error
}

func (f *fakeSessionRepository) Save(ctx context.Context, session *model.Session) error {
	_ = ctx
	f.session = session
	return nil
}

func (f *fakeSessionRepository) Load(ctx context.Context, sessionID string) (*model.Session, error) {
	_ = ctx
	_ = sessionID
	return f.session, f.err
}

func (f *fakeSessionRepository) ListByUserID(ctx context.Context, userID string) ([]*model.Session, error) {
	_ = ctx
	_ = userID
	return nil, nil
}

func (f *fakeSessionRepository) Delete(ctx context.Context, sessionID string) error {
	_ = ctx
	_ = sessionID
	return nil
}

type fakeSessionSummarizer struct {
	result    SummaryResult
	err       error
	callCount int
	messages  []SummarizerMessage
}

func (f *fakeSessionSummarizer) SummarizeMessages(ctx context.Context, sessionID string, messages []SummarizerMessage) (SummaryResult, error) {
	_ = ctx
	_ = sessionID
	f.callCount++
	f.messages = append([]SummarizerMessage(nil), messages...)
	return f.result, f.err
}
