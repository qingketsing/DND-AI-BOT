package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

var (
	// ErrInvalidMessage 表示消息内容为空或只包含空白字符。
	ErrInvalidMessage = errors.New("invalid message")
	// ErrInvalidChannel 表示传入了系统暂不支持的接入渠道。
	ErrInvalidChannel = errors.New("invalid channel")
)

// SessionService 负责编排会话创建、读取和消息收发流程。
type SessionService struct {
	repository repository.SessionRepository
}

// SendMessageInput 定义发送消息时需要的最小输入。
type SendMessageInput struct {
	SessionID string
	UserID    string
	UserName  string
	Content   string
}

// NewSessionService 创建会话服务。
func NewSessionService(repository repository.SessionRepository) *SessionService {
	return &SessionService{repository: repository}
}

// CreateSession 创建并保存一个新会话。
func (s *SessionService) CreateSession(ctx context.Context, channel model.Channel, now time.Time) (*model.Session, error) {
	if !isValidChannel(channel) {
		return nil, ErrInvalidChannel
	}

	session := model.NewSession(generateSessionID(now), channel, now)
	if err := s.repository.Save(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession 按 ID 读取会话。
func (s *SessionService) GetSession(ctx context.Context, id string) (*model.Session, error) {
	return s.repository.Load(ctx, id)
}

// SendMessage 将用户消息写入会话，并追加一条 mock agent 回复。
func (s *SessionService) SendMessage(ctx context.Context, input SendMessageInput, now time.Time) (*model.Session, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" || strings.TrimSpace(input.UserID) == "" || strings.TrimSpace(input.UserName) == "" {
		return nil, ErrInvalidMessage
	}

	session, err := s.repository.Load(ctx, strings.TrimSpace(input.SessionID))
	if err != nil {
		return nil, err
	}

	user := model.User{
		ID:   strings.TrimSpace(input.UserID),
		Name: strings.TrimSpace(input.UserName),
	}
	session.AppendUserMessage(user, content, now)
	session.AppendAgentMessage(model.User{ID: "agent", Name: "DM Agent"}, buildMockReply(content), now)

	if err := s.repository.Save(ctx, session); err != nil {
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
