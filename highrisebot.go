package highrisebot

import (
	"context"
	"fmt"

	"github.com/rezadesky/highrisebot/gateway"
	"github.com/rezadesky/highrisebot/models"
	"github.com/rezadesky/highrisebot/webapi"
)

// Bot represents the main Highrise Bot client.
type Bot struct {
	Token   string
	RoomID  string
	Gateway *gateway.Client
	WebAPI  *webapi.Client

	// Session data populated once connected
	Session *models.SessionMetadata

	// Direct event callback registrations
	beforeStart  func(roomID string)
	onReady      func(*models.SessionMetadata)
	onDisconnect func(error)
	onChat       func(*models.User, string)
	onWhisper    func(*models.User, string)
	onUserJoin  func(*models.User, models.Position)
	onUserLeave func(*models.User)
	onUserMove  func(*models.User, models.Position)
	onEmote     func(*models.User, string, *models.User)
	onReaction  func(*models.User, string, *models.User)
	onTip       func(*models.User, *models.User, models.Item)
	onMessage   func(userID, conversationID string, isNew bool)
	onError     func(error)
}

// Config allows custom configuration options.
type Config struct {
	GatewayURL string
	WebAPIURL  string
}

// New creates a new Highrise Bot instance with default configurations.
func New(token, roomID string, cfgs ...Config) *Bot {
	var cfg Config
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}

	gw := gateway.NewClient(token, roomID, cfg.GatewayURL)
	api := webapi.NewClient(cfg.WebAPIURL)

	bot := &Bot{
		Token:   token,
		RoomID:  roomID,
		Gateway: gw,
		WebAPI:  api,
	}

	return bot
}

// Start begins connecting to Highrise Gateway and blocks until context is cancelled or stopped.
func (b *Bot) Start(ctx ...context.Context) error {
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
			if b.onReady != nil {
				b.onReady(session)
			}
		},
		OnDisconnect: b.onDisconnect,
		OnChat:       b.onChat,
		OnWhisper:    b.onWhisper,
		OnUserJoin:   b.onUserJoin,
		OnUserLeave: b.onUserLeave,
		OnUserMove:  b.onUserMove,
		OnEmote:     b.onEmote,
		OnReaction:  b.onReaction,
		OnTip:       b.onTip,
		OnMessage:   b.onMessage,
		OnError:     b.onError,
	}

	return b.Gateway.Connect(c)
}

// Stop gracefully terminates the bot session.
func (b *Bot) Stop() {
	b.Gateway.Close()
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
