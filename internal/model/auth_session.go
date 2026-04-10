package model

import "time"

// AuthSession stores the durable server-side state for a login session.
type AuthSession struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	TokenHash  string     `json:"-"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	UserAgent  *string    `json:"user_agent"`
	IPAddress  *string    `json:"ip_address"`
	RevokedAt  *time.Time `json:"revoked_at"`
}
