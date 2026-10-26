package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rezadesky/highrisebot/models"
)

const (
	DefaultGatewayURL = "wss://highrise.game/web/botapi"
	writeWait         = 10 * time.Second
	pongWait          = 60 * time.Second
	pingPeriod        = (pongWait * 9) / 10
	maxReconnectDelay = 30 * time.Second
)

// HandlerFuncs holds all dispatch callbacks.
type HandlerFuncs struct {
	OnReady     func(*models.SessionMetadata)
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
	GatewayURL string
	Token      string
	RoomID     string
	Handlers   HandlerFuncs

	conn      *websocket.Conn
	mu        sync.Mutex
	writeChan chan []byte
	stopChan  chan struct{}
	closeOnce sync.Once
	running   bool

	// Request tracking for RPC responses (rid -> response channel)
	pendingMu sync.Mutex
	pending   map[string]chan []byte
}

// NewClient initializes a new Gateway Client instance.
func NewClient(token, roomID, gatewayURL string) *Client {
	if gatewayURL == "" {
		gatewayURL = DefaultGatewayURL
	}
	return &Client{
		GatewayURL: gatewayURL,
		Token:      token,
		RoomID:     roomID,
		writeChan:  make(chan []byte, 128),
		stopChan:   make(chan struct{}),
		pending:    make(map[string]chan []byte),
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
	backoff := 1 * time.Second

	for c.running {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopChan:
			return nil
		default:
		}

		err := c.runSession(ctx)
		if err != nil && c.Handlers.OnError != nil {
			c.Handlers.OnError(fmt.Errorf("gateway error: %w", err))
		}

		if !c.running {
			break
		}

		log.Printf("[highrisebot] Reconnecting in %v...", backoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopChan:
			return nil
		}

		backoff *= 2
		if backoff > maxReconnectDelay {
			backoff = maxReconnectDelay
		}
	}
	return nil
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
	data, err := json.Marshal(payload)
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
		HandshakeTimeout: 15 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, c.GatewayURL, headers)
	if err != nil {
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
	conn.SetReadLimit(512 * 1024)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
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
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case message, ok := <-c.writeChan:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				errChan <- err
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
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
	var env InboundEnvelope
	if err := json.Unmarshal(msg, &env); err != nil {
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
		if c.Handlers.OnReady != nil {
			var sm models.SessionMetadata
			if err := json.Unmarshal(msg, &sm); err == nil {
				c.Handlers.OnReady(&sm)
			}
		}

	case "ChatEvent":
		if c.Handlers.OnChat != nil {
			var ce models.ChatEvent
			if err := json.Unmarshal(msg, &ce); err == nil {
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
			if err := json.Unmarshal(msg, &je); err == nil {
				c.Handlers.OnUserJoin(&je.User, je.Position)
			}
		}

	case "UserLeftEvent":
		if c.Handlers.OnUserLeave != nil {
			var le models.UserLeftEvent
			if err := json.Unmarshal(msg, &le); err == nil {
				c.Handlers.OnUserLeave(&le.User)
			}
		}

	case "UserMovedEvent":
		if c.Handlers.OnUserMove != nil {
			var me models.UserMovedEvent
			if err := json.Unmarshal(msg, &me); err == nil {
				c.Handlers.OnUserMove(&me.User, me.Position)
			}
		}

	case "EmoteEvent":
		if c.Handlers.OnEmote != nil {
			var ee models.EmoteEvent
			if err := json.Unmarshal(msg, &ee); err == nil {
				c.Handlers.OnEmote(&ee.User, ee.EmoteID, ee.Receiver)
			}
		}

	case "ReactionEvent":
		if c.Handlers.OnReaction != nil {
			var re models.ReactionEvent
			if err := json.Unmarshal(msg, &re); err == nil {
				c.Handlers.OnReaction(&re.User, re.Reaction, &re.Receiver)
			}
		}

	case "TipReactionEvent":
		if c.Handlers.OnTip != nil {
			var te models.TipReactionEvent
			if err := json.Unmarshal(msg, &te); err == nil {
				c.Handlers.OnTip(&te.Sender, &te.Receiver, te.Item)
			}
		}

	case "MessageEvent":
		if c.Handlers.OnMessage != nil {
			var me models.MessageEvent
			if err := json.Unmarshal(msg, &me); err == nil {
				c.Handlers.OnMessage(me.UserID, me.ConversationID, me.IsNewConversation)
			}
		}

	case "Error":
		if c.Handlers.OnError != nil {
			var ee models.ErrorEvent
			if err := json.Unmarshal(msg, &ee); err == nil {
				c.Handlers.OnError(fmt.Errorf("server error: %s (code: %d)", ee.Message, ee.Code))
			}
		}
	}
}
