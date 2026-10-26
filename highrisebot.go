package highrisebot

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Bot represents the main Highrise Bot client.
type Bot struct {
	Config     Config
	Token      string
	RoomID     string
	Gateway    *GatewayClient
	WebAPI     *WebAPIClient
	Dispatcher *EventDispatcher
	RoomState  *RoomTracker
	Commands   *CommandRegistry
	Scheduler  *BotScheduler
	Plugins    *PluginManager
	Security   *SecurityManager
	chatQ      *ChatQueue

	Session *SessionMetadata
	status  ConnectionStatus
	mu      sync.RWMutex

	beforeStart  func(roomID string)
	onReady      func(*SessionMetadata)
	onDisconnect func(error)
	onChat       func(*User, string)
	onWhisper    func(*User, string)
	onUserJoin   func(*User, Position)
	onUserLeave  func(*User)
	onUserMove   func(*User, Position)
	onEmote      func(*User, string, *User)
	onReaction   func(*User, string, *User)
	onTip        func(*User, *User, Item)
	onMessage    func(userID, conversationID string, isNew bool)
	onError      func(error)
}

// New creates a new Highrise Bot instance with functional options.
func New(token, roomID string, opts ...Option) *Bot {
	cfg := DefaultConfig()
	cfg.Token = token
	cfg.RoomID = roomID

	for _, opt := range opts {
		opt(&cfg)
	}

	gwOpts := GatewayOptions{
		GatewayURL:        cfg.GatewayURL,
		HandshakeTimeout:  cfg.HandshakeTimeout,
		WriteWait:         cfg.WriteWait,
		PongWait:          cfg.PongWait,
		MaxReconnectDelay: cfg.MaxReconnectDelay,
		AutoReconnect:     cfg.AutoReconnect,
	}

	disp := NewEventDispatcher(1024, 4)
	gw := NewGatewayClient(token, roomID, gwOpts)
	api := NewWebAPIClient(cfg.WebAPIURL)
	tracker := NewRoomTracker()
	cmdReg := NewCommandRegistry("!")
	sched := NewBotScheduler()
	sec := NewSecurityManager()

	bot := &Bot{
		Config:     cfg,
		Token:      token,
		RoomID:     roomID,
		Gateway:    gw,
		WebAPI:     api,
		Dispatcher: disp,
		RoomState:  tracker,
		Commands:   cmdReg,
		Scheduler:  sched,
		Security:   sec,
		status:     StatusDisconnected,
	}

	bot.Plugins = NewPluginManager(bot)

	bot.chatQ = NewChatQueue(500*time.Millisecond, func(m QueuedChatMessage) error {
		if m.Whisper {
			return bot.rawWhisper(m.UserID, m.Content)
		}
		return bot.rawChat(m.Content)
	})

	gw.OnStatusChange = func(s string) {
		bot.mu.Lock()
		bot.status = ConnectionStatus(s)
		bot.mu.Unlock()
	}

	return bot
}

// Status returns the current connection status of the bot.
func (b *Bot) Status() ConnectionStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.status
}

// Connect establishes the gateway connection with validation and context support.
func (b *Bot) Connect(ctx ...context.Context) error {
	if err := b.Config.Validate(); err != nil {
		return err
	}

	b.mu.Lock()
	if b.status == StatusConnected || b.status == StatusConnecting {
		b.mu.Unlock()
		return ErrAlreadyRunning
	}
	b.status = StatusConnecting
	b.mu.Unlock()

	var c context.Context
	if len(ctx) > 0 {
		c = ctx[0]
	} else {
		c = context.Background()
	}

	b.Gateway.Handlers = GatewayHandlers{
		BeforeStart: b.beforeStart,
		OnReady: func(session *SessionMetadata) {
			b.Session = session
			b.Dispatcher.Publish(EventReady, session)
			go func() {
				if users, err := b.GetRoomUsers(); err == nil {
					b.RoomState.SetUsers(users)
				}
			}()
			if b.onReady != nil {
				b.onReady(session)
			}
		},
		OnDisconnect: func(err error) {
			b.mu.Lock()
			b.Session = nil
			b.mu.Unlock()
			if b.onDisconnect != nil {
				b.onDisconnect(err)
			}
		},
		OnChat: func(u *User, msg string) {
			b.Dispatcher.Publish(EventChat, &ChatEvent{User: *u, Message: msg})

			if b.Security.IsDenied(u.ID) {
				return
			}

			ownerID := ""
			if b.Session != nil {
				ownerID = b.Session.RoomInfo.OwnerID
			}
			handled, _ := b.Commands.Execute(context.Background(), *u, msg, false, func(reply string) error {
				return b.Chat(reply)
			}, ownerID)

			if !handled && b.onChat != nil {
				b.onChat(u, msg)
			}
		},
		OnWhisper: func(u *User, msg string) {
			b.Dispatcher.Publish(EventWhisper, &WhisperEvent{User: *u, Message: msg})
			if b.onWhisper != nil {
				b.onWhisper(u, msg)
			}
		},
		OnUserJoin: func(u *User, pos Position) {
			b.RoomState.OnUserJoined(*u, pos)
			b.Dispatcher.Publish(EventUserJoin, &UserJoinedEvent{User: *u, Position: pos})
			if b.onUserJoin != nil {
				b.onUserJoin(u, pos)
			}
		},
		OnUserLeave: func(u *User) {
			b.RoomState.OnUserLeft(u.ID)
			b.Dispatcher.Publish(EventUserLeave, &UserLeftEvent{User: *u})
			if b.onUserLeave != nil {
				b.onUserLeave(u)
			}
		},
		OnUserMove: func(u *User, pos Position) {
			b.RoomState.OnUserMoved(u.ID, pos)
			b.Dispatcher.Publish(EventUserMove, &UserMovedEvent{User: *u, Position: pos})
			if b.onUserMove != nil {
				b.onUserMove(u, pos)
			}
		},
		OnEmote: func(u *User, emoteID string, receiver *User) {
			b.Dispatcher.Publish(EventEmote, &EmoteEvent{User: *u, EmoteID: emoteID, Receiver: receiver})
			if b.onEmote != nil {
				b.onEmote(u, emoteID, receiver)
			}
		},
		OnReaction: func(u *User, reaction string, receiver *User) {
			rec := User{}
			if receiver != nil {
				rec = *receiver
			}
			b.Dispatcher.Publish(EventReaction, &ReactionEvent{User: *u, Reaction: reaction, Receiver: rec})
			if b.onReaction != nil {
				b.onReaction(u, reaction, receiver)
			}
		},
		OnTip: func(sender *User, receiver *User, item Item) {
			rec := User{}
			if receiver != nil {
				rec = *receiver
			}
			b.Dispatcher.Publish(EventTip, &TipReactionEvent{Sender: *sender, Receiver: rec, Item: item})
			if b.onTip != nil {
				b.onTip(sender, receiver, item)
			}
		},
		OnMessage: func(userID, conversationID string, isNew bool) {
			b.Dispatcher.Publish(EventMessage, &MessageEvent{UserID: userID, ConversationID: conversationID, IsNewConversation: isNew})
			if b.onMessage != nil {
				b.onMessage(userID, conversationID, isNew)
			}
		},
		OnError: b.onError,
	}

	return b.Gateway.Connect(c)
}

// Start begins the bot connection lifecycle (alias for Connect).
func (b *Bot) Start(ctx ...context.Context) error {
	return b.Connect(ctx...)
}

// Run starts the bot and blocks until OS interrupt or context completion (graceful shutdown).
func (b *Bot) Run(ctx ...context.Context) error {
	var c context.Context
	var cancel context.CancelFunc

	if len(ctx) > 0 {
		c, cancel = context.WithCancel(ctx[0])
	} else {
		c, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			b.Stop()
			cancel()
		case <-c.Done():
		}
	}()

	return b.Connect(c)
}

// Disconnect closes the active connection gracefully.
func (b *Bot) Disconnect() {
	b.mu.Lock()
	b.Session = nil
	b.status = StatusDisconnected
	b.mu.Unlock()
	b.Scheduler.Stop()
	b.chatQ.Close()
	b.Gateway.Close()
	b.Dispatcher.Close()
}

// Stop gracefully terminates the bot session (alias for Disconnect).
func (b *Bot) Stop() {
	b.Disconnect()
}

// AddMiddleware adds an interceptor middleware to the event pipeline.
func (b *Bot) AddMiddleware(mw Middleware) {
	b.Dispatcher.Use(mw)
}

// On registers a generic handler for any EventType with optional filter and returns an unregistration handle.
func (b *Bot) On(evt EventType, fn HandlerFunc, filter ...FilterFunc) *HandlerRegistration {
	return b.Dispatcher.Register(evt, fn, filter...)
}

// OnUnknown registers a fallback handler for novel or unhandled events.
func (b *Bot) OnUnknown(fn func(rawType string, payload []byte)) {
	b.Dispatcher.OnUnknown(fn)
}

// EventStats returns metrics of events received, dispatched, dropped, and errored.
func (b *Bot) EventStats() EventStats {
	return b.Dispatcher.Stats()
}

// Event Listeners

func (b *Bot) BeforeStart(fn func(roomID string)) {
	b.beforeStart = fn
}

func (b *Bot) OnReady(fn func(*SessionMetadata)) {
	b.onReady = fn
}

func (b *Bot) OnDisconnect(fn func(err error)) {
	b.onDisconnect = fn
}

func (b *Bot) OnChat(fn func(*User, string)) {
	b.onChat = fn
}

func (b *Bot) OnWhisper(fn func(*User, string)) {
	b.onWhisper = fn
}

func (b *Bot) OnUserJoin(fn func(*User, Position)) {
	b.onUserJoin = fn
}

func (b *Bot) OnUserLeave(fn func(*User)) {
	b.onUserLeave = fn
}

func (b *Bot) OnUserMove(fn func(*User, Position)) {
	b.onUserMove = fn
}

func (b *Bot) OnEmote(fn func(*User, string, *User)) {
	b.onEmote = fn
}

func (b *Bot) OnReaction(fn func(*User, string, *User)) {
	b.onReaction = fn
}

func (b *Bot) OnTip(fn func(sender *User, receiver *User, item Item)) {
	b.onTip = fn
}

func (b *Bot) OnMessage(fn func(userID, conversationID string, isNew bool)) {
	b.onMessage = fn
}

func (b *Bot) OnError(fn func(error)) {
	b.onError = fn
}

// GetBotID returns the connected bot's user ID if online.
func (b *Bot) GetBotID() string {
	if b.Session != nil {
		return b.Session.UserID
	}
	return ""
}

// IsConnected returns whether the bot session is actively running.
func (b *Bot) IsConnected() bool {
	return b.Session != nil
}
