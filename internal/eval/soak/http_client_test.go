package soak

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGameHTTPClientSendMessageReturnsLatestAgentReply(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.String() != "http://example.test/sessions/session-1/messages" {
			t.Fatalf("expected session message URL, got %s", r.URL.String())
		}
		if cookie, err := r.Cookie("dnd_auth_session"); err != nil || cookie.Value != "session-token" {
			t.Fatalf("expected auth cookie session-token, got cookie=%v err=%v", cookie, err)
		}
		return jsonResponse(http.StatusOK, `{
			"id":"session-1",
			"history":[
				{"source":"user","message":{"content":"你好"}},
				{"source":"agent","message":{"content":"你好，冒险者"}}
			]
		}`), nil
	})

	client := NewGameHTTPClient("http://example.test", "session-token", &http.Client{Transport: transport})
	result, err := client.SendMessage(context.Background(), "session-1", "你好")
	if err != nil {
		t.Fatalf("expected send message to succeed, got %v", err)
	}

	if result.AgentReply != "你好，冒险者" {
		t.Fatalf("expected latest agent reply, got %q", result.AgentReply)
	}
	if result.HTTPStatus != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.HTTPStatus)
	}
	if result.LatencyMS < 0 {
		t.Fatalf("expected non-negative latency, got %d", result.LatencyMS)
	}
}

func TestGameHTTPClientSendMessageReturnsErrorForNon2xx(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnauthorized, `{"error":{"code":"unauthorized"}}`), nil
	})

	client := NewGameHTTPClient("http://example.test", "session-token", &http.Client{Transport: transport})
	_, err := client.SendMessage(context.Background(), "session-1", "你好")
	if err == nil {
		t.Fatal("expected non-2xx response to return error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected status in error, got %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}
