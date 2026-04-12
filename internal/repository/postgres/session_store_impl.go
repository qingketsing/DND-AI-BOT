package postgres

import (
	"context"
	"database/sql"
	"errors"

	"DND-AI-BOT/internal/model"
	"DND-AI-BOT/internal/repository"
)

type PGSessionStore struct {
	db *sql.DB
}

// NewPGSessionStore 创建基于 database/sql 的 Session PG 存储实现。
func NewPGSessionStore(db *sql.DB) *PGSessionStore {
	return &PGSessionStore{db: db}
}

// UpsertSession 使用 UPSERT 语句将会话数据写入数据库，确保在高并发场景下数据的一致性和完整性。注意，这里是全量重写历史记录，适用于历史记录较短的情况。如果历史记录较长，可能需要优化为增量更新。
func (s *PGSessionStore) UpsertSession(ctx context.Context, session *model.Session) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, title, channel, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET user_id = EXCLUDED.user_id,
		    title = EXCLUDED.title,
		    channel = EXCLUDED.channel,
		    updated_at = EXCLUDED.updated_at
	`, session.ID, session.UserID, session.Title, string(session.Channel), session.CreatedAt, session.UpdatedAt)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM session_messages WHERE session_id = $1`, session.ID); err != nil {
		return err
	}

	for _, record := range session.History {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO session_messages (
				id, session_id, user_id, user_name, content, sequence, source, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
			record.ID,
			session.ID,
			record.User.ID,
			record.User.Name,
			record.Message.Content,
			record.Sequence,
			string(record.Source),
			record.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetSession 从数据库中查询会话数据，并构建完整的 Session 对象返回。如果会话不存在，返回 repository.ErrSessionNotFound 错误。
func (s *PGSessionStore) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	var (
		id        string
		userID    string
		title     string
		channel   string
		createdAt sql.NullTime
		updatedAt sql.NullTime
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, title, channel, created_at, updated_at
		FROM sessions
		WHERE id = $1
	`, sessionID).Scan(&id, &userID, &title, &channel, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, user_name, content, sequence, source, created_at
		FROM session_messages
		WHERE session_id = $1
		ORDER BY sequence ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]model.HistoryRecord, 0)
	for rows.Next() {
		var (
			recordID   string
			userID     string
			userName   string
			content    string
			sequence   int64
			source     string
			recordTime sql.NullTime
		)

		if err := rows.Scan(&recordID, &userID, &userName, &content, &sequence, &source, &recordTime); err != nil {
			return nil, err
		}

		history = append(history, model.HistoryRecord{
			ID: recordID,
			User: model.SessionUser{
				ID:   userID,
				Name: userName,
			},
			Message: model.Message{
				Content: content,
			},
			Sequence:  sequence,
			Source:    model.MessageSource(source),
			CreatedAt: recordTime.Time,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &model.Session{
		ID:        id,
		UserID:    userID,
		Title:     title,
		Channel:   model.Channel(channel),
		History:   history,
		CreatedAt: createdAt.Time,
		UpdatedAt: updatedAt.Time,
	}, nil
}

// ListSessionsByUserID 返回指定用户的所有会话。
func (s *PGSessionStore) ListSessionsByUserID(ctx context.Context, userID string) ([]*model.Session, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, title, channel, created_at, updated_at
		FROM sessions
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]*model.Session, 0)
	for rows.Next() {
		var (
			id            string
			sessionUserID string
			title         string
			channel       string
			createdAt     sql.NullTime
			updatedAt     sql.NullTime
		)

		if err := rows.Scan(&id, &sessionUserID, &title, &channel, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		sessions = append(sessions, &model.Session{
			ID:        id,
			UserID:    sessionUserID,
			Title:     title,
			Channel:   model.Channel(channel),
			History:   []model.HistoryRecord{},
			CreatedAt: createdAt.Time,
			UpdatedAt: updatedAt.Time,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

// DeleteSession 删除指定会话及其消息历史。
func (s *PGSessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM session_messages WHERE session_id = $1`, sessionID); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return repository.ErrSessionNotFound
	}

	return tx.Commit()
}
