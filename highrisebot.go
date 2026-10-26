package highrisebot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rezadesky/highrisebot/events"
	"github.com/rezadesky/highrisebot/gateway"
	"github.com/rezadesky/highrisebot/models"
	"github.com/rezadesky/highrisebot/webapi"
)

// Bot represents the main Highrise Bot client.
type Bot struct {
	Config     Config
	Token      string
	RoomID     string
	Gateway    *gateway.Client
	WebAPI     *webapi.Client
	Dispatcher *events.Dispatcher

	// Session data populated once connected
	Session *models.SessionMetadata
	status  ConnectionStatus
	mu      sync.RWMutex

	// Direct event callback registrations
	beforeStart  func(roomID string)
	onReady      func(*models.SessionMetadata)
	onDisconnect func(error)
	onChat       func(*models.User, string)
	onWhisper    func(*models.User, string)
	onUserJoin   func(*models.User, models.Position)
	onUserLeave  func(*models.User)
	onUserMove   func(*models.User, models.Position)
	onEmote      func(*models.User, string, *models.User)
	onReaction   func(*models.User, string, *models.User)
	onTip        func(*models.User, *models.User, models.Item)
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

	gwOpts := gateway.GatewayOptions{
		GatewayURL:        cfg.GatewayURL,
		HandshakeTimeout:  cfg.HandshakeTimeout,
		WriteWait:         cfg.WriteWait,
		PongWait:          cfg.PongWait,
		MaxReconnectDelay: cfg.MaxReconnectDelay,
		AutoReconnect:     cfg.AutoReconnect,
	}

	disp := events.NewDispatcher()
	gw := gateway.NewClient(token, roomID, gwOpts)
	api := webapi.NewClient(cfg.WebAPIURL)

	bot := &Bot{
		Config:     cfg,
		Token:      token,
		RoomID:     roomID,
		Gateway:    gw,
		WebAPI:     api,
		Dispatcher: disp,
		status:     StatusDisconnected,
	}

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

	// Wire user handlers to gateway handlers
	b.Gateway.Handlers = gateway.HandlerFuncs{
		BeforeStart: b.beforeStart,
		OnReady: func(session *models.SessionMetadata) {
			b.Session = session
			b.Dispatcher.Publish(events.EventReady, session)
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
		OnChat: func(u *models.User, msg string) {
			b.Dispatcher.Publish(events.EventChat, &models.ChatEvent{User: *u, Message: msg})
			if b.onChat != nil {
				b.onChat(u, msg)
			}
		},
		OnWhisper: func(u *models.User, msg string) {
			b.Dispatcher.Publish(events.EventWhisper, &models.WhisperEvent{User: *u, Message: msg})
			if b.onWhisper != nil {
				b.onWhisper(u, msg)
			}
		},
		OnUserJoin: func(u *models.User, pos models.Position) {
			b.Dispatcher.Publish(events.EventUserJoin, &models.UserJoinedEvent{User: *u, Position: pos})
			if b.onUserJoin != nil {
				b.onUserJoin(u, pos)
			}
		},
		OnUserLeave: func(u *models.User) {
			b.Dispatcher.Publish(events.EventUserLeave, &models.UserLeftEvent{User: *u})
			if b.onUserLeave != nil {
				b.onUserLeave(u)
			}
		},
		OnUserMove: func(u *models.User, pos models.Position) {
			b.Dispatcher.Publish(events.EventUserMove, &models.UserMovedEvent{User: *u, Position: pos})
			if b.onUserMove != nil {
				b.onUserMove(u, pos)
			}
		},
		OnEmote: func(u *models.User, emoteID string, receiver *models.User) {
			b.Dispatcher.Publish(events.EventEmote, &models.EmoteEvent{User: *u, EmoteID: emoteID, Receiver: receiver})
			if b.onEmote != nil {
				b.onEmote(u, emoteID, receiver)
			}
		},
		OnReaction: func(u *models.User, reaction string, receiver *models.User) {
			rec := models.User{}
			if receiver != nil {
				rec = *receiver
			}
			b.Dispatcher.Publish(events.EventReaction, &models.ReactionEvent{User: *u, Reaction: reaction, Receiver: rec})
			if b.onReaction != nil {
				b.onReaction(u, reaction, receiver)
			}
		},
		OnTip: func(sender *models.User, receiver *models.User, item models.Item) {
			rec := models.User{}
			if receiver != nil {
				rec = *receiver
			}
			b.Dispatcher.Publish(events.EventTip, &models.TipReactionEvent{Sender: *sender, Receiver: rec, Item: item})
			if b.onTip != nil {
				b.onTip(sender, receiver, item)
			}
		},
		OnMessage: func(userID, conversationID string, isNew bool) {
			b.Dispatcher.Publish(events.EventMessage, &models.MessageEvent{UserID: userID, ConversationID: conversationID, IsNewConversation: isNew})
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
	b.Gateway.Close()
	b.Dispatcher.Close()
}

// Stop gracefully terminates the bot session (alias for Disconnect).
func (b *Bot) Stop() {
	b.Disconnect()
}

// AddMiddleware adds an interceptor middleware to the event pipeline.
func (b *Bot) AddMiddleware(mw events.Middleware) {
	b.Dispatcher.Use(mw)
}

// On registers a generic handler for any EventType with optional filter and returns an unregistration handle.
func (b *Bot) On(evt events.EventType, fn events.HandlerFunc, filter ...events.FilterFunc) *events.HandlerRegistration {
	return b.Dispatcher.Register(evt, fn, filter...)
}

// OnUnknown registers a fallback handler for novel or unhandled events.
func (b *Bot) OnUnknown(fn func(rawType string, payload []byte)) {
	b.Dispatcher.OnUnknown(fn)
}

// EventStats returns metrics of events received, dispatched, dropped, and errored.
func (b *Bot) EventStats() events.Stats {
	return b.Dispatcher.Stats()
}

// Event Listeners

func (b *Bot) BeforeStart(fn func(roomID string)) {
	b.beforeStart = fn
}

func (b *Bot) OnReady(fn func(*models.SessionMetadata)) {
	b.onReady = fn
}

func (b *Bot) OnDisconnect(fn func(err error)) {
	b.onDisconnect = fn
}

func (b *Bot) OnChat(fn func(*models.User, string)) {
	b.onChat = fn
}

func (b *Bot) OnWhisper(fn func(*models.User, string)) {
	b.onWhisper = fn
}

func (b *Bot) OnUserJoin(fn func(*models.User, models.Position)) {
	b.onUserJoin = fn
}

func (b *Bot) OnUserLeave(fn func(*models.User)) {
	b.onUserLeave = fn
}

func (b *Bot) OnUserMove(fn func(*models.User, models.Position)) {
	b.onUserMove = fn
}

func (b *Bot) OnEmote(fn func(*models.User, string, *models.User)) {
	b.onEmote = fn
}

func (b *Bot) OnReaction(fn func(*models.User, string, *models.User)) {
	b.onReaction = fn
}

func (b *Bot) OnTip(fn func(sender *models.User, receiver *models.User, item models.Item)) {
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

// Outgoing Bot Actions

// Chat sends a public message in the room.
func (b *Bot) Chat(message string) error {
	payload := map[string]any{
		"_type":   "ChatRequest",
		"message": message,
	}
	return b.Gateway.Send(payload)
}

// Whisper sends a private message to a specific user.
func (b *Bot) Whisper(userID, message string) error {
	payload := map[string]any{
		"_type":   "ChatRequest",
		"message": message,
		"whisper": true,
		"user_id": userID,
	}
	return b.Gateway.Send(payload)
}

// Walk moves the bot avatar to a target coordinate in the room.
func (b *Bot) Walk(x, y, z float64, facing ...string) error {
	face := "FrontRight"
	if len(facing) > 0 {
		face = facing[0]
	}
	payload := map[string]any{
		"_type": "MoveUserRequest",
		"position": map[string]any{
			"x":      x,
			"y":      y,
			"z":      z,
			"facing": face,
		},
	}
	return b.Gateway.Send(payload)
}

// Teleport moves a specific user (or bot itself) immediately to coordinates.
func (b *Bot) Teleport(userID string, x, y, z float64, facing ...string) error {
	face := "FrontRight"
	if len(facing) > 0 {
		face = facing[0]
	}
	payload := map[string]any{
		"_type":   "TeleportRequest",
		"user_id": userID,
		"position": map[string]any{
			"x":      x,
			"y":      y,
			"z":      z,
			"facing": face,
		},
	}
	return b.Gateway.Send(payload)
}

// Emote triggers an emote animation on the bot or a target user.
func (b *Bot) Emote(emoteID string, targetUserID ...string) error {
	payload := map[string]any{
		"_type":    "EmoteRequest",
		"emote_id": emoteID,
	}
	if len(targetUserID) > 0 && targetUserID[0] != "" {
		payload["target_user_id"] = targetUserID[0]
	}
	return b.Gateway.Send(payload)
}

// React sends an emoji reaction to a user.
func (b *Bot) React(reaction, targetUserID string) error {
	payload := map[string]any{
		"_type":          "ReactionRequest",
		"reaction":       reaction,
		"target_user_id": targetUserID,
	}
	return b.Gateway.Send(payload)
}

// Kick kicks a user from the room.
func (b *Bot) Kick(userID string) error {
	payload := map[string]any{
		"_type":   "ModerateRoomRequest",
		"user_id": userID,
		"action":  "kick",
	}
	return b.Gateway.Send(payload)
}

// Mute mutes a user in the room for a duration in seconds.
func (b *Bot) Mute(userID string, durationSeconds int) error {
	payload := map[string]any{
		"_type":    "ModerateRoomRequest",
		"user_id":  userID,
		"action":   "mute",
		"duration": durationSeconds,
	}
	return b.Gateway.Send(payload)
}

// Ban bans a user from the room.
func (b *Bot) Ban(userID string, durationSeconds int) error {
	payload := map[string]any{
		"_type":    "ModerateRoomRequest",
		"user_id":  userID,
		"action":   "ban",
		"duration": durationSeconds,
	}
	return b.Gateway.Send(payload)
}

// GetRoomUsers fetches all users currently in the room along with their positions.
func (b *Bot) GetRoomUsers(ctx ...context.Context) ([]models.UserPosition, error) {
	c := context.Background()
	if len(ctx) > 0 {
		c = ctx[0]
	}

	payload := map[string]any{
		"_type": "GetRoomUsersRequest",
	}

	data, err := b.Gateway.Request(c, payload)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Content [][]json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	results := make([]models.UserPosition, 0, len(resp.Content))
	for _, pair := range resp.Content {
		if len(pair) >= 2 {
			var u models.User
			var p models.Position
			_ = json.Unmarshal(pair[0], &u)
			_ = json.Unmarshal(pair[1], &p)
			results = append(results, models.UserPosition{User: u, Position: p})
		}
	}
	return results, nil
}

// GetWallet retrieves the current currency balances of the bot (gold, tokens).
func (b *Bot) GetWallet(ctx ...context.Context) ([]models.Currency, error) {
	c := context.Background()
	if len(ctx) > 0 {
		c = ctx[0]
	}

	payload := map[string]any{
		"_type": "GetWalletRequest",
	}

	data, err := b.Gateway.Request(c, payload)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Content []models.Currency `json:"content"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Content, nil
}

// GetInventory retrieves the bot's avatar clothing inventory.
func (b *Bot) GetInventory(ctx ...context.Context) ([]models.Item, error) {
	c := context.Background()
	if len(ctx) > 0 {
		c = ctx[0]
	}

	payload := map[string]any{
		"_type": "GetInventoryRequest",
	}

	data, err := b.Gateway.Request(c, payload)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Items []models.Item `json:"items"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// SetOutfit updates the bot's avatar outfit with a list of item IDs.
func (b *Bot) SetOutfit(outfitItems []models.Item) error {
	payload := map[string]any{
		"_type":  "SetOutfitRequest",
		"outfit": outfitItems,
	}
	return b.Gateway.Send(payload)
}

// TipUser tips a user in the room with gold bars.
// Valid bar denominations: "gold_bar_1", "gold_bar_5", "gold_bar_10", "gold_bar_50", "gold_bar_100", "gold_bar_500", "gold_bar_1k"
func (b *Bot) TipUser(userID, barAmount string) error {
	payload := map[string]any{
		"_type":   "TipUserRequest",
		"user_id": userID,
		"bar":     barAmount,
	}
	return b.Gateway.Send(payload)
}

// BuyItem purchases an item using the bot's wallet.
func (b *Bot) BuyItem(itemID string) error {
	payload := map[string]any{
		"_type":   "BuyItemRequest",
		"item_id": itemID,
	}
	return b.Gateway.Send(payload)
}

// BuyVoiceTime buys voice chat time for the room using tokens or gold.
// paymentType can be "bot_wallet_only", "bot_wallet_priority", or "user_wallet_only".
func (b *Bot) BuyVoiceTime(paymentType string) error {
	if paymentType == "" {
		paymentType = "bot_wallet_only"
	}
	payload := map[string]any{
		"_type":        "BuyVoiceTimeRequest",
		"payment_type": paymentType,
	}
	return b.Gateway.Send(payload)
}

// GetVoiceStatus returns current room voice chat details.
func (b *Bot) GetVoiceStatus(ctx ...context.Context) (*models.VoiceStatus, error) {
	c := context.Background()
	if len(ctx) > 0 {
		c = ctx[0]
	}

	payload := map[string]any{
		"_type": "GetVoiceStatusRequest",
	}

	data, err := b.Gateway.Request(c, payload)
	if err != nil {
		return nil, err
	}

	var resp struct {
		VoiceStatus models.VoiceStatus `json:"voice_status"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp.VoiceStatus, nil
}

// AddUserToVoice invites/adds a user to room voice chat.
func (b *Bot) AddUserToVoice(userID string) error {
	payload := map[string]any{
		"_type":   "AddUserToVoiceRequest",
		"user_id": userID,
	}
	return b.Gateway.Send(payload)
}

// RemoveUserFromVoice removes a user from room voice chat.
func (b *Bot) RemoveUserFromVoice(userID string) error {
	payload := map[string]any{
		"_type":   "RemoveUserFromVoiceRequest",
		"user_id": userID,
	}
	return b.Gateway.Send(payload)
}

// SendDirectMessage sends a message in an inbox direct conversation.
// msgType can be "text" or "invite" (if "invite", pass roomID).
func (b *Bot) SendDirectMessage(conversationID, message, msgType string, roomID ...string) error {
	if msgType == "" {
		msgType = "text"
	}
	payload := map[string]any{
		"_type":           "SendMessageRequest",
		"conversation_id": conversationID,
		"message":         message,
		"type":            msgType,
	}
	if len(roomID) > 0 && roomID[0] != "" {
		payload["room_id"] = roomID[0]
	}
	return b.Gateway.Send(payload)
}

// GetConversations retrieves recent direct message inbox threads.
func (b *Bot) GetConversations(notJoined bool, lastID string, ctx ...context.Context) ([]models.Conversation, error) {
	c := context.Background()
	if len(ctx) > 0 {
		c = ctx[0]
	}

	payload := map[string]any{
		"_type":      "GetConversationsRequest",
		"not_joined": notJoined,
	}
	if lastID != "" {
		payload["last_id"] = lastID
	}

	data, err := b.Gateway.Request(c, payload)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Conversations []models.Conversation `json:"conversations"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Conversations, nil
}

// GetMessages retrieves messages from an inbox conversation.
func (b *Bot) GetMessages(conversationID, lastID string, ctx ...context.Context) ([]models.DirectMessage, error) {
	c := context.Background()
	if len(ctx) > 0 {
		c = ctx[0]
	}

	payload := map[string]any{
		"_type":           "GetMessagesRequest",
		"conversation_id": conversationID,
	}
	if lastID != "" {
		payload["last_id"] = lastID
	}

	data, err := b.Gateway.Request(c, payload)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Messages []models.DirectMessage `json:"messages"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Messages, nil
}

// Export models directly as alias for convenience
type (
	User            = models.User
	Position        = models.Position
	UserPosition    = models.UserPosition
	RoomInfo        = models.RoomInfo
	SessionMetadata = models.SessionMetadata
	Item            = models.Item
	Currency        = models.Currency
	VoiceStatus     = models.VoiceStatus
	Conversation    = models.Conversation
	DirectMessage   = models.DirectMessage
)
