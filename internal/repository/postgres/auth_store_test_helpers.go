package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"DND-AI-BOT/internal/model"
)

type fakePGState struct {
	mu           sync.Mutex
	users        map[string]model.User
	emails       map[string]string
	sessions     map[string]model.AuthSession
	tokens       map[string]string
	gameSessions map[string]model.Session
	messageAsync map[string]fakeSessionMessageAsyncFields
	memories     map[string]model.SessionMemory
	messageJobs  map[string]model.MessageJob
	outboxEvents map[string]model.OutboxEvent
	indexMeta    map[string]fakeKnowledgeIndexMetadata
	knowledge    map[string]fakeKnowledgeChunk
}

type fakeSessionMessageAsyncFields struct {
	SessionID        string
	SourceJobID      string
	ReplyToMessageID string
}

type fakeKnowledgeIndexMetadata struct {
	KnowledgeBase     string
	EmbeddingProvider string
	EmbeddingModel    string
	EmbeddingDim      int
	BuiltAt           time.Time
}

type fakeKnowledgeChunk struct {
	ID            string
	KnowledgeBase string
	SourceID      string
	Title         string
	Aliases       []string
	Content       string
	Metadata      []byte
	Embedding     []float32
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func newFakePGState() *fakePGState {
	return &fakePGState{
		users:        make(map[string]model.User),
		emails:       make(map[string]string),
		sessions:     make(map[string]model.AuthSession),
		tokens:       make(map[string]string),
		gameSessions: make(map[string]model.Session),
		messageAsync: make(map[string]fakeSessionMessageAsyncFields),
		memories:     make(map[string]model.SessionMemory),
		messageJobs:  make(map[string]model.MessageJob),
		outboxEvents: make(map[string]model.OutboxEvent),
		indexMeta:    make(map[string]fakeKnowledgeIndexMetadata),
		knowledge:    make(map[string]fakeKnowledgeChunk),
	}
}

func NewFakeKnowledgePGState() *fakePGState {
	return newFakePGState()
}

func newFakePGDB(t *testing.T, state *fakePGState) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("fake-pg-%d", time.Now().UnixNano())
	sql.Register(name, &fakePGDriver{state: state})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("expected sql.Open to succeed, got %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func NewFakeKnowledgePGDB(t *testing.T, state *fakePGState) *sql.DB {
	return newFakePGDB(t, state)
}

type fakePGDriver struct {
	state *fakePGState
}

func (d *fakePGDriver) Open(name string) (driver.Conn, error) {
	return &fakePGConn{state: d.state}, nil
}

type fakePGConn struct {
	state *fakePGState
}

func (c *fakePGConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}
func (c *fakePGConn) Close() error              { return nil }
func (c *fakePGConn) Begin() (driver.Tx, error) { return &fakePGTx{}, nil }

func (c *fakePGConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	switch {
	case strings.Contains(query, "INSERT INTO users"):
		user := model.User{
			ID:           args[0].Value.(string),
			Email:        args[1].Value.(string),
			PasswordHash: args[2].Value.(string),
			DisplayName:  args[3].Value.(string),
			Status:       model.UserStatus(args[4].Value.(string)),
			CreatedAt:    args[5].Value.(time.Time),
			UpdatedAt:    args[6].Value.(time.Time),
		}
		if len(args) > 7 {
			if args[7].Value != nil {
				v := args[7].Value.(time.Time)
				user.LastLoginAt = &v
			}
		}
		c.state.users[user.ID] = user
		c.state.emails[user.Email] = user.ID
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "INSERT INTO auth_sessions"):
		session := model.AuthSession{
			ID:        args[0].Value.(string),
			UserID:    args[1].Value.(string),
			TokenHash: args[2].Value.(string),
			ExpiresAt: args[3].Value.(time.Time),
			CreatedAt: args[4].Value.(time.Time),
			UpdatedAt: args[5].Value.(time.Time),
		}
		if len(args) > 6 && args[6].Value != nil {
			v := args[6].Value.(time.Time)
			session.LastSeenAt = &v
		}
		if len(args) > 7 && args[7].Value != nil {
			v := args[7].Value.(string)
			session.UserAgent = &v
		}
		if len(args) > 8 && args[8].Value != nil {
			v := args[8].Value.(string)
			session.IPAddress = &v
		}
		if len(args) > 9 && args[9].Value != nil {
			v := args[9].Value.(time.Time)
			session.RevokedAt = &v
		}
		c.state.sessions[session.ID] = session
		c.state.tokens[session.TokenHash] = session.ID
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "UPDATE auth_sessions"):
		sessionID := args[0].Value.(string)
		now := args[1].Value.(time.Time)
		session, ok := c.state.sessions[sessionID]
		if !ok {
			return driver.RowsAffected(0), nil
		}
		session.RevokedAt = &now
		session.UpdatedAt = now
		c.state.sessions[sessionID] = session
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "INSERT INTO sessions"):
		session := model.Session{
			ID:        args[0].Value.(string),
			UserID:    args[1].Value.(string),
			Title:     args[2].Value.(string),
			Channel:   model.Channel(args[3].Value.(string)),
			CreatedAt: args[4].Value.(time.Time),
			UpdatedAt: args[5].Value.(time.Time),
		}
		existing := c.state.gameSessions[session.ID]
		session.History = existing.History
		c.state.gameSessions[session.ID] = session
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "INSERT INTO session_memories"):
		var events []string
		switch value := args[4].Value.(type) {
		case []byte:
			if err := json.Unmarshal(value, &events); err != nil {
				return nil, err
			}
		case string:
			if err := json.Unmarshal([]byte(value), &events); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unexpected recent_key_events type %T", value)
		}
		memory := model.SessionMemory{
			SessionID:        args[0].Value.(string),
			CharacterSummary: args[1].Value.(string),
			SceneSummary:     args[2].Value.(string),
			CurrentObjective: args[3].Value.(string),
			RecentKeyEvents:  events,
			UpdatedAt:        args[5].Value.(time.Time),
		}
		c.state.memories[memory.SessionID] = memory
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "INSERT INTO message_jobs"):
		job := model.MessageJob{
			ID:               args[0].Value.(string),
			MessageID:        args[1].Value.(string),
			SessionID:        args[2].Value.(string),
			UserID:           args[3].Value.(string),
			Status:           model.MessageJobStatus(args[4].Value.(string)),
			AttemptCount:     int(args[5].Value.(int64)),
			MaxAttempts:      int(args[6].Value.(int64)),
			WorkerID:         args[7].Value.(string),
			QueuedAt:         args[8].Value.(time.Time),
			LastErrorCode:    args[11].Value.(string),
			LastErrorMessage: args[12].Value.(string),
			LatencyMS:        args[13].Value.(int64),
			CreatedAt:        args[14].Value.(time.Time),
			UpdatedAt:        args[15].Value.(time.Time),
		}
		if args[9].Value != nil {
			value := args[9].Value.(time.Time)
			job.StartedAt = &value
		}
		if args[10].Value != nil {
			value := args[10].Value.(time.Time)
			job.FinishedAt = &value
		}
		c.state.messageJobs[job.ID] = job
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "INSERT INTO outbox_events"):
		var payload []byte
		switch value := args[4].Value.(type) {
		case []byte:
			payload = append([]byte(nil), value...)
		case string:
			payload = []byte(value)
		default:
			return nil, fmt.Errorf("unexpected outbox payload type %T", value)
		}
		event := model.OutboxEvent{
			ID:            args[0].Value.(string),
			AggregateType: args[1].Value.(string),
			AggregateID:   args[2].Value.(string),
			EventType:     args[3].Value.(string),
			PayloadJSON:   payload,
			Status:        model.OutboxEventStatus(args[5].Value.(string)),
			AttemptCount:  int(args[6].Value.(int64)),
			LastError:     args[7].Value.(string),
			CreatedAt:     args[8].Value.(time.Time),
			UpdatedAt:     args[10].Value.(time.Time),
		}
		if args[9].Value != nil {
			value := args[9].Value.(time.Time)
			event.PublishedAt = &value
		}
		c.state.outboxEvents[event.ID] = event
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "INSERT INTO knowledge_chunks"):
		var metadata []byte
		switch value := args[6].Value.(type) {
		case []byte:
			metadata = append([]byte(nil), value...)
		case string:
			metadata = []byte(value)
		default:
			return nil, fmt.Errorf("unexpected knowledge metadata type %T", value)
		}
		embedding, err := parseVectorLiteral(args[7].Value.(string))
		if err != nil {
			return nil, err
		}
		aliases, err := parseTextArrayLiteral(args[4].Value.(string))
		if err != nil {
			return nil, err
		}
		chunk := fakeKnowledgeChunk{
			ID:            args[0].Value.(string),
			KnowledgeBase: args[1].Value.(string),
			SourceID:      args[2].Value.(string),
			Title:         args[3].Value.(string),
			Aliases:       aliases,
			Content:       args[5].Value.(string),
			Metadata:      metadata,
			Embedding:     embedding,
			CreatedAt:     args[8].Value.(time.Time),
			UpdatedAt:     args[9].Value.(time.Time),
		}
		c.state.knowledge[chunk.ID] = chunk
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "INSERT INTO knowledge_index_metadata"):
		metadata := fakeKnowledgeIndexMetadata{
			KnowledgeBase:     args[0].Value.(string),
			EmbeddingProvider: args[1].Value.(string),
			EmbeddingModel:    args[2].Value.(string),
			EmbeddingDim:      int(args[3].Value.(int64)),
			BuiltAt:           args[4].Value.(time.Time),
		}
		c.state.indexMeta[metadata.KnowledgeBase] = metadata
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "UPDATE message_jobs") && strings.Contains(query, "attempt_count = attempt_count + 1"):
		jobID := args[0].Value.(string)
		job, ok := c.state.messageJobs[jobID]
		if !ok {
			return driver.RowsAffected(0), nil
		}
		job.AttemptCount++
		job.UpdatedAt = args[1].Value.(time.Time)
		c.state.messageJobs[jobID] = job
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "UPDATE message_jobs") && strings.Contains(query, "worker_id = $3"):
		jobID := args[0].Value.(string)
		job, ok := c.state.messageJobs[jobID]
		if !ok {
			return driver.RowsAffected(0), nil
		}
		startedAt := args[3].Value.(time.Time)
		job.Status = model.MessageJobStatus(args[1].Value.(string))
		job.WorkerID = args[2].Value.(string)
		job.StartedAt = &startedAt
		job.UpdatedAt = startedAt
		c.state.messageJobs[jobID] = job
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "UPDATE message_jobs") && strings.Contains(query, "latency_ms = $4"):
		jobID := args[0].Value.(string)
		job, ok := c.state.messageJobs[jobID]
		if !ok {
			return driver.RowsAffected(0), nil
		}
		finishedAt := args[2].Value.(time.Time)
		job.Status = model.MessageJobStatus(args[1].Value.(string))
		job.FinishedAt = &finishedAt
		job.LatencyMS = args[3].Value.(int64)
		job.UpdatedAt = finishedAt
		c.state.messageJobs[jobID] = job
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "UPDATE message_jobs") && strings.Contains(query, "last_error_code = $4"):
		jobID := args[0].Value.(string)
		job, ok := c.state.messageJobs[jobID]
		if !ok {
			return driver.RowsAffected(0), nil
		}
		finishedAt := args[2].Value.(time.Time)
		job.Status = model.MessageJobStatus(args[1].Value.(string))
		job.FinishedAt = &finishedAt
		job.LastErrorCode = args[3].Value.(string)
		job.LastErrorMessage = args[4].Value.(string)
		job.UpdatedAt = finishedAt
		c.state.messageJobs[jobID] = job
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "UPDATE outbox_events") && strings.Contains(query, "attempt_count = attempt_count + 1"):
		eventID := args[0].Value.(string)
		event, ok := c.state.outboxEvents[eventID]
		if !ok {
			return driver.RowsAffected(0), nil
		}
		failedAt := args[3].Value.(time.Time)
		event.Status = model.OutboxEventStatus(args[1].Value.(string))
		event.AttemptCount++
		event.LastError = args[2].Value.(string)
		event.UpdatedAt = failedAt
		c.state.outboxEvents[eventID] = event
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "UPDATE outbox_events") && strings.Contains(query, "published_at = $3"):
		eventID := args[0].Value.(string)
		event, ok := c.state.outboxEvents[eventID]
		if !ok {
			return driver.RowsAffected(0), nil
		}
		publishedAt := args[2].Value.(time.Time)
		event.Status = model.OutboxEventStatus(args[1].Value.(string))
		event.LastError = ""
		event.PublishedAt = &publishedAt
		event.UpdatedAt = publishedAt
		c.state.outboxEvents[eventID] = event
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "DELETE FROM session_messages"):
		sessionID := args[0].Value.(string)
		session, ok := c.state.gameSessions[sessionID]
		if !ok {
			return driver.RowsAffected(0), nil
		}
		session.History = nil
		c.state.gameSessions[sessionID] = session
		for messageID, fields := range c.state.messageAsync {
			if fields.SessionID == sessionID {
				delete(c.state.messageAsync, messageID)
			}
		}
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "DELETE FROM sessions"):
		sessionID := args[0].Value.(string)
		if _, ok := c.state.gameSessions[sessionID]; !ok {
			return driver.RowsAffected(0), nil
		}
		delete(c.state.gameSessions, sessionID)
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "INSERT INTO session_messages"):
		sessionID := args[1].Value.(string)
		session := c.state.gameSessions[sessionID]
		createdAtIndex := 8
		if len(args) > 10 {
			createdAtIndex = 10
		}
		session.History = append(session.History, model.HistoryRecord{
			ID: args[0].Value.(string),
			User: model.SessionUser{
				ID:   args[2].Value.(string),
				Name: args[3].Value.(string),
			},
			Message: model.Message{
				Content: args[4].Value.(string),
			},
			Sequence:  args[5].Value.(int64),
			Source:    model.MessageSource(args[6].Value.(string)),
			CreatedAt: args[createdAtIndex].Value.(time.Time),
		})
		asyncFields := fakeSessionMessageAsyncFields{SessionID: sessionID}
		if len(args) > 8 && args[8].Value != nil {
			asyncFields.SourceJobID = args[8].Value.(string)
		}
		if len(args) > 9 && args[9].Value != nil {
			asyncFields.ReplyToMessageID = args[9].Value.(string)
		}
		c.state.messageAsync[args[0].Value.(string)] = asyncFields
		c.state.gameSessions[sessionID] = session
		return driver.RowsAffected(1), nil
	default:
		return nil, fmt.Errorf("unexpected exec query: %s", query)
	}
}

func (c *fakePGConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	switch {
	case strings.Contains(query, "FROM users") && strings.Contains(query, "WHERE id = $1"):
		user, ok := c.state.users[args[0].Value.(string)]
		if !ok {
			return &fakeRows{}, nil
		}
		return userRows([]model.User{user}), nil
	case strings.Contains(query, "FROM users") && strings.Contains(query, "WHERE email = $1"):
		id, ok := c.state.emails[args[0].Value.(string)]
		if !ok {
			return &fakeRows{}, nil
		}
		user := c.state.users[id]
		return userRows([]model.User{user}), nil
	case strings.Contains(query, "FROM auth_sessions") && strings.Contains(query, "WHERE token_hash = $1"):
		sessionID, ok := c.state.tokens[args[0].Value.(string)]
		if !ok {
			return &fakeRows{}, nil
		}
		session := c.state.sessions[sessionID]
		return authSessionRows([]model.AuthSession{session}), nil
	case strings.Contains(query, "FROM sessions") && strings.Contains(query, "WHERE id = $1"):
		session, ok := c.state.gameSessions[args[0].Value.(string)]
		if !ok {
			return &fakeRows{}, nil
		}
		return sessionRows([]model.Session{session}), nil
	case strings.Contains(query, "FROM sessions") && strings.Contains(query, "WHERE user_id = $1"):
		userID := args[0].Value.(string)
		sessions := make([]model.Session, 0)
		for _, session := range c.state.gameSessions {
			if session.UserID == userID {
				sessions = append(sessions, session)
			}
		}
		return sessionRows(sessions), nil
	case strings.Contains(query, "FROM session_memories"):
		memory, ok := c.state.memories[args[0].Value.(string)]
		if !ok {
			return &fakeRows{}, nil
		}
		return sessionMemoryRows([]model.SessionMemory{memory}), nil
	case strings.Contains(query, "FROM message_jobs") && strings.Contains(query, "WHERE id = $1"):
		job, ok := c.state.messageJobs[args[0].Value.(string)]
		if !ok {
			return &fakeRows{}, nil
		}
		return messageJobRows([]model.MessageJob{job}), nil
	case strings.Contains(query, "FROM message_jobs") && strings.Contains(query, "WHERE message_id = $1"):
		messageID := args[0].Value.(string)
		for _, job := range c.state.messageJobs {
			if job.MessageID == messageID {
				return messageJobRows([]model.MessageJob{job}), nil
			}
		}
		return &fakeRows{}, nil
	case strings.Contains(query, "FROM outbox_events"):
		statuses := make(map[string]struct{}, len(args)-1)
		for _, arg := range args[:len(args)-1] {
			statuses[arg.Value.(string)] = struct{}{}
		}
		limit := int(args[len(args)-1].Value.(int64))
		events := make([]model.OutboxEvent, 0)
		for _, event := range c.state.outboxEvents {
			if _, ok := statuses[string(event.Status)]; ok {
				events = append(events, event)
			}
		}
		sort.SliceStable(events, func(i, j int) bool {
			if events[i].CreatedAt.Equal(events[j].CreatedAt) {
				return events[i].ID < events[j].ID
			}
			return events[i].CreatedAt.Before(events[j].CreatedAt)
		})
		return outboxEventRows(events, limit), nil
	case strings.Contains(query, "FROM session_messages") && strings.Contains(query, "user_name") && strings.Contains(query, "source_job_id"):
		sessionID := args[0].Value.(string)
		session, ok := c.state.gameSessions[sessionID]
		if !ok {
			return &fakeRows{cols: []string{"id", "user_id", "user_name", "content", "sequence", "source", "source_job_id", "reply_to_message_id", "created_at"}}, nil
		}
		return historyRowsWithAsync(session.History, c.state.messageAsync), nil
	case strings.Contains(query, "FROM session_messages") && strings.Contains(query, "source_job_id"):
		sessionID := args[0].Value.(string)
		rows := &fakeRows{
			cols: []string{"id", "source_job_id", "reply_to_message_id"},
		}
		for messageID, fields := range c.state.messageAsync {
			if fields.SessionID != sessionID {
				continue
			}
			row := []driver.Value{messageID, nil, nil}
			if fields.SourceJobID != "" {
				row[1] = fields.SourceJobID
			}
			if fields.ReplyToMessageID != "" {
				row[2] = fields.ReplyToMessageID
			}
			rows.data = append(rows.data, row)
		}
		sort.SliceStable(rows.data, func(i, j int) bool {
			return rows.data[i][0].(string) < rows.data[j][0].(string)
		})
		return rows, nil
	case strings.Contains(query, "FROM session_messages"):
		session, ok := c.state.gameSessions[args[0].Value.(string)]
		if !ok {
			return &fakeRows{cols: []string{"id", "user_id", "user_name", "content", "sequence", "source", "created_at"}}, nil
		}
		return historyRows(session.History), nil
	case strings.Contains(query, "FROM knowledge_chunks") && strings.Contains(query, "websearch_to_tsquery"):
		knowledgeBase := args[0].Value.(string)
		queryText := strings.ToLower(args[1].Value.(string))
		candidates := make([]fakeKnowledgeChunk, 0)
		for _, chunk := range c.state.knowledge {
			if chunk.KnowledgeBase != knowledgeBase {
				continue
			}
			score := 0.0
			title := strings.ToLower(chunk.Title)
			content := strings.ToLower(chunk.Content)
			if strings.Contains(title, queryText) {
				score += 5
			}
			if strings.Contains(content, queryText) {
				score += 2
			}
			for _, alias := range chunk.Aliases {
				if strings.Contains(strings.ToLower(alias), queryText) {
					score += 4
				}
			}
			if score > 0 {
				candidates = append(candidates, chunkWithScore(chunk, score))
			}
		}
		sortKnowledgeByScore(candidates)
		return knowledgeRows(candidates, int(args[2].Value.(int64))), nil
	case strings.Contains(query, "FROM knowledge_chunks") && strings.Contains(query, "embedding <=>"):
		knowledgeBase := args[0].Value.(string)
		queryVector, err := parseVectorLiteral(args[1].Value.(string))
		if err != nil {
			return nil, err
		}
		candidates := make([]fakeKnowledgeChunk, 0)
		for _, chunk := range c.state.knowledge {
			if chunk.KnowledgeBase != knowledgeBase {
				continue
			}
			score := cosineSimilarity(chunk.Embedding, queryVector)
			candidates = append(candidates, chunkWithScore(chunk, score))
		}
		sortKnowledgeByScore(candidates)
		return knowledgeRows(candidates, int(args[2].Value.(int64))), nil
	case strings.Contains(query, "FROM knowledge_index_metadata"):
		metadata, ok := c.state.indexMeta[args[0].Value.(string)]
		if !ok {
			return &fakeRows{}, nil
		}
		return knowledgeIndexMetadataRows([]fakeKnowledgeIndexMetadata{metadata}), nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
}

func (c *fakePGConn) Ping(ctx context.Context) error { return nil }

type fakePGTx struct{}

func (t *fakePGTx) Commit() error   { return nil }
func (t *fakePGTx) Rollback() error { return nil }

type fakeRows struct {
	cols []string
	data [][]driver.Value
	idx  int
}

func (r *fakeRows) Columns() []string { return r.cols }
func (r *fakeRows) Close() error      { return nil }

func (r *fakeRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.idx])
	r.idx++
	return nil
}

func userRows(users []model.User) driver.Rows {
	rows := &fakeRows{
		cols: []string{"id", "email", "password_hash", "display_name", "status", "created_at", "updated_at", "last_login_at"},
	}
	for _, user := range users {
		row := []driver.Value{
			user.ID,
			user.Email,
			user.PasswordHash,
			user.DisplayName,
			string(user.Status),
			user.CreatedAt,
			user.UpdatedAt,
			nil,
		}
		if user.LastLoginAt != nil {
			row[7] = *user.LastLoginAt
		}
		rows.data = append(rows.data, row)
	}
	return rows
}

func authSessionRows(sessions []model.AuthSession) driver.Rows {
	rows := &fakeRows{
		cols: []string{"id", "user_id", "token_hash", "expires_at", "created_at", "updated_at", "last_seen_at", "user_agent", "ip_address", "revoked_at"},
	}
	for _, session := range sessions {
		row := []driver.Value{
			session.ID,
			session.UserID,
			session.TokenHash,
			session.ExpiresAt,
			session.CreatedAt,
			session.UpdatedAt,
			nil,
			nil,
			nil,
			nil,
		}
		if session.LastSeenAt != nil {
			row[6] = *session.LastSeenAt
		}
		if session.UserAgent != nil {
			row[7] = *session.UserAgent
		}
		if session.IPAddress != nil {
			row[8] = *session.IPAddress
		}
		if session.RevokedAt != nil {
			row[9] = *session.RevokedAt
		}
		rows.data = append(rows.data, row)
	}
	return rows
}

func sessionRows(sessions []model.Session) driver.Rows {
	rows := &fakeRows{
		cols: []string{"id", "user_id", "title", "channel", "created_at", "updated_at"},
	}
	for _, session := range sessions {
		rows.data = append(rows.data, []driver.Value{
			session.ID,
			session.UserID,
			session.Title,
			string(session.Channel),
			session.CreatedAt,
			session.UpdatedAt,
		})
	}
	return rows
}

func historyRows(history []model.HistoryRecord) driver.Rows {
	rows := &fakeRows{
		cols: []string{"id", "user_id", "user_name", "content", "sequence", "source", "created_at"},
	}
	for _, record := range history {
		rows.data = append(rows.data, []driver.Value{
			record.ID,
			record.User.ID,
			record.User.Name,
			record.Message.Content,
			record.Sequence,
			string(record.Source),
			record.CreatedAt,
		})
	}
	return rows
}

func historyRowsWithAsync(history []model.HistoryRecord, async map[string]fakeSessionMessageAsyncFields) driver.Rows {
	rows := &fakeRows{
		cols: []string{"id", "user_id", "user_name", "content", "sequence", "source", "source_job_id", "reply_to_message_id", "created_at"},
	}
	for _, record := range history {
		asyncFields := async[record.ID]
		row := []driver.Value{
			record.ID,
			record.User.ID,
			record.User.Name,
			record.Message.Content,
			record.Sequence,
			string(record.Source),
			nil,
			nil,
			record.CreatedAt,
		}
		sourceJobID := record.SourceJobID
		if sourceJobID == "" {
			sourceJobID = asyncFields.SourceJobID
		}
		replyToMessageID := record.ReplyToMessageID
		if replyToMessageID == "" {
			replyToMessageID = asyncFields.ReplyToMessageID
		}
		if sourceJobID != "" {
			row[6] = sourceJobID
		}
		if replyToMessageID != "" {
			row[7] = replyToMessageID
		}
		rows.data = append(rows.data, row)
	}
	return rows
}

func sessionMemoryRows(memories []model.SessionMemory) driver.Rows {
	rows := &fakeRows{
		cols: []string{"character_summary", "scene_summary", "current_objective", "recent_key_events", "updated_at"},
		data: make([][]driver.Value, 0, len(memories)),
	}
	for _, memory := range memories {
		events, _ := json.Marshal(memory.RecentKeyEvents)
		rows.data = append(rows.data, []driver.Value{
			memory.CharacterSummary,
			memory.SceneSummary,
			memory.CurrentObjective,
			events,
			memory.UpdatedAt,
		})
	}
	return rows
}

func messageJobRows(jobs []model.MessageJob) driver.Rows {
	rows := &fakeRows{
		cols: []string{
			"id", "message_id", "session_id", "user_id", "status",
			"attempt_count", "max_attempts", "worker_id",
			"queued_at", "started_at", "finished_at",
			"last_error_code", "last_error_message", "latency_ms",
			"created_at", "updated_at",
		},
		data: make([][]driver.Value, 0, len(jobs)),
	}
	for _, job := range jobs {
		row := []driver.Value{
			job.ID,
			job.MessageID,
			job.SessionID,
			job.UserID,
			string(job.Status),
			int64(job.AttemptCount),
			int64(job.MaxAttempts),
			job.WorkerID,
			job.QueuedAt,
			nil,
			nil,
			job.LastErrorCode,
			job.LastErrorMessage,
			job.LatencyMS,
			job.CreatedAt,
			job.UpdatedAt,
		}
		if job.StartedAt != nil {
			row[9] = *job.StartedAt
		}
		if job.FinishedAt != nil {
			row[10] = *job.FinishedAt
		}
		rows.data = append(rows.data, row)
	}
	return rows
}

func outboxEventRows(events []model.OutboxEvent, limit int) driver.Rows {
	rows := &fakeRows{
		cols: []string{
			"id", "aggregate_type", "aggregate_id", "event_type", "payload_json",
			"status", "attempt_count", "last_error", "created_at", "published_at", "updated_at",
		},
		data: make([][]driver.Value, 0, len(events)),
	}
	for i, event := range events {
		if limit > 0 && i >= limit {
			break
		}
		row := []driver.Value{
			event.ID,
			event.AggregateType,
			event.AggregateID,
			event.EventType,
			[]byte(event.PayloadJSON),
			string(event.Status),
			int64(event.AttemptCount),
			event.LastError,
			event.CreatedAt,
			nil,
			event.UpdatedAt,
		}
		if event.PublishedAt != nil {
			row[9] = *event.PublishedAt
		}
		rows.data = append(rows.data, row)
	}
	return rows
}

func knowledgeRows(chunks []fakeKnowledgeChunk, limit int) driver.Rows {
	rows := &fakeRows{
		cols: []string{"id", "source_id", "knowledge_base", "title", "content", "metadata", "score"},
	}
	for i, chunk := range chunks {
		if limit > 0 && i >= limit {
			break
		}
		score := 0.0
		if chunk.Metadata != nil {
			var payload map[string]any
			_ = json.Unmarshal(chunk.Metadata, &payload)
			if value, ok := payload["_fake_score"].(float64); ok {
				score = value
				delete(payload, "_fake_score")
				chunk.Metadata, _ = json.Marshal(payload)
			}
		}
		rows.data = append(rows.data, []driver.Value{
			chunk.ID,
			chunk.SourceID,
			chunk.KnowledgeBase,
			chunk.Title,
			chunk.Content,
			chunk.Metadata,
			score,
		})
	}
	return rows
}

func knowledgeIndexMetadataRows(items []fakeKnowledgeIndexMetadata) driver.Rows {
	rows := &fakeRows{
		cols: []string{"knowledge_base", "embedding_provider", "embedding_model", "embedding_dim", "built_at"},
		data: make([][]driver.Value, 0, len(items)),
	}
	for _, item := range items {
		rows.data = append(rows.data, []driver.Value{
			item.KnowledgeBase,
			item.EmbeddingProvider,
			item.EmbeddingModel,
			int64(item.EmbeddingDim),
			item.BuiltAt,
		})
	}
	return rows
}

func chunkWithScore(chunk fakeKnowledgeChunk, score float64) fakeKnowledgeChunk {
	var payload map[string]any
	if len(chunk.Metadata) > 0 {
		_ = json.Unmarshal(chunk.Metadata, &payload)
	}
	if payload == nil {
		payload = make(map[string]any)
	}
	payload["_fake_score"] = score
	raw, _ := json.Marshal(payload)
	chunk.Metadata = raw
	return chunk
}

func sortKnowledgeByScore(chunks []fakeKnowledgeChunk) {
	sort.SliceStable(chunks, func(i, j int) bool {
		scoreI := extractFakeScore(chunks[i].Metadata)
		scoreJ := extractFakeScore(chunks[j].Metadata)
		if scoreI == scoreJ {
			return chunks[i].ID < chunks[j].ID
		}
		return scoreI > scoreJ
	})
}

func extractFakeScore(raw []byte) float64 {
	if len(raw) == 0 {
		return 0
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return 0
	}
	if value, ok := payload["_fake_score"].(float64); ok {
		return value
	}
	return 0
}

func parseVectorLiteral(value string) ([]float32, error) {
	value = strings.TrimSpace(strings.Trim(value, "[]"))
	if value == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	vector := make([]float32, 0, len(parts))
	for _, part := range parts {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(part), 32)
		if err != nil {
			return nil, err
		}
		vector = append(vector, float32(parsed))
	}
	return vector, nil
}

func parseTextArrayLiteral(value string) ([]string, error) {
	value = strings.TrimSpace(strings.Trim(value, "{}"))
	if value == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"`)
		part = strings.ReplaceAll(part, `\"`, `"`)
		part = strings.ReplaceAll(part, `\\`, `\`)
		values = append(values, part)
	}
	return values, nil
}

func cosineSimilarity(a []float32, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func decodeJSONRow[T any](t *testing.T, raw string, out *T) {
	t.Helper()
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		t.Fatalf("expected json to decode, got %v", err)
	}
}
