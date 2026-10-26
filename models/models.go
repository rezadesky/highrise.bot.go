package models

// User represents a Highrise player profile.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// Position represents 3D coordinates and facing direction in a room.
type Position struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Z      float64 `json:"z"`
	Facing string  `json:"facing,omitempty"` // Facing direction, e.g. "FrontRight", "FrontLeft"
}

// AnchorPosition represents anchored position (like sitting on a chair).
type AnchorPosition struct {
	EntityID string `json:"entity_id"`
	AnchorIx int    `json:"anchor_ix"`
	Facing   string `json:"facing,omitempty"`
}

// RoomInfo contains metadata regarding the current room.
type RoomInfo struct {
	RoomID   string `json:"room_id"`
	RoomName string `json:"room_name,omitempty"`
	OwnerID  string `json:"owner_id,omitempty"`
}

// SessionMetadata represents data received upon successful bot connection.
type SessionMetadata struct {
	UserID        string        `json:"user_id"`
	RoomInfo      RoomInfo      `json:"room_info"`
	RateLimits    RateLimits    `json:"rate_limits,omitempty"`
	ConnectionID  string        `json:"connection_id,omitempty"`
	SDKVersion    string        `json:"sdk_version,omitempty"`
}

// RateLimits indicates current client rate limits.
type RateLimits struct {
	Client  []float64 `json:"client,omitempty"`
	Socials []float64 `json:"socials,omitempty"`
}

// Item represents a Highrise virtual item/clothing/furniture.
type Item struct {
	ID            string `json:"id"`
	Type          string `json:"type,omitempty"`
	Amount        int    `json:"amount,omitempty"`
	AccountBound  bool   `json:"account_bound,omitempty"`
	ActivePalette *int   `json:"active_palette,omitempty"`
	Category      string `json:"category,omitempty"`
	ItemName      string `json:"item_name,omitempty"`
	Rarity        string `json:"rarity,omitempty"`
}

// UserPosition pairs a User with their current Position in a room.
type UserPosition struct {
	User     User     `json:"user"`
	Position Position `json:"position"`
}

// Currency represents in-game wallet currencies (Gold, Bubbles).
type Currency struct {
	Type   string `json:"type"`   // "gold" or "bubbles"
	Amount int    `json:"amount"`
}
