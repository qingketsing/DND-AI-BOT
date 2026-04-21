package ratelimit

import (
	"context"
	"strings"
	"time"

	"DND-AI-BOT/internal/observability"

	"github.com/prometheus/client_golang/prometheus"
)

type Clock interface {
	Now() time.Time
}

type Service struct {
	limiter Limiter
	config  Config
	clock   Clock
	metrics *observability.Metrics
}

type ServiceOption func(*Service)

type CheckInput struct {
	IP        string
	Email     string
	UserID    string
	SessionID string
}

func NewService(limiter Limiter, config Config, clock Clock, options ...ServiceOption) *Service {
	service := &Service{
		limiter: limiter,
		config:  config,
		clock:   clock,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
}

func WithMetrics(metrics *observability.Metrics) ServiceOption {
	return func(service *Service) {
		service.metrics = metrics
	}
}

func (s *Service) CheckLogin(ctx context.Context, input CheckInput) error {
	return s.check(ctx, "login", []checkSpec{
		{scope: "ip", key: "login:ip:" + normalizeKeyPart(input.IP), policy: Policy{Name: "login_ip", Limit: s.config.LoginIPLimit, Window: s.config.LoginIPWindow}},
		{scope: "account", key: "login:account:" + normalizeEmail(input.Email), policy: Policy{Name: "login_account", Limit: s.config.LoginAccountLimit, Window: s.config.LoginAccountWindow}},
	})
}

func (s *Service) CheckRegister(ctx context.Context, input CheckInput) error {
	return s.check(ctx, "register", []checkSpec{
		{scope: "ip", key: "register:ip:" + normalizeKeyPart(input.IP), policy: Policy{Name: "register_ip", Limit: s.config.RegisterIPLimit, Window: s.config.RegisterIPWindow}},
		{scope: "email", key: "register:email:" + normalizeEmail(input.Email), policy: Policy{Name: "register_email", Limit: s.config.RegisterEmailLimit, Window: s.config.RegisterEmailWindow}},
	})
}

func (s *Service) CheckMessage(ctx context.Context, input CheckInput) error {
	return s.check(ctx, "message", []checkSpec{
		{scope: "user", key: "message:user:" + normalizeKeyPart(input.UserID), policy: Policy{Name: "message_user", Limit: s.config.MessageUserLimit, Window: s.config.MessageUserWindow}},
		{scope: "session", key: "message:session:" + normalizeKeyPart(input.SessionID), policy: Policy{Name: "message_session", Limit: s.config.MessageSessionLimit, Window: s.config.MessageSessionWindow}},
		{scope: "ip", key: "message:ip:" + normalizeKeyPart(input.IP), policy: Policy{Name: "message_ip", Limit: s.config.MessageIPLimit, Window: s.config.MessageIPWindow}},
	})
}

type checkSpec struct {
	scope  string
	key    string
	policy Policy
}

func (s *Service) check(ctx context.Context, endpoint string, specs []checkSpec) error {
	if s == nil || !s.config.Enabled || s.limiter == nil {
		return nil
	}
	now := time.Now().UTC()
	if s.clock != nil {
		now = s.clock.Now()
	}
	for _, spec := range specs {
		if !isPolicyEnabled(spec.policy) || strings.HasSuffix(spec.key, ":") {
			continue
		}
		decision, err := s.limiter.Allow(ctx, spec.key, spec.policy, now)
		if err != nil {
			s.record(endpoint, spec.scope, "error")
			return err
		}
		if !decision.Allowed {
			s.record(endpoint, spec.scope, "blocked")
			return &RateLimitError{Decision: decision}
		}
		s.record(endpoint, spec.scope, "allowed")
	}
	return nil
}

func (s *Service) record(endpoint string, scope string, status string) {
	if s == nil || s.metrics == nil || s.metrics.RateLimitChecksTotal == nil {
		return
	}
	labels := prometheus.Labels{"endpoint": endpoint, "scope": scope, "status": status}
	s.metrics.RateLimitChecksTotal.With(labels).Inc()
	if status == "blocked" && s.metrics.RateLimitBlockedTotal != nil {
		s.metrics.RateLimitBlockedTotal.With(prometheus.Labels{"endpoint": endpoint, "scope": scope}).Inc()
	}
	if status == "error" && s.metrics.RateLimitErrorsTotal != nil {
		s.metrics.RateLimitErrorsTotal.With(prometheus.Labels{"endpoint": endpoint, "scope": scope}).Inc()
	}
}

func isPolicyEnabled(policy Policy) bool {
	return strings.TrimSpace(policy.Name) != "" && policy.Limit > 0 && policy.Window > 0
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeKeyPart(value string) string {
	return strings.TrimSpace(value)
}
