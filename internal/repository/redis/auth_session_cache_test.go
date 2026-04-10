package redis

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"DND-AI-BOT/internal/repository"
)

func TestRedisAuthSessionCacheSetGetAndDelete(t *testing.T) {
	server := newFakeRedisServer(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cache := NewRedisAuthSessionCache(client)
	now := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	session := CachedAuthSession{
		SessionID: "session-1",
		UserID:    "user-1",
		ExpiresAt: now.Add(time.Hour),
		Revoked:   false,
	}

	if err := cache.Set(context.Background(), "token-hash-1", session, time.Minute); err != nil {
		t.Fatalf("expected set to succeed, got %v", err)
	}

	got, err := cache.Get(context.Background(), "token-hash-1")
	if err != nil {
		t.Fatalf("expected get to succeed, got %v", err)
	}
	if got != session {
		t.Fatalf("expected cached session to round-trip, got %+v", got)
	}

	if err := cache.Delete(context.Background(), "token-hash-1"); err != nil {
		t.Fatalf("expected delete to succeed, got %v", err)
	}
	if _, err := cache.Get(context.Background(), "token-hash-1"); !errors.Is(err, repository.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss after delete, got %v", err)
	}
}

func TestRedisAuthSessionCacheDistinguishesMarkerAndMiss(t *testing.T) {
	server := newFakeRedisServer(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	if err := client.Set(context.Background(), authSessionCacheKey("missing"), notFoundMarker, time.Minute).Err(); err != nil {
		t.Fatalf("expected set marker to succeed, got %v", err)
	}

	cache := NewRedisAuthSessionCache(client)
	_, err := cache.Get(context.Background(), "missing")
	if !errors.Is(err, repository.ErrCacheNotFoundMarker) {
		t.Fatalf("expected ErrCacheNotFoundMarker, got %v", err)
	}
}

type fakeRedisServer struct {
	ln   net.Listener
	data map[string]string
}

func newFakeRedisServer(t *testing.T) *fakeRedisServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("expected listen to succeed, got %v", err)
	}
	s := &fakeRedisServer{ln: ln, data: make(map[string]string)}
	go s.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return s
}

func (s *fakeRedisServer) Addr() string { return s.ln.Addr().String() }

func (s *fakeRedisServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *fakeRedisServer) handle(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	for {
		cmd, err := readRESPArray(r)
		if err != nil {
			return
		}
		if len(cmd) == 0 {
			continue
		}
		switch strings.ToUpper(cmd[0]) {
		case "PING":
			_, _ = w.WriteString("+PONG\r\n")
		case "SET":
			if len(cmd) < 3 {
				_, _ = w.WriteString("-ERR wrong number of arguments for 'set'\r\n")
				_ = w.Flush()
				continue
			}
			s.data[cmd[1]] = cmd[2]
			_, _ = w.WriteString("+OK\r\n")
		case "GET":
			if value, ok := s.data[cmd[1]]; ok {
				_, _ = w.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(value), value))
			} else {
				_, _ = w.WriteString("$-1\r\n")
			}
		case "DEL":
			if _, ok := s.data[cmd[1]]; ok {
				delete(s.data, cmd[1])
				_, _ = w.WriteString(":1\r\n")
			} else {
				_, _ = w.WriteString(":0\r\n")
			}
		default:
			_, _ = w.WriteString("-ERR unsupported command\r\n")
		}
		_ = w.Flush()
	}
}

func readRESPArray(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("expected array, got %q", line)
	}
	count, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "*")))
	if err != nil {
		return nil, err
	}
	items := make([]string, 0, count)
	for i := 0; i < count; i++ {
		if _, err := r.ReadString('\n'); err != nil {
			return nil, err
		}
		valueLine, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		items = append(items, strings.TrimSuffix(valueLine, "\r\n"))
	}
	return items, nil
}

func TestRedisAuthSessionCacheRoundTripsJSON(t *testing.T) {
	server := newFakeRedisServer(t)
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	cache := NewRedisAuthSessionCache(client)
	session := CachedAuthSession{SessionID: "session-1", UserID: "user-1", ExpiresAt: time.Unix(100, 0).UTC(), Revoked: true}
	if err := cache.Set(context.Background(), "token-hash-1", session, time.Minute); err != nil {
		t.Fatalf("expected set to succeed, got %v", err)
	}

	raw := server.data[authSessionCacheKey("token-hash-1")]
	var got CachedAuthSession
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("expected cached payload to be valid json, got %v", err)
	}
	if got != session {
		t.Fatalf("expected json payload to round-trip, got %+v", got)
	}
}

func TestAuthSessionCacheKeyUsesSpecPrefix(t *testing.T) {
	if got := authSessionCacheKey("token-hash-1"); got != "auth:session:token-hash-1" {
		t.Fatalf("expected auth session cache key to use spec prefix, got %q", got)
	}
}
