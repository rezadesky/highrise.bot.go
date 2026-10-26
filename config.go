package highrisebot

import (
	"errors"
	"time"
)

var (
	// ErrMissingToken is returned when API token is empty.
	ErrMissingToken = errors.New("highrisebot: api token is required")

	// ErrMissingRoomID is returned when room ID is empty.
	ErrMissingRoomID = errors.New("highrisebot: room id is required")

	// ErrAlreadyRunning is returned when attempting to start an already running bot.
	ErrAlreadyRunning = errors.New("highrisebot: bot is already running")

	// ErrNotConnected is returned when trying to perform room actions while offline.
	ErrNotConnected = errors.New("highrisebot: bot is not connected to room")
)

// ConnectionStatus represents the lifecycle connection state of the bot.
type ConnectionStatus string

const (
	StatusDisconnected ConnectionStatus = "DISCONNECTED"
	StatusConnecting   ConnectionStatus = "CONNECTING"
	StatusConnected    ConnectionStatus = "CONNECTED"
	StatusReconnecting ConnectionStatus = "RECONNECTING"
	StatusStopped      ConnectionStatus = "STOPPED"
)

// Config defines configurable parameters for Highrise Bot client.
type Config struct {
	Token              string
	RoomID             string
	GatewayURL         string
	WebAPIURL          string
	HandshakeTimeout   time.Duration
	WriteWait          time.Duration
	PongWait           time.Duration
	MaxReconnectDelay  time.Duration
	AutoReconnect      bool
}

// DefaultConfig returns default production settings.
func DefaultConfig() Config {
	return Config{
		GatewayURL:        "wss://highrise.game/web/botapi",
		WebAPIURL:         "https://webapi.highrise.game",
		HandshakeTimeout:  15 * time.Second,
		WriteWait:         10 * time.Second,
		PongWait:          60 * time.Second,
		MaxReconnectDelay: 30 * time.Second,
		AutoReconnect:     true,
	}
}

// Option is a functional option for configuring a Bot client.
type Option func(*Config)

// WithGatewayURL sets a custom WebSocket Gateway URL.
func WithGatewayURL(url string) Option {
	return func(c *Config) {
		if url != "" {
			c.GatewayURL = url
		}
	}
}

// WithWebAPIURL sets a custom Web API base URL.
func WithWebAPIURL(url string) Option {
	return func(c *Config) {
		if url != "" {
			c.WebAPIURL = url
		}
	}
}

// WithHandshakeTimeout sets custom timeout for WebSocket dial/handshake.
func WithHandshakeTimeout(d time.Duration) Option {
	return func(c *Config) {
		if d > 0 {
			c.HandshakeTimeout = d
		}
	}
}

// WithMaxReconnectDelay sets the ceiling for exponential backoff delay.
func WithMaxReconnectDelay(d time.Duration) Option {
	return func(c *Config) {
		if d > 0 {
			c.MaxReconnectDelay = d
		}
	}
}

// WithAutoReconnect toggles the automatic reconnection loop.
func WithAutoReconnect(enabled bool) Option {
	return func(c *Config) {
		c.AutoReconnect = enabled
	}
}

// Validate checks whether the configuration is valid.
func (c *Config) Validate() error {
	if c.Token == "" {
		return ErrMissingToken
	}
	if c.RoomID == "" {
		return ErrMissingRoomID
	}
	return nil
}
