package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
}

func newFakePGState() *fakePGState {
	return &fakePGState{
		users:        make(map[string]model.User),
		emails:       make(map[string]string),
		sessions:     make(map[string]model.AuthSession),
		tokens:       make(map[string]string),
		gameSessions: make(map[string]model.Session),
	}
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
	case strings.Contains(query, "DELETE FROM session_messages"):
		sessionID := args[0].Value.(string)
		session, ok := c.state.gameSessions[sessionID]
		if !ok {
			return driver.RowsAffected(0), nil
		}
		session.History = nil
		c.state.gameSessions[sessionID] = session
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "INSERT INTO session_messages"):
		sessionID := args[1].Value.(string)
		session := c.state.gameSessions[sessionID]
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
			CreatedAt: args[7].Value.(time.Time),
		})
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
	case strings.Contains(query, "FROM session_messages"):
		session, ok := c.state.gameSessions[args[0].Value.(string)]
		if !ok {
			return &fakeRows{cols: []string{"id", "user_id", "user_name", "content", "sequence", "source", "created_at"}}, nil
		}
		return historyRows(session.History), nil
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

func decodeJSONRow[T any](t *testing.T, raw string, out *T) {
	t.Helper()
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		t.Fatalf("expected json to decode, got %v", err)
	}
}
