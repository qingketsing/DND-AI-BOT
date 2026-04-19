package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"DND-AI-BOT/internal/observability"
)

const requestIDHeader = "X-Request-ID"

// NewRequestIDMiddleware 注入或透传请求 ID。
func NewRequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(requestIDHeader)
			if requestID == "" {
				requestID = generateRequestID()
			}
			w.Header().Set(requestIDHeader, requestID)
			next.ServeHTTP(w, r.WithContext(observability.WithRequestID(r.Context(), requestID)))
		})
	}
}

// RequestIDFromRequest 从请求 context 读取请求 ID。
func RequestIDFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	return observability.RequestIDFromContext(r.Context())
}

func generateRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(bytes[:])
}
