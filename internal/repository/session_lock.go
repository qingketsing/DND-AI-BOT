package repository

import "context"

type SessionLockInspector interface {
	Inspect(ctx context.Context, sessionID string) (SessionLockOwner, error)
}

type SessionLockOwner struct {
	Exists   bool
	JobID    string
	WorkerID string
}
