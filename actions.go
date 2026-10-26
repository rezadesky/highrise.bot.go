package highrisebot

import (
	"context"
	"encoding/json"
	"errors"
)

// Chat sends a public message in the room via rate-limited queue with length validation.
func (b *Bot) Chat(message string) error {
	return b.chatQ.Enqueue(message, false, "")
}

// Whisper sends a private message to a specific user via rate-limited queue with validation.
func (b *Bot) Whisper(userID, message string) error {
	return b.chatQ.Enqueue(message, true, userID)
}

func (b *Bot) rawChat(message string) error {
	payload := map[string]any{
		"_type":   "ChatRequest",
		"message": message,
	}
	return b.Gateway.Send(payload)
}

func (b *Bot) rawWhisper(userID, message string) error {
	payload := map[string]any{
		"_type":   "ChatRequest",
		"message": message,
		"whisper": true,
		"user_id": userID,
	}
	return b.Gateway.Send(payload)
}

// Sit anchors the avatar to a chair or seat entity in the room.
func (b *Bot) Sit(entityID string, anchorIx int, facing ...string) error {
	if entityID == "" {
		return errors.New("sit: entityID is required")
	}
	face := "FrontRight"
	if len(facing) > 0 && facing[0] != "" {
		face = facing[0]
	}
	payload := map[string]any{
		"_type": "MoveUserRequest",
		"position": map[string]any{
			"entity_id": entityID,
			"anchor_ix": anchorIx,
			"facing":    face,
		},
	}
	return b.Gateway.Send(payload)
}

// Unban removes a room ban from a user.
func (b *Bot) Unban(userID string) error {
	if userID == "" {
		return errors.New("unban: target userID is required")
	}
	payload := map[string]any{
		"_type":   "ModerateRoomRequest",
		"user_id": userID,
		"action":  "unban",
	}
	return b.Gateway.Send(payload)
}

// Walk moves the bot avatar to a target coordinate in the room with coordinate validation.
func (b *Bot) Walk(x, y, z float64, facing ...string) error {
	if x < 0 || y < 0 || z < 0 {
		return errors.New("walk: coordinates must be positive numbers")
	}
	face := "FrontRight"
	if len(facing) > 0 && facing[0] != "" {
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

// Teleport moves a specific user (or bot itself) immediately to coordinates with validation.
func (b *Bot) Teleport(userID string, x, y, z float64, facing ...string) error {
	if userID == "" {
		return errors.New("teleport: target userID is required")
	}
	if x < 0 || y < 0 || z < 0 {
		return errors.New("teleport: coordinates must be positive numbers")
	}
	face := "FrontRight"
	if len(facing) > 0 && facing[0] != "" {
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
func (b *Bot) GetRoomUsers(ctx ...context.Context) ([]UserPosition, error) {
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

	results := make([]UserPosition, 0, len(resp.Content))
	for _, pair := range resp.Content {
		if len(pair) >= 2 {
			var u User
			var p Position
			_ = json.Unmarshal(pair[0], &u)
			_ = json.Unmarshal(pair[1], &p)
			results = append(results, UserPosition{User: u, Position: p})
		}
	}
	return results, nil
}

// GetWallet retrieves the current currency balances of the bot (gold, tokens).
func (b *Bot) GetWallet(ctx ...context.Context) ([]Currency, error) {
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
		Content []Currency `json:"content"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Content, nil
}

// GetInventory retrieves the bot's avatar clothing inventory.
func (b *Bot) GetInventory(ctx ...context.Context) ([]Item, error) {
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
		Items []Item `json:"items"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// SetOutfit updates the bot's avatar outfit with a list of item IDs.
func (b *Bot) SetOutfit(outfitItems []Item) error {
	payload := map[string]any{
		"_type":  "SetOutfitRequest",
		"outfit": outfitItems,
	}
	return b.Gateway.Send(payload)
}

// TipUser tips a user in the room with gold bars.
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
func (b *Bot) GetVoiceStatus(ctx ...context.Context) (*VoiceStatus, error) {
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
		VoiceStatus VoiceStatus `json:"voice_status"`
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
func (b *Bot) GetConversations(notJoined bool, lastID string, ctx ...context.Context) ([]Conversation, error) {
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
		Conversations []Conversation `json:"conversations"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Conversations, nil
}

// GetMessages retrieves messages from an inbox conversation.
func (b *Bot) GetMessages(conversationID, lastID string, ctx ...context.Context) ([]DirectMessage, error) {
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
		Messages []DirectMessage `json:"messages"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Messages, nil
}
