package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository/memory"
	"DND-AI-BOT/internal/service"
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
