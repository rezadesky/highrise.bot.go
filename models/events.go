package models

// ChatEvent represents a public room chat event.
type ChatEvent struct {
	User      User   `json:"user"`
	Message   string `json:"message"`
	Whisper   bool   `json:"whisper,omitempty"`
	MessageID string `json:"message_id,omitempty"`
}

// WhisperEvent represents a private direct whisper sent to the bot.
type WhisperEvent struct {
	User    User   `json:"user"`
	Message string `json:"message"`
}

// UserJoinedEvent represents when a user enters the room.
type UserJoinedEvent struct {
	User     User     `json:"user"`
	Position Position `json:"position"`
}

// UserLeftEvent represents when a user leaves the room.
type UserLeftEvent struct {
	User User `json:"user"`
}

// UserMovedEvent represents when a user walks or teleports to a new coordinate.
type UserMovedEvent struct {
	User     User     `json:"user"`
	Position Position `json:"position"`
}

// EmoteEvent represents when a user performs an emote.
type EmoteEvent struct {
	User     User   `json:"user"`
	EmoteID  string `json:"emote_id"`
	Receiver *User  `json:"receiver,omitempty"`
}

// ReactionEvent represents when a reaction emoji is sent towards a user.
type ReactionEvent struct {
	User     User   `json:"user"`
	Reaction string `json:"reaction"` // e.g. "heart", "clap", "thumbs"
	Receiver User   `json:"receiver"`
}

// TipReactionEvent represents when a tip is given to another user.
type TipReactionEvent struct {
	Sender   User   `json:"sender"`
	Receiver User   `json:"receiver"`
	Item     Item   `json:"item"`
}

// VoiceEvent represents voice chat state updates.
type VoiceEvent struct {
	Users  []User `json:"users"`
	Status string `json:"status"` // "speaking", "muted", etc.
}

// ErrorEvent represents error payloads dispatched from Highrise servers.
type ErrorEvent struct {
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}
