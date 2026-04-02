package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"../../../model"
	"../../../repository/memory"
	"../../../service"
)

func TestCreateSessionReturnsCreatedSession(t *testing.T) {
	handler := newTestSessionHandler()

	request := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{"channel":"web"}`))
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
}

func TestGetSessionReturnsSessionAndMissingReturnsNotFound(t *testing.T) {
	repository := memory.NewSessionRepository()
	service := service.NewSessionService(repository)
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := service.CreateSession(model.ChannelWeb, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}
	handler := NewSessionHandler(service)

	request := httptest.NewRequest(http.MethodGet, "/sessions/"+session.ID, nil)
	recorder := httptest.NewRecorder()
	handler.GetSession(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	missingRequest := httptest.NewRequest(http.MethodGet, "/sessions/missing", nil)
	missingRecorder := httptest.NewRecorder()
	handler.GetSession(missingRecorder, missingRequest)
	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, missingRecorder.Code)
	}
}

func TestSendMessageReturnsUpdatedSession(t *testing.T) {
	repository := memory.NewSessionRepository()
	service := service.NewSessionService(repository)
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	session, err := service.CreateSession(model.ChannelWeb, now)
	if err != nil {
		t.Fatalf("expected create session to succeed, got %v", err)
	}
	handler := NewSessionHandler(service)

	body := `{"user_id":"user-1","user_name":"Alice","content":"hello"}`
	request := httptest.NewRequest(http.MethodPost, "/sessions/"+session.ID+"/messages", strings.NewReader(body))
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

func newTestSessionHandler() *SessionHandler {
	repository := memory.NewSessionRepository()
	service := service.NewSessionService(repository)
	return NewSessionHandler(service)
}

func TestCreateSessionRejectsInvalidChannel(t *testing.T) {
	handler := newTestSessionHandler()

	request := httptest.NewRequest(http.MethodPost, "/sessions", strings.NewReader(`{"channel":"desktop"}`))
	recorder := httptest.NewRecorder()

	handler.CreateSession(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}
