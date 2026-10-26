package gateway

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rezadesky/highrisebot/models"
	"github.com/rezadesky/highrisebot/protocol"
)

const (
	DefaultGatewayURL = "wss://highrise.game/web/botapi"
	defaultWriteWait  = 10 * time.Second
	defaultPongWait   = 60 * time.Second
)

// HandlerFuncs holds all dispatch callbacks.
type HandlerFuncs struct {
	BeforeStart func(roomID string)
	OnReady     func(*models.SessionMetadata)
	OnDisconnect func(err error)
	OnChat      func(*models.User, string)
	OnWhisper   func(*models.User, string)
	OnUserJoin  func(*models.User, models.Position)
	OnUserLeave func(*models.User)
	OnUserMove  func(*models.User, models.Position)
	OnEmote     func(*models.User, string, *models.User)
	OnReaction  func(*models.User, string, *models.User)
	OnTip       func(*models.User, *models.User, models.Item)
	OnMessage   func(userID, conversationID string, isNew bool)
	OnError     func(error)
}

// Client manages the live WebSocket gateway connection to Highrise.
type Client struct {
	GatewayURL        string
	Token             string
	RoomID            string
	HandshakeTimeout  time.Duration
	WriteWait         time.Duration
	PongWait          time.Duration
	MaxReconnectDelay time.Duration
	AutoReconnect     bool
	Handlers          HandlerFuncs

	conn      *websocket.Conn
	mu        sync.Mutex
	writeChan chan []byte
	stopChan  chan struct{}
	closeOnce sync.Once
	running   bool

	// Request tracking for RPC responses (rid -> response channel)
	pendingMu sync.Mutex
	pending   map[string]chan []byte

	// Status hook
	OnStatusChange func(status string)
	encoder *protocol.Encoder
	decoder *protocol.Decoder
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

// NewClient initializes a new Gateway Client instance.
func NewClient(token, roomID string, opts ...GatewayOptions) *Client {
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

	return &Client{
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
		encoder:           protocol.NewEncoder(),
		decoder:           protocol.NewDecoder(),
	}
}

// Request sends an RPC request and waits for the response matching rid with timeout.
func (c *Client) Request(ctx context.Context, payload map[string]any) ([]byte, error) {
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
func (c *Client) Connect(ctx context.Context) error {
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

			// Fatal errors that should not reconnect
			if errors.Is(err, protocol.ErrAuthRejected) || errors.Is(err, protocol.ErrRoomNotFound) {
				return err
			}
		}

		if !c.running || !c.AutoReconnect {
			break
		}

		// Exponential backoff with full jitter
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

func (c *Client) changeStatus(s string) {
	if c.OnStatusChange != nil {
		c.OnStatusChange(s)
	}
}

// Close gracefully closes the gateway connection.
func (c *Client) Close() {
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
func (c *Client) Send(payload any) error {
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

func (c *Client) runSession(ctx context.Context) error {
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
			return protocol.ErrAuthRejected
		}
		return fmt.Errorf("failed to dial gateway: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	defer conn.Close()

	// Initial handshake auth frame if required by server format
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

func (c *Client) readPump(conn *websocket.Conn, errChan chan<- error) {
	conn.SetReadLimit(protocol.MaxMessageSize)
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

func (c *Client) writePump(conn *websocket.Conn, errChan chan<- error) {
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

// InboundEnvelope captures generic highrise event types.
type InboundEnvelope struct {
	Type   string          `json:"_type"`
	RID    string          `json:"rid,omitempty"`
	Event  string          `json:"event,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
	RawMap map[string]any  `json:"-"`
}

func (c *Client) handleMessage(msg []byte) {
	// Protocol decoder & validation
	env, err := c.decoder.DecodeEnvelope(msg)
	if err != nil {
		if c.Handlers.OnError != nil {
			c.Handlers.OnError(fmt.Errorf("protocol decode error: %w", err))
		}
		return
	}

	// Dispatch to pending request waiter if matching rid exists
	if env.RID != "" {
		c.pendingMu.Lock()
		ch, exists := c.pending[env.RID]
		c.pendingMu.Unlock()
		if exists {
			ch <- msg
			return
		}
	}

	// Highrise event dispatching
	switch env.Type {
	case "SessionMetadata":
		c.changeStatus("CONNECTED")
		if c.Handlers.OnReady != nil {
			var sm models.SessionMetadata
			if err := c.decoder.DecodeTyped(msg, &sm); err == nil {
				c.Handlers.OnReady(&sm)
			}
		}

	case "ChatEvent":
		if c.Handlers.OnChat != nil {
			var ce models.ChatEvent
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
			var je models.UserJoinedEvent
			if err := c.decoder.DecodeTyped(msg, &je); err == nil {
				c.Handlers.OnUserJoin(&je.User, je.Position)
			}
		}

	case "UserLeftEvent":
		if c.Handlers.OnUserLeave != nil {
			var le models.UserLeftEvent
			if err := c.decoder.DecodeTyped(msg, &le); err == nil {
				c.Handlers.OnUserLeave(&le.User)
			}
		}

	case "UserMovedEvent":
		if c.Handlers.OnUserMove != nil {
			var me models.UserMovedEvent
			if err := c.decoder.DecodeTyped(msg, &me); err == nil {
				c.Handlers.OnUserMove(&me.User, me.Position)
			}
		}

	case "EmoteEvent":
		if c.Handlers.OnEmote != nil {
			var ee models.EmoteEvent
			if err := c.decoder.DecodeTyped(msg, &ee); err == nil {
				c.Handlers.OnEmote(&ee.User, ee.EmoteID, ee.Receiver)
			}
		}

	case "ReactionEvent":
		if c.Handlers.OnReaction != nil {
			var re models.ReactionEvent
			if err := c.decoder.DecodeTyped(msg, &re); err == nil {
				c.Handlers.OnReaction(&re.User, re.Reaction, &re.Receiver)
			}
		}

	case "TipReactionEvent":
		if c.Handlers.OnTip != nil {
			var te models.TipReactionEvent
			if err := c.decoder.DecodeTyped(msg, &te); err == nil {
				c.Handlers.OnTip(&te.Sender, &te.Receiver, te.Item)
			}
		}

	case "MessageEvent":
		if c.Handlers.OnMessage != nil {
			var me models.MessageEvent
			if err := c.decoder.DecodeTyped(msg, &me); err == nil {
				c.Handlers.OnMessage(me.UserID, me.ConversationID, me.IsNewConversation)
			}
		}

	case "Error":
		var ee protocol.ErrorPayload
		_ = c.decoder.DecodeTyped(msg, &ee)
		if c.Handlers.OnError != nil {
			c.Handlers.OnError(fmt.Errorf("server error: %s (code: %d)", ee.Message, ee.Code))
		}
	}
}
