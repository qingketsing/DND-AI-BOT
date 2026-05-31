package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type SessionLock interface {
	Acquire(ctx context.Context, sessionID string, jobID string, workerID string, ttl time.Duration) (bool, error)
	Renew(ctx context.Context, sessionID string, jobID string, workerID string, ttl time.Duration) error
	Release(ctx context.Context, sessionID string, jobID string, workerID string) error
}

type SessionLockInspector interface {
	Inspect(ctx context.Context, sessionID string) (SessionLockOwner, error)
}

type SessionLockOwner struct {
	Exists   bool
	JobID    string
	WorkerID string
}

const (
	defaultSessionLockTTL               = 180 * time.Second
	defaultSessionLockHeartbeatInterval = 30 * time.Second
)

type sessionLockBackend interface {
	SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error)
	CompareAndExpire(ctx context.Context, key string, expected string, ttl time.Duration) (bool, error)
	CompareAndDelete(ctx context.Context, key string, expected string) (bool, error)
	Get(ctx context.Context, key string) (string, error)
}

type RedisSessionLock struct {
	backend sessionLockBackend
}

func NewRedisSessionLock(client *goredis.Client) *RedisSessionLock {
	return &RedisSessionLock{
		backend: &redisSessionLockBackend{client: client},
	}
}

func newRedisSessionLockWithBackend(backend sessionLockBackend) *RedisSessionLock {
	return &RedisSessionLock{backend: backend}
}

func (l *RedisSessionLock) Acquire(ctx context.Context, sessionID string, jobID string, workerID string, ttl time.Duration) (bool, error) {
	if l.backend == nil {
		return false, errors.New("session lock backend is nil")
	}
	payload, err := sessionLockValue(jobID, workerID)
	if err != nil {
		return false, err
	}
	return l.backend.SetNX(ctx, sessionLockKey(sessionID), payload, ttl)
}

func (l *RedisSessionLock) Renew(ctx context.Context, sessionID string, jobID string, workerID string, ttl time.Duration) error {
	if l.backend == nil {
		return errors.New("session lock backend is nil")
	}
	payload, err := sessionLockValue(jobID, workerID)
	if err != nil {
		return err
	}
	ok, err := l.backend.CompareAndExpire(ctx, sessionLockKey(sessionID), payload, ttl)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("session lock not owned by requester")
	}
	return nil
}

func (l *RedisSessionLock) Release(ctx context.Context, sessionID string, jobID string, workerID string) error {
	if l.backend == nil {
		return errors.New("session lock backend is nil")
	}
	payload, err := sessionLockValue(jobID, workerID)
	if err != nil {
		return err
	}
	_, err = l.backend.CompareAndDelete(ctx, sessionLockKey(sessionID), payload)
	return err
}

func (l *RedisSessionLock) Inspect(ctx context.Context, sessionID string) (SessionLockOwner, error) {
	if l.backend == nil {
		return SessionLockOwner{}, errors.New("session lock backend is nil")
	}
	raw, err := l.backend.Get(ctx, sessionLockKey(sessionID))
	if errors.Is(err, goredis.Nil) {
		return SessionLockOwner{}, nil
	}
	if err != nil {
		return SessionLockOwner{}, err
	}
	var state sessionLockState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return SessionLockOwner{}, err
	}
	return SessionLockOwner{
		Exists:   true,
		JobID:    state.JobID,
		WorkerID: state.WorkerID,
	}, nil
}

func startSessionLockHeartbeat(
	ctx context.Context,
	lock SessionLock,
	sessionID string,
	jobID string,
	workerID string,
	ttl time.Duration,
	interval time.Duration,
	tickerFactory func(time.Duration) <-chan time.Time,
) <-chan error {
	result := make(chan error, 1)
	if tickerFactory == nil {
		tickerFactory = func(interval time.Duration) <-chan time.Time {
			ticker := time.NewTicker(interval)
			ch := make(chan time.Time)
			go func() {
				defer ticker.Stop()
				defer close(ch)
				for {
					select {
					case <-ctx.Done():
						return
					case tick := <-ticker.C:
						select {
						case <-ctx.Done():
							return
						case ch <- tick:
						}
					}
				}
			}()
			return ch
		}
	}

	go func() {
		defer close(result)
		ticks := tickerFactory(interval)
		for {
			select {
			case <-ctx.Done():
				result <- nil
				return
			case _, ok := <-ticks:
				if !ok {
					result <- nil
					return
				}
				if err := lock.Renew(ctx, sessionID, jobID, workerID, ttl); err != nil {
					result <- err
					return
				}
			}
		}
	}()

	return result
}

func sessionLockKey(sessionID string) string {
	return fmt.Sprintf("session:%s:processing_lock", sessionID)
}

type sessionLockState struct {
	JobID    string `json:"job_id"`
	WorkerID string `json:"worker_id"`
}

func sessionLockValue(jobID string, workerID string) (string, error) {
	raw, err := json.Marshal(sessionLockState{
		JobID:    jobID,
		WorkerID: workerID,
	})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

type redisSessionLockBackend struct {
	client *goredis.Client
}

func (b *redisSessionLockBackend) SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	return b.client.SetNX(ctx, key, value, ttl).Result()
}

func (b *redisSessionLockBackend) CompareAndExpire(ctx context.Context, key string, expected string, ttl time.Duration) (bool, error) {
	result, err := b.client.Eval(ctx, `
local current = redis.call("GET", KEYS[1])
if current == ARGV[1] then
  redis.call("PEXPIRE", KEYS[1], ARGV[2])
  return 1
end
return 0
`, []string{key}, expected, ttl.Milliseconds()).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (b *redisSessionLockBackend) CompareAndDelete(ctx context.Context, key string, expected string) (bool, error) {
	result, err := b.client.Eval(ctx, `
local current = redis.call("GET", KEYS[1])
if current == ARGV[1] then
  redis.call("DEL", KEYS[1])
  return 1
end
return 0
`, []string{key}, expected).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (b *redisSessionLockBackend) Get(ctx context.Context, key string) (string, error) {
	return b.client.Get(ctx, key).Result()
}
