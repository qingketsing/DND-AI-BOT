package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/ratelimit"
	"DND-AI-BOT/internal/repository/memory"
	"DND-AI-BOT/internal/service"
	"DND-AI-BOT/internal/transport/http/dto"
	"DND-AI-BOT/internal/transport/http/middleware"
)

func TestCreateSessionReturnsCreatedSession(t *testing.T) {
	handler := newTestSessionHandler()

	request := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{"channel":"web"}`))
	request = request.WithContext(middleware.WithAuthenticatedUser(request.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	recorder := httptest.NewRecorder()

	handler.CreateSession(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if response["id"] == "" {
		t.Fatal("expected response to contain session id")
	}
	if response["channel"] != "web" {
		t.Fatalf("expected response channel web, got %v", response["channel"])
	}
	if response["user_id"] != "user-1" {
		t.Fatalf("expected response user_id user-1, got %v", response["user_id"])
	}
}

func TestGetSessionReturnsSessionAndMissingReturnsNotFound(t *testing.T) {
	repository := memory.NewSessionRepository()
	sessionService := service.NewSessionService(repository)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := sessionService.CreateSession(ctx, service.CreateSessionInput{UserID: "user-1", Channel: model.ChannelWeb}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}
	handler := NewSessionHandler(sessionService)

	request := httptest.NewRequest(http.MethodGet, "/sessions/"+session.ID, nil)
	request = request.WithContext(middleware.WithAuthenticatedUser(request.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	recorder := httptest.NewRecorder()
	handler.GetSession(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	missingRequest := httptest.NewRequest(http.MethodGet, "/sessions/missing", nil)
	missingRequest = missingRequest.WithContext(middleware.WithAuthenticatedUser(missingRequest.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	missingRecorder := httptest.NewRecorder()
	handler.GetSession(missingRecorder, missingRequest)
	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, missingRecorder.Code)
	}
}

func TestSendMessageReturnsUpdatedSession(t *testing.T) {
	repository := memory.NewSessionRepository()
	sessionService := service.NewSessionService(repository)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := sessionService.CreateSession(ctx, service.CreateSessionInput{UserID: "user-1", Channel: model.ChannelWeb}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}
	handler := NewSessionHandler(sessionService)

	body := `{"content":"hello"}`
	request := httptest.NewRequest(http.MethodPost, "/sessions/"+session.ID+"/messages", strings.NewReader(body))
	request = request.WithContext(middleware.WithAuthenticatedUser(request.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	recorder := httptest.NewRecorder()

	handler.SendMessage(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		History []struct {
			Source string `json:"source"`
		} `json:"history"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if len(response.History) != 2 {
		t.Fatalf("expected 2 history records, got %d", len(response.History))
	}
	if response.History[0].Source != "user" || response.History[1].Source != "agent" {
		t.Fatalf("expected user and agent sources, got %+v", response.History)
	}
}

func TestSendMessageReturnsAcceptedWhenAsyncServiceConfigured(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	sessionService := service.NewSessionService(sessions)
	asyncService := service.NewAsyncMessageService(sessions, jobs)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := sessionService.CreateSession(ctx, service.CreateSessionInput{UserID: "user-1", Channel: model.ChannelWeb}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}
	handler := NewSessionHandler(sessionService, WithAsyncMessageService(asyncService))

	body := `{"content":"hello"}`
	request := httptest.NewRequest(http.MethodPost, "/sessions/"+session.ID+"/messages", strings.NewReader(body))
	request = request.WithContext(middleware.WithAuthenticatedUser(request.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	recorder := httptest.NewRecorder()

	handler.SendMessage(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, recorder.Code)
	}

	var response struct {
		MessageID string `json:"message_id"`
		JobID     string `json:"job_id"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if response.MessageID == "" || response.JobID == "" {
		t.Fatalf("expected message and job ids, got %+v", response)
	}
	if response.Status != "queued" {
		t.Fatalf("expected queued status, got %q", response.Status)
	}
}

func TestSessionHandlerSendMessageAcceptedCanBeFetchedViaGetMessage(t *testing.T) {
	sessions, jobs := memory.NewAsyncMessageRepositories()
	sessionService := service.NewSessionService(sessions)
	asyncService := service.NewAsyncMessageService(sessions, jobs)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := sessionService.CreateSession(ctx, service.CreateSessionInput{UserID: "user-1", Channel: model.ChannelWeb}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}
	handler := NewSessionHandler(sessionService, WithAsyncMessageService(asyncService))

	sendRequest := httptest.NewRequest(http.MethodPost, "/sessions/"+session.ID+"/messages", strings.NewReader(`{"content":"hello"}`))
	sendRequest = sendRequest.WithContext(middleware.WithAuthenticatedUser(sendRequest.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	sendRecorder := httptest.NewRecorder()

	handler.SendMessage(sendRecorder, sendRequest)

	if sendRecorder.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, sendRecorder.Code)
	}

	var sendResponse struct {
		MessageID string `json:"message_id"`
		JobID     string `json:"job_id"`
	}
	if err := json.Unmarshal(sendRecorder.Body.Bytes(), &sendResponse); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/messages/"+sendResponse.MessageID, nil)
	getRequest = getRequest.WithContext(middleware.WithAuthenticatedUser(getRequest.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	getRecorder := httptest.NewRecorder()

	handler.GetMessage(getRecorder, getRequest)

	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, getRecorder.Code)
	}

	var getResponse struct {
		MessageID string `json:"message_id"`
		Status    string `json:"status"`
		Job       struct {
			ID string `json:"id"`
		} `json:"job"`
	}
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &getResponse); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if getResponse.MessageID != sendResponse.MessageID {
		t.Fatalf("expected message id %q, got %q", sendResponse.MessageID, getResponse.MessageID)
	}
	if getResponse.Job.ID != sendResponse.JobID {
		t.Fatalf("expected job id %q, got %q", sendResponse.JobID, getResponse.Job.ID)
	}
	if getResponse.Status != "queued" {
		t.Fatalf("expected queued status, got %q", getResponse.Status)
	}
}

func TestGetMessageReturnsAsyncStatus(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	sessionService := service.NewSessionService(sessions)
	asyncService := service.NewAsyncMessageService(sessions, jobs)
	handler := NewSessionHandler(sessionService, WithAsyncMessageService(asyncService))

	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	record := session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))
	if err := sessions.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}
	if err := jobs.Create(context.Background(), model.MessageJob{
		ID:        "job-1",
		MessageID: record.ID,
		SessionID: "session-1",
		UserID:    "user-1",
		Status:    model.MessageJobQueued,
		QueuedAt:  now.Add(time.Minute),
		CreatedAt: now.Add(time.Minute),
		UpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("expected job create to succeed, got %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/messages/"+record.ID, nil)
	request = request.WithContext(middleware.WithAuthenticatedUser(request.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	recorder := httptest.NewRecorder()

	handler.GetMessage(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		MessageID string `json:"message_id"`
		Status    string `json:"status"`
		Job       struct {
			ID string `json:"id"`
		} `json:"job"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if response.MessageID != record.ID {
		t.Fatalf("expected message id %q, got %q", record.ID, response.MessageID)
	}
	if response.Status != "queued" {
		t.Fatalf("expected queued status, got %q", response.Status)
	}
	if response.Job.ID != "job-1" {
		t.Fatalf("expected job id job-1, got %q", response.Job.ID)
	}
}

func TestSessionHandlerGetMessageDoesNotReturnImplicitAssistantReply(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	sessionService := service.NewSessionService(sessions)
	asyncService := service.NewAsyncMessageService(sessions, jobs)
	handler := NewSessionHandler(sessionService, WithAsyncMessageService(asyncService))

	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	record := session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))
	session.AppendAgentMessage(model.SessionUser{ID: "agent", Name: "DM Agent"}, "unrelated reply", now.Add(2*time.Minute))
	if err := sessions.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}
	if err := jobs.Create(context.Background(), model.MessageJob{
		ID:        "job-1",
		MessageID: record.ID,
		SessionID: "session-1",
		UserID:    "user-1",
		Status:    model.MessageJobCompleted,
		QueuedAt:  now.Add(time.Minute),
		CreatedAt: now.Add(time.Minute),
		UpdatedAt: now.Add(3 * time.Minute),
	}); err != nil {
		t.Fatalf("expected job create to succeed, got %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/messages/"+record.ID, nil)
	request = request.WithContext(middleware.WithAuthenticatedUser(request.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	recorder := httptest.NewRecorder()

	handler.GetMessage(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if _, ok := response["assistant_reply"]; ok {
		t.Fatalf("expected assistant_reply to be omitted without explicit link, got %+v", response["assistant_reply"])
	}
}

func TestSessionHandlerAssistantReplyRecordExposesExplicitAssociationFields(t *testing.T) {
	replyType := reflect.TypeOf(dto.AssistantReplyRecord{})
	for _, fieldName := range []string{"ReplyToMessageID", "SourceJobID"} {
		field, ok := replyType.FieldByName(fieldName)
		if !ok {
			t.Fatalf("expected AssistantReplyRecord to expose %s", fieldName)
		}
		if field.Type.Kind() != reflect.String {
			t.Fatalf("expected AssistantReplyRecord.%s to be a string, got %s", fieldName, field.Type)
		}
	}
}

func TestSessionHandlerGetMessageReturnsExplicitAssistantReplyFields(t *testing.T) {
	sessions := memory.NewSessionRepository()
	jobs := memory.NewMessageJobRepository()
	sessionService := service.NewSessionService(sessions)
	asyncService := service.NewAsyncMessageService(sessions, jobs)
	handler := NewSessionHandler(sessionService, WithAsyncMessageService(asyncService))

	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session := model.NewSession("session-1", "user-1", model.ChannelWeb, now)
	record := session.AppendUserMessage(model.SessionUser{ID: "user-1", Name: "Alice"}, "hello", now.Add(time.Minute))
	session.AppendAssistantReply(
		model.SessionUser{ID: "agent", Name: "DM Agent"},
		"world",
		record.ID,
		"job-1",
		now.Add(2*time.Minute),
	)
	if err := sessions.Save(context.Background(), session); err != nil {
		t.Fatalf("expected session save to succeed, got %v", err)
	}
	if err := jobs.Create(context.Background(), model.MessageJob{
		ID:        "job-1",
		MessageID: record.ID,
		SessionID: "session-1",
		UserID:    "user-1",
		Status:    model.MessageJobCompleted,
		QueuedAt:  now.Add(time.Minute),
		CreatedAt: now.Add(time.Minute),
		UpdatedAt: now.Add(3 * time.Minute),
	}); err != nil {
		t.Fatalf("expected job create to succeed, got %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/messages/"+record.ID, nil)
	request = request.WithContext(middleware.WithAuthenticatedUser(request.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	recorder := httptest.NewRecorder()

	handler.GetMessage(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		AssistantReply struct {
			ReplyToMessageID string `json:"reply_to_message_id"`
			SourceJobID      string `json:"source_job_id"`
		} `json:"assistant_reply"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if response.AssistantReply.ReplyToMessageID != record.ID {
		t.Fatalf("expected reply_to_message_id %q, got %q", record.ID, response.AssistantReply.ReplyToMessageID)
	}
	if response.AssistantReply.SourceJobID != "job-1" {
		t.Fatalf("expected source_job_id job-1, got %q", response.AssistantReply.SourceJobID)
	}
}

func TestSendMessageReturnsTooManyRequestsWhenRateLimited(t *testing.T) {
	repository := memory.NewSessionRepository()
	sessionService := service.NewSessionService(repository)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := sessionService.CreateSession(ctx, service.CreateSessionInput{UserID: "user-1", Channel: model.ChannelWeb}, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}
	handler := NewSessionHandler(
		sessionService,
		WithSessionRateLimiter(ratelimit.NewService(
			&fakeSessionRateLimitBackend{
				decision: ratelimit.Decision{
					Allowed:    false,
					PolicyName: "message_user",
					RetryAfter: 15 * time.Second,
				},
			},
			ratelimit.DefaultConfig(),
			fakeSessionRateLimitClock{now: now},
		)),
	)

	body := `{"content":"hello"}`
	request := httptest.NewRequest(http.MethodPost, "/sessions/"+session.ID+"/messages", strings.NewReader(body))
	request = request.WithContext(middleware.WithAuthenticatedUser(request.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	recorder := httptest.NewRecorder()

	handler.SendMessage(recorder, request)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, recorder.Code)
	}
	if recorder.Header().Get("Retry-After") != "15" {
		t.Fatalf("expected Retry-After 15, got %q", recorder.Header().Get("Retry-After"))
	}
	assertErrorCode(t, recorder, "rate_limited")
}

func TestListSessionsReturnsCurrentUserSessions(t *testing.T) {
	repository := memory.NewSessionRepository()
	sessionService := service.NewSessionService(repository)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	_, _ = sessionService.CreateSession(ctx, service.CreateSessionInput{UserID: "user-1", Channel: model.ChannelWeb}, now)
	_, _ = sessionService.CreateSession(ctx, service.CreateSessionInput{UserID: "user-2", Channel: model.ChannelWeb}, now.Add(time.Minute))
	_, _ = sessionService.CreateSession(ctx, service.CreateSessionInput{UserID: "user-1", Channel: model.ChannelBot}, now.Add(2*time.Minute))
	handler := NewSessionHandler(sessionService)

	request := httptest.NewRequest(http.MethodGet, "/sessions", nil)
	request = request.WithContext(middleware.WithAuthenticatedUser(request.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	recorder := httptest.NewRecorder()

	handler.ListSessions(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}
	if len(response.Items) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(response.Items))
	}
}

func TestGetSessionReturnsForbiddenForDifferentOwner(t *testing.T) {
	repository := memory.NewSessionRepository()
	sessionService := service.NewSessionService(repository)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, _ := sessionService.CreateSession(ctx, service.CreateSessionInput{UserID: "user-1", Channel: model.ChannelWeb}, now)
	handler := NewSessionHandler(sessionService)

	request := httptest.NewRequest(http.MethodGet, "/sessions/"+session.ID, nil)
	request = request.WithContext(middleware.WithAuthenticatedUser(request.Context(), middleware.AuthenticatedUser{
		UserID:      "user-2",
		DisplayName: "Bob",
	}))
	recorder := httptest.NewRecorder()

	handler.GetSession(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}
}

func TestDeleteSessionReturnsSuccess(t *testing.T) {
	repository := memory.NewSessionRepository()
	sessionService := service.NewSessionService(repository)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, _ := sessionService.CreateSession(ctx, service.CreateSessionInput{UserID: "user-1", Channel: model.ChannelWeb}, now)
	handler := NewSessionHandler(sessionService)

	request := httptest.NewRequest(http.MethodDelete, "/sessions/"+session.ID, nil)
	request = request.WithContext(middleware.WithAuthenticatedUser(request.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	recorder := httptest.NewRecorder()

	handler.DeleteSession(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestDeleteSessionReturnsForbiddenForDifferentOwner(t *testing.T) {
	repository := memory.NewSessionRepository()
	sessionService := service.NewSessionService(repository)
	ctx := context.Background()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, _ := sessionService.CreateSession(ctx, service.CreateSessionInput{UserID: "user-1", Channel: model.ChannelWeb}, now)
	handler := NewSessionHandler(sessionService)

	request := httptest.NewRequest(http.MethodDelete, "/sessions/"+session.ID, nil)
	request = request.WithContext(middleware.WithAuthenticatedUser(request.Context(), middleware.AuthenticatedUser{
		UserID:      "user-2",
		DisplayName: "Bob",
	}))
	recorder := httptest.NewRecorder()

	handler.DeleteSession(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}
}

func newTestSessionHandler() *SessionHandler {
	repository := memory.NewSessionRepository()
	service := service.NewSessionService(repository)
	return NewSessionHandler(service)
}

func TestCreateSessionRejectsInvalidChannel(t *testing.T) {
	handler := newTestSessionHandler()

	request := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{"channel":"desktop"}`))
	request = request.WithContext(middleware.WithAuthenticatedUser(request.Context(), middleware.AuthenticatedUser{
		UserID:      "user-1",
		DisplayName: "Alice",
	}))
	recorder := httptest.NewRecorder()

	handler.CreateSession(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

type fakeSessionRateLimitBackend struct {
	decision ratelimit.Decision
}

func (f *fakeSessionRateLimitBackend) Allow(ctx context.Context, key string, policy ratelimit.Policy, now time.Time) (ratelimit.Decision, error) {
	decision := f.decision
	decision.Key = key
	decision.PolicyName = policy.Name
	decision.Limit = policy.Limit
	return decision, nil
}

type fakeSessionRateLimitClock struct {
	now time.Time
}

func (c fakeSessionRateLimitClock) Now() time.Time {
	return c.now
}
