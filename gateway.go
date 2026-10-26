package highrisebot

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	DefaultGatewayURL = "wss://highrise.game/web/botapi"
	defaultWriteWait  = 10 * time.Second
	defaultPongWait   = 60 * time.Second
)

// GatewayHandlers holds all dispatch callbacks for the gateway.
type GatewayHandlers struct {
	BeforeStart  func(roomID string)
	OnReady      func(*SessionMetadata)
	OnDisconnect func(err error)
	OnChat       func(*User, string)
	OnWhisper    func(*User, string)
	OnUserJoin   func(*User, Position)
	OnUserLeave  func(*User)
	OnUserMove   func(*User, Position)
	OnEmote      func(*User, string, *User)
	OnReaction   func(*User, string, *User)
	OnTip        func(*User, *User, Item)
	OnMessage    func(userID, conversationID string, isNew bool)
	OnError      func(error)
}

// GatewayClient manages the live WebSocket gateway connection to Highrise.
type GatewayClient struct {
	GatewayURL        string
	Token             string
	RoomID            string
	HandshakeTimeout  time.Duration
	WriteWait         time.Duration
	PongWait          time.Duration
	MaxReconnectDelay time.Duration
	AutoReconnect     bool
	Handlers          GatewayHandlers

	conn      *websocket.Conn
	mu        sync.Mutex
	writeChan chan []byte
	stopChan  chan struct{}
	closeOnce sync.Once
	running   bool

	pendingMu sync.Mutex
	pending   map[string]chan []byte

	encoder *ProtocolEncoder
	decoder *ProtocolDecoder

	OnStatusChange func(status string)
}

// GatewayOptions holds client parameters.
type GatewayOptions struct {
	GatewayURL        string
	HandshakeTimeout  time.Duration
	WriteWait         time.Duration
	PongWait          time.Duration
	MaxReconnectDelay time.Duration
	AutoReconnect     bool
	TLSConfig         *tls.Config
}

// NewGatewayClient initializes a new Gateway Client instance.
func NewGatewayClient(token, roomID string, opts ...GatewayOptions) *GatewayClient {
	opt := GatewayOptions{
		GatewayURL:        DefaultGatewayURL,
		HandshakeTimeout:  15 * time.Second,
		WriteWait:         defaultWriteWait,
		PongWait:          defaultPongWait,
		MaxReconnectDelay: 30 * time.Second,
		AutoReconnect:     true,
	}
	if len(opts) > 0 {
		o := opts[0]
		if o.GatewayURL != "" {
			opt.GatewayURL = o.GatewayURL
		}
		if o.HandshakeTimeout > 0 {
			opt.HandshakeTimeout = o.HandshakeTimeout
		}
		if o.WriteWait > 0 {
			opt.WriteWait = o.WriteWait
		}
		if o.PongWait > 0 {
			opt.PongWait = o.PongWait
		}
		if o.MaxReconnectDelay > 0 {
			opt.MaxReconnectDelay = o.MaxReconnectDelay
		}
		opt.AutoReconnect = o.AutoReconnect
	}

	return &GatewayClient{
		GatewayURL:        opt.GatewayURL,
		Token:             token,
		RoomID:            roomID,
		HandshakeTimeout:  opt.HandshakeTimeout,
		WriteWait:         opt.WriteWait,
		PongWait:          opt.PongWait,
		MaxReconnectDelay: opt.MaxReconnectDelay,
		AutoReconnect:     opt.AutoReconnect,
		writeChan:         make(chan []byte, 128),
		stopChan:          make(chan struct{}),
		pending:           make(map[string]chan []byte),
		encoder:           &ProtocolEncoder{},
		decoder:           &ProtocolDecoder{},
	}
}

// Request sends an RPC request and waits for the response matching rid with timeout.
func (c *GatewayClient) Request(ctx context.Context, payload map[string]any) ([]byte, error) {
	rid := fmt.Sprintf("req_%d", time.Now().UnixNano())
	payload["rid"] = rid

	respChan := make(chan []byte, 1)

	c.pendingMu.Lock()
	c.pending[rid] = respChan
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, rid)
		c.pendingMu.Unlock()
	}()

	if err := c.Send(payload); err != nil {
		return nil, err
	}

	select {
	case res := <-respChan:
		return res, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(15 * time.Second):
		return nil, errors.New("request timeout waiting for response")
	}
}

// Connect establishes the connection and loops automatically with reconnect logic.
func (c *GatewayClient) Connect(ctx context.Context) error {
	c.running = true
	baseBackoff := 1 * time.Second

	if c.Handlers.BeforeStart != nil {
		c.Handlers.BeforeStart(c.RoomID)
	}

	for c.running {
		select {
		case <-ctx.Done():
			c.changeStatus("DISCONNECTED")
			return ctx.Err()
		case <-c.stopChan:
			c.changeStatus("STOPPED")
			return nil
		default:
		}

		c.changeStatus("CONNECTING")
		err := c.runSession(ctx)
		if err != nil {
			c.changeStatus("DISCONNECTED")
			if c.Handlers.OnDisconnect != nil {
				c.Handlers.OnDisconnect(err)
			}
			if c.Handlers.OnError != nil {
				c.Handlers.OnError(fmt.Errorf("gateway error: %w", err))
			}

			if errors.Is(err, ErrAuthRejected) || errors.Is(err, ErrRoomNotFound) {
				return err
			}
		}

		if !c.running || !c.AutoReconnect {
			break
		}

		jitter := time.Duration(rand.Int63n(int64(baseBackoff / 2)))
		sleepDuration := baseBackoff + jitter
		if sleepDuration > c.MaxReconnectDelay {
			sleepDuration = c.MaxReconnectDelay
		}

		c.changeStatus("RECONNECTING")
		log.Printf("[highrisebot] Reconnecting in %v (jitter: %v)...", sleepDuration, jitter)
		select {
		case <-time.After(sleepDuration):
		case <-ctx.Done():
			c.changeStatus("DISCONNECTED")
			return ctx.Err()
		case <-c.stopChan:
			c.changeStatus("STOPPED")
			return nil
		}

		baseBackoff *= 2
		if baseBackoff > c.MaxReconnectDelay {
			baseBackoff = c.MaxReconnectDelay
		}
	}
	return nil
}

func (c *GatewayClient) changeStatus(s string) {
	if c.OnStatusChange != nil {
		c.OnStatusChange(s)
	}
}

// Close gracefully closes the gateway connection.
func (c *GatewayClient) Close() {
	c.closeOnce.Do(func() {
		c.running = false
		close(c.stopChan)
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
		}
		c.mu.Unlock()
	})
}

// Send sends a raw payload to the gateway.
func (c *GatewayClient) Send(payload any) error {
	data, err := c.encoder.Encode(payload)
	if err != nil {
		return err
	}

	select {
	case c.writeChan <- data:
		return nil
	case <-c.stopChan:
		return errors.New("gateway is closed")
	default:
		return errors.New("gateway write buffer full")
	}
}

func (c *GatewayClient) runSession(ctx context.Context) error {
	headers := make(http.Header)
	headers.Set("room-id", c.RoomID)
	headers.Set("api-token", c.Token)

	dialer := websocket.Dialer{
		HandshakeTimeout: c.HandshakeTimeout,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	conn, resp, err := dialer.DialContext(ctx, c.GatewayURL, headers)
	if err != nil {
		if resp != nil && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) {
			return ErrAuthRejected
		}
		return fmt.Errorf("failed to dial gateway: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	defer conn.Close()

	initPayload := map[string]any{
		"_type":   "KeepaliveRequest",
		"token":   c.Token,
		"room_id": c.RoomID,
	}
	_ = c.Send(initPayload)

	errChan := make(chan error, 2)
	go c.writePump(conn, errChan)
	go c.readPump(conn, errChan)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.stopChan:
		return nil
	case err := <-errChan:
		return err
	}
}

func (c *GatewayClient) readPump(conn *websocket.Conn, errChan chan<- error) {
	conn.SetReadLimit(MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(c.PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(c.PongWait))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			errChan <- err
			return
		}
		go c.handleMessage(message)
	}
}

func (c *GatewayClient) writePump(conn *websocket.Conn, errChan chan<- error) {
	pingPeriod := (c.PongWait * 9) / 10
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case message, ok := <-c.writeChan:
			conn.SetWriteDeadline(time.Now().Add(c.WriteWait))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				errChan <- err
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(c.WriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				errChan <- err
				return
			}
		}
	}
}

func (c *GatewayClient) handleMessage(msg []byte) {
	env, err := c.decoder.DecodeEnvelope(msg)
	if err != nil {
		if c.Handlers.OnError != nil {
			c.Handlers.OnError(fmt.Errorf("protocol decode error: %w", err))
		}
		return
	}

	if env.RID != "" {
		c.pendingMu.Lock()
		ch, exists := c.pending[env.RID]
		c.pendingMu.Unlock()
		if exists {
			ch <- msg
			return
		}
	}

	switch env.Type {
	case "SessionMetadata":
		c.changeStatus("CONNECTED")
		if c.Handlers.OnReady != nil {
			var sm SessionMetadata
			if err := c.decoder.DecodeTyped(msg, &sm); err == nil {
				c.Handlers.OnReady(&sm)
			}
		}

	case "ChatEvent":
		if c.Handlers.OnChat != nil {
			var ce ChatEvent
			if err := c.decoder.DecodeTyped(msg, &ce); err == nil {
				if ce.Whisper && c.Handlers.OnWhisper != nil {
					c.Handlers.OnWhisper(&ce.User, ce.Message)
				} else {
					c.Handlers.OnChat(&ce.User, ce.Message)
				}
			}
		}

	case "UserJoinedEvent":
		if c.Handlers.OnUserJoin != nil {
			var je UserJoinedEvent
			if err := c.decoder.DecodeTyped(msg, &je); err == nil {
				c.Handlers.OnUserJoin(&je.User, je.Position)
			}
		}

	case "UserLeftEvent":
		if c.Handlers.OnUserLeave != nil {
			var le UserLeftEvent
			if err := c.decoder.DecodeTyped(msg, &le); err == nil {
				c.Handlers.OnUserLeave(&le.User)
			}
		}

	case "UserMovedEvent":
		if c.Handlers.OnUserMove != nil {
			var me UserMovedEvent
			if err := c.decoder.DecodeTyped(msg, &me); err == nil {
				c.Handlers.OnUserMove(&me.User, me.Position)
			}
		}

	case "EmoteEvent":
		if c.Handlers.OnEmote != nil {
			var ee EmoteEvent
			if err := c.decoder.DecodeTyped(msg, &ee); err == nil {
				c.Handlers.OnEmote(&ee.User, ee.EmoteID, ee.Receiver)
			}
		}

	case "ReactionEvent":
		if c.Handlers.OnReaction != nil {
			var re ReactionEvent
			if err := c.decoder.DecodeTyped(msg, &re); err == nil {
				c.Handlers.OnReaction(&re.User, re.Reaction, &re.Receiver)
			}
		}

	case "TipReactionEvent":
		if c.Handlers.OnTip != nil {
			var te TipReactionEvent
			if err := c.decoder.DecodeTyped(msg, &te); err == nil {
				c.Handlers.OnTip(&te.Sender, &te.Receiver, te.Item)
			}
		}

	case "MessageEvent":
		if c.Handlers.OnMessage != nil {
			var me MessageEvent
			if err := c.decoder.DecodeTyped(msg, &me); err == nil {
				c.Handlers.OnMessage(me.UserID, me.ConversationID, me.IsNewConversation)
			}
		}

	case "Error":
		var ee ErrorPayload
		_ = c.decoder.DecodeTyped(msg, &ee)
		if c.Handlers.OnError != nil {
			c.Handlers.OnError(fmt.Errorf("server error: %s (code: %d)", ee.Message, ee.Code))
		}

	default:
		if c.Handlers.OnError != nil {
			c.Handlers.OnError(fmt.Errorf("unhandled event type: %s", env.Type))
		}
	}
}
