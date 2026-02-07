package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Config configures the OneBot client
type Config struct {
	WSURL       string
	AccessToken string
}

type OneBot struct {
	config Config
	conn   *websocket.Conn
	mu     sync.Mutex
	done   chan struct{}

	// Handler for group messages
	// Handler should take (groupID, senderID, message)
	GroupMsgHandler func(groupID int64, senderID int64, msg string)
}

// Event represents a basic OneBot event
type Event struct {
	PostType      string `json:"post_type"`
	MetaEventType string `json:"meta_event_type"`
	MessageType   string `json:"message_type"`
	SubType       string `json:"sub_type"`
	GroupID     int64  `json:"group_id"`
	UserID      int64       `json:"user_id"`
	Message     interface{} `json:"message"`     // content
	RawMessage  string      `json:"raw_message"` // content with CQ codes
	SelfID      int64  `json:"self_id"`
}

// ActionFrame is the wrapper for sending requests
type ActionFrame struct {
	Action string      `json:"action"`
	Params interface{} `json:"params"`
	Echo   string      `json:"echo"`
}

type GroupMsgParams struct {
	GroupID int64  `json:"group_id"`
	Message string `json:"message"`
}

func New(cfg Config) *OneBot {
	return &OneBot{
		config: cfg,
		done:   make(chan struct{}),
	}
}

// Start initiates the connection and blocking listen loop
// In a real app, you might want this non-blocking or managed by a supervisor
func (b *OneBot) Start() {
	go b.connectAndListen()
}

func (b *OneBot) connectAndListen() {
	u, err := url.Parse(b.config.WSURL)
	if err != nil {
		logrus.Fatalf("Invalid WS URL: %v", err)
	}

	// 强制添加 access_token query param，以防 NapCat 不识别 Header
	if b.config.AccessToken != "" {
		q := u.Query()
		q.Set("access_token", b.config.AccessToken)
		u.RawQuery = q.Encode()
	}

	for {
		logrus.Infof("Connecting to OneBot at %s (Token len: %d)...", u.String(), len(b.config.AccessToken))
		
		// Setup headers if needed
		header := http.Header{}
		if b.config.AccessToken != "" {
			header.Add("Authorization", "Bearer "+b.config.AccessToken)
		}

		c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
		if err != nil {
			logrus.Errorf("Connection failed: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		b.mu.Lock()
		b.conn = c
		b.mu.Unlock()

		logrus.Info("Connected to OneBot!")

		// Read loop
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				logrus.Errorf("Read error: %v", err)
				break 
			}
			b.handleMessage(message)
		}
		
		logrus.Warn("Disconnected. Reconnecting...")
		time.Sleep(2 * time.Second)
	}
}

func (b *OneBot) handleMessage(msg []byte) {
	// Explicitly log received messages for debugging
	strMsg := string(msg)
	if !strings.Contains(strMsg, "\"meta_event_type\":\"heartbeat\"") {
		logrus.Infof("[RECV] Raw: %s", strMsg)
	}

	// Simple parsing for now
	var evt Event
	if err := json.Unmarshal(msg, &evt); err != nil {
		logrus.Warnf("Failed to unmarshal event: %v | Msg: %s", err, string(msg))
		return
	}

	if evt.PostType == "meta_event" && evt.MetaEventType == "heartbeat" {
		return
	}
	
	logrus.Infof("Debug: Received Event: PostType=%s | User=%d | Raw=%s", evt.PostType, evt.UserID, evt.RawMessage)

	if evt.PostType == "message" && evt.MessageType == "group" {
		// Check for @ mention OR command prefix
		target := fmt.Sprintf("[CQ:at,qq=%d]", evt.SelfID)
		isAt := strings.Contains(evt.RawMessage, target)
		isCommand := strings.HasPrefix(evt.RawMessage, ".")

		if isAt || isCommand {
			// Clean message
			content := evt.RawMessage
			if isAt {
				content = strings.ReplaceAll(content, target, "")
			}
			content = strings.TrimSpace(content)
			
			logrus.Infof("Received Group Msg from %d in Group %d: %s", evt.UserID, evt.GroupID, content)

			if b.GroupMsgHandler != nil {
				// Execute handler in goroutine to avoid blocking read loop
				go b.GroupMsgHandler(evt.GroupID, evt.UserID, content)
			}
		}
	}
}

func (b *OneBot) SendGroupMsg(groupID int64, msg string) error {
	logrus.Infof("[SEND] To Group %d: %s", groupID, msg)
	b.mu.Lock()
	conn := b.conn
	b.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	frame := ActionFrame{
		Action: "send_group_msg",
		Params: GroupMsgParams{
			GroupID: groupID,
			Message: msg,
		},
	}

	// We are sharing the connection for writing. Gorilla websocket supports concurrent reading and writing?
	// No, it supports one concurrent reader and one concurrent writer.
	// Since SendGroupMsg might be called from multiple goroutines, we need to lock the write.
	// However, conn.WriteJSON is not thread safe if called concurrently.
	// We should probably have a dedicated write loop or a mutex. 
	// For simplicity, let's wrap WriteJSON with a mutex.
	
	// WARNING: In production, use a channel for outgoing messages.
	// We will create a simple send lock here. 
	// But `b.conn` is replaced on reconnect, so locking `b.conn` directly is tricky.
	// Let's rely on b.mu not just for setting conn but for writing.
	
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Check conn again
	if b.conn != nil {
		return b.conn.WriteJSON(frame)
	}
	return fmt.Errorf("connection lost")
}
