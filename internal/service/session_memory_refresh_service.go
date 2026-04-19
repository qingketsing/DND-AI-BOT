package service

import (
	"context"
	"errors"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/observability"
	"DND-AI-BOT/internal/repository"

	"github.com/prometheus/client_golang/prometheus"
)

// SessionMemoryRefreshService 负责在历史过长时将较老消息压缩进 session memory。
type SessionMemoryRefreshService struct {
	sessions         repository.SessionRepository
	memoryService    *SessionMemoryService
	summarizer       SessionSummarizer
	messageWindow    int
	summaryThreshold int
	metrics          *observability.Metrics
}

// SessionMemoryRefreshOption 定义会话记忆刷新服务可选配置。
type SessionMemoryRefreshOption func(*SessionMemoryRefreshService)

// NewSessionMemoryRefreshService 创建会话记忆刷新服务。
func NewSessionMemoryRefreshService(
	sessions repository.SessionRepository,
	memoryService *SessionMemoryService,
	summarizer SessionSummarizer,
	messageWindow int,
	summaryThreshold int,
	options ...SessionMemoryRefreshOption,
) *SessionMemoryRefreshService {
	service := &SessionMemoryRefreshService{
		sessions:         sessions,
		memoryService:    memoryService,
		summarizer:       summarizer,
		messageWindow:    messageWindow,
		summaryThreshold: summaryThreshold,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
}

// WithSessionMemoryRefreshMetrics 注入会话记忆刷新指标。
func WithSessionMemoryRefreshMetrics(metrics *observability.Metrics) SessionMemoryRefreshOption {
	return func(service *SessionMemoryRefreshService) {
		if metrics != nil {
			service.metrics = metrics
		}
	}
}

// RefreshIfNeeded 在会话历史过长时摘要旧消息并合并进长期记忆。
func (s *SessionMemoryRefreshService) RefreshIfNeeded(ctx context.Context, sessionID string, now time.Time) error {
	session, err := s.sessions.Load(ctx, sessionID)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			s.recordRefreshMetric("skipped")
			return nil
		}
		s.recordRefreshMetric("error")
		return err
	}
	if len(session.History) < s.summaryThreshold {
		s.recordRefreshMetric("skipped")
		return nil
	}

	cutoff := len(session.History) - s.messageWindow
	if cutoff <= 0 {
		s.recordRefreshMetric("skipped")
		return nil
	}

	messages := make([]SummarizerMessage, 0, cutoff)
	for _, record := range session.History[:cutoff] {
		messages = append(messages, summarizerMessageFromRecord(record))
	}

	summary, err := s.summarizer.SummarizeMessages(ctx, sessionID, messages)
	if err != nil {
		s.recordRefreshMetric("error")
		return err
	}
	_, err = s.memoryService.MergeSummary(ctx, MergeSummaryInput{
		SessionID:        sessionID,
		CharacterSummary: summary.CharacterSummary,
		SceneSummary:     summary.SceneSummary,
		CurrentObjective: summary.CurrentObjective,
		RecentKeyEvents:  summary.RecentKeyEvents,
	}, now)
	if err != nil {
		s.recordRefreshMetric("error")
		return err
	}
	s.recordRefreshMetric("success")
	return err
}

func summarizerMessageFromRecord(record model.HistoryRecord) SummarizerMessage {
	return SummarizerMessage{
		Source:  string(record.Source),
		User:    record.User.Name,
		Content: record.Message.Content,
	}
}

func (s *SessionMemoryRefreshService) recordRefreshMetric(status string) {
	if s.metrics == nil {
		return
	}
	s.metrics.SessionMemoryRuns.With(prometheus.Labels{"status": status}).Inc()
}
