package soak

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"DND-AI-BOT/internal/transport/http/dto"
)

const authCookieName = "dnd_auth_session"

// MessageResult is the observable result of one backend message call.
type MessageResult struct {
	AgentReply string `json:"agent_reply"`
	HTTPStatus int    `json:"http_status"`
	LatencyMS  int64  `json:"latency_ms"`
	RawBody    string `json:"raw_body"`
}

// GameHTTPClient calls the real game backend message API.
type GameHTTPClient struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewGameHTTPClient creates a backend HTTP client for soak evaluation.
func NewGameHTTPClient(baseURL string, token string, httpClient *http.Client) *GameHTTPClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &GameHTTPClient{
		baseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		authToken:  strings.TrimSpace(token),
		httpClient: httpClient,
	}
}

// SendMessage posts a user message and extracts the latest agent reply.
func (c *GameHTTPClient) SendMessage(ctx context.Context, sessionID string, content string) (MessageResult, error) {
	startedAt := time.Now()
	body, err := json.Marshal(dto.SendMessageRequest{Content: content})
	if err != nil {
		return MessageResult{}, err
	}

	endpoint := c.baseURL + "/sessions/" + url.PathEscape(sessionID) + "/messages"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return MessageResult{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: c.authToken})
		request.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return MessageResult{}, err
	}
	defer response.Body.Close()

	rawBody, readErr := io.ReadAll(response.Body)
	result := MessageResult{
		HTTPStatus: response.StatusCode,
		LatencyMS:  time.Since(startedAt).Milliseconds(),
		RawBody:    string(rawBody),
	}
	if readErr != nil {
		return result, readErr
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return result, fmt.Errorf("send message failed with status %d: %s", response.StatusCode, result.RawBody)
	}

	var session dto.SessionResponse
	if err := json.Unmarshal(rawBody, &session); err != nil {
		return result, err
	}
	reply := latestAgentReply(session.History)
	if reply == "" {
		return result, fmt.Errorf("agent reply not found in session response")
	}
	result.AgentReply = reply
	return result, nil
}

func latestAgentReply(history []dto.HistoryRecordDTO) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Source == "agent" {
			return strings.TrimSpace(history[i].Message.Content)
		}
	}
	return ""
}
