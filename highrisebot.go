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

	// Direct event callback registrations
	onReady     func(*models.SessionMetadata)
	onChat      func(*models.User, string)
	onWhisper   func(*models.User, string)
	onUserJoin  func(*models.User, models.Position)
	onUserLeave func(*models.User)
	onUserMove  func(*models.User, models.Position)
	onEmote     func(*models.User, string, *models.User)
	onReaction  func(*models.User, string, *models.User)
	onTip       func(*models.User, *models.User, models.Item)
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
		OnReady:     b.onReady,
		OnChat:      b.onChat,
		OnWhisper:   b.onWhisper,
		OnUserJoin:  b.onUserJoin,
		OnUserLeave: b.onUserLeave,
		OnUserMove:  b.onUserMove,
		OnEmote:     b.onEmote,
		OnReaction:  b.onReaction,
		OnTip:       b.onTip,
		OnError:     b.onError,
	}

	return b.Gateway.Connect(c)
}

// Stop gracefully terminates the bot session.
func (b *Bot) Stop() {
	b.Gateway.Close()
}

// Event Listeners

func (b *Bot) OnReady(fn func(*models.SessionMetadata)) {
	b.onReady = fn
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

func (b *Bot) OnError(fn func(error)) {
	b.onError = fn
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

// Export models directly as alias for convenience
type (
	User            = models.User
	Position        = models.Position
	UserPosition    = models.UserPosition
	RoomInfo        = models.RoomInfo
	SessionMetadata = models.SessionMetadata
	Item            = models.Item
	Currency        = models.Currency
)
