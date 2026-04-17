package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	agentprompt "DND-AI-BOT/internal/agent/prompt"
	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

var (
	// ErrInvalidMessage 表示消息内容为空或只包含空白字符。
	ErrInvalidMessage = errors.New("invalid message")
	// ErrInvalidChannel 表示传入了系统暂不支持的接入渠道。
	ErrInvalidChannel = errors.New("invalid channel")
	// ErrSessionForbidden 表示当前用户无权访问目标会话。
	ErrSessionForbidden = errors.New("session forbidden")
)

// SessionService 负责编排会话创建、读取和消息收发流程。
type SessionService struct {
	repository     repository.SessionRepository
	agentService   *AgentService
	refresher      SessionMemoryRefresher
	deleteCleaners []SessionDeleteCleaner
}

// SessionMemoryRefresher 定义消息发送完成后的长期记忆刷新接口。
type SessionMemoryRefresher interface {
	RefreshIfNeeded(ctx context.Context, sessionID string, now time.Time) error
}

// SessionDeleteCleaner 定义删除会话后需要清理的会话级派生数据或缓存。
type SessionDeleteCleaner interface {
	DeleteBySessionID(ctx context.Context, sessionID string) error
}

// CreateSessionInput 定义创建会话时需要的最小输入。
type CreateSessionInput struct {
	UserID  string
	Channel model.Channel
}

// SendMessageInput 定义发送消息时需要的最小输入。
type SendMessageInput struct {
	SessionID string
	Content   string
}

// NewSessionService 创建会话服务，并按需注入 Agent 回复能力。
func NewSessionService(repository repository.SessionRepository, agentServices ...*AgentService) *SessionService {
	var agentService *AgentService
	if len(agentServices) > 0 {
		agentService = agentServices[0]
	}

	return &SessionService{
		repository:   repository,
		agentService: agentService,
	}
}

// SetMemoryRefresher 注入可选的长期记忆刷新器。
func (s *SessionService) SetMemoryRefresher(refresher SessionMemoryRefresher) {
	s.refresher = refresher
}

// SetDeleteCleaners 注入删除会话后需要执行的派生数据清理器。
func (s *SessionService) SetDeleteCleaners(cleaners ...SessionDeleteCleaner) {
	s.deleteCleaners = append([]SessionDeleteCleaner(nil), cleaners...)
}

// CreateSession 创建并保存一个新会话。
func (s *SessionService) CreateSession(ctx context.Context, input CreateSessionInput, now time.Time) (*model.Session, error) {
	if !isValidChannel(input.Channel) {
		return nil, ErrInvalidChannel
	}
	if strings.TrimSpace(input.UserID) == "" {
		return nil, ErrUnauthorized
	}

	session := model.NewSession(generateSessionID(now), strings.TrimSpace(input.UserID), input.Channel, now)
	if err := s.repository.Save(ctx, session); err != nil {
		return nil, err
	}
	if err := s.refreshSessionMemoryIfNeeded(ctx, session.ID, now); err != nil {
		return nil, err
	}

	return session, nil
}

// ListSessions 返回指定用户的会话列表。
func (s *SessionService) ListSessions(ctx context.Context, userID string) ([]*model.Session, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrUnauthorized
	}

	return s.repository.ListByUserID(ctx, strings.TrimSpace(userID))
}

// GetSessionForUser 按 ID 读取指定用户拥有的会话。
func (s *SessionService) GetSessionForUser(ctx context.Context, userID string, sessionID string) (*model.Session, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrUnauthorized
	}

	session, err := s.repository.Load(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, err
	}
	if session.UserID != strings.TrimSpace(userID) {
		return nil, ErrSessionForbidden
	}

	return session, nil
}

// DeleteSession 删除指定用户拥有的会话。
func (s *SessionService) DeleteSession(ctx context.Context, userID string, sessionID string) error {
	if _, err := s.GetSessionForUser(ctx, userID, sessionID); err != nil {
		return err
	}

	cleanSessionID := strings.TrimSpace(sessionID)
	if err := s.repository.Delete(ctx, cleanSessionID); err != nil {
		return err
	}
	s.cleanupDeletedSession(ctx, cleanSessionID)
	return nil
}

// SendMessage 将用户消息写入会话，并通过 AgentService 或 mock 逻辑生成回复。
func (s *SessionService) SendMessage(ctx context.Context, userID string, userName string, input SendMessageInput, now time.Time) (*model.Session, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" || strings.TrimSpace(userID) == "" || strings.TrimSpace(userName) == "" {
		return nil, ErrInvalidMessage
	}

	session, err := s.GetSessionForUser(ctx, userID, input.SessionID)
	if err != nil {
		return nil, err
	}

	user := model.SessionUser{
		ID:   strings.TrimSpace(userID),
		Name: strings.TrimSpace(userName),
	}
	session.AppendUserMessage(user, content, now)

	if err := s.repository.Save(ctx, session); err != nil {
		return nil, err
	}

	reply := buildMockReply(content)
	if s.agentService != nil {
		result, err := s.agentService.Reply(ctx, AgentReplyInput{
			SessionID:    session.ID,
			SystemPrompt: agentprompt.DefaultSystemPrompt,
			UserMessage:  content,
		})
		if err != nil {
			return nil, err
		}
		reply = result.Reply
	}

	session.AppendAgentMessage(model.SessionUser{ID: "agent", Name: "DM Agent"}, reply, now)

	if err := s.repository.Save(ctx, session); err != nil {
		return nil, err
	}

	if err := s.refreshSessionMemoryIfNeeded(ctx, session.ID, now); err != nil {
		return nil, err
	}

	return session, nil
}

// buildMockReply 生成最小可用的 mock 回复内容。
func buildMockReply(content string) string {
	return "收到你的消息了：" + content
}

// generateSessionID 生成最小可用的会话 ID，后续可替换为更稳定的 ID 方案。
func generateSessionID(now time.Time) string {
	return fmt.Sprintf("session-%d", now.UnixNano())
}

// isValidChannel 校验当前是否为系统支持的接入渠道。
func isValidChannel(channel model.Channel) bool {
	return channel == model.ChannelWeb || channel == model.ChannelBot
}

func (s *SessionService) refreshSessionMemoryIfNeeded(ctx context.Context, sessionID string, now time.Time) error {
	if s.refresher == nil {
		return nil
	}
	return s.refresher.RefreshIfNeeded(ctx, sessionID, now)
}

func (s *SessionService) cleanupDeletedSession(ctx context.Context, sessionID string) {
	for _, cleaner := range s.deleteCleaners {
		if cleaner == nil {
			continue
		}
		_ = cleaner.DeleteBySessionID(ctx, sessionID)
	}
}
