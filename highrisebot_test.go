package highrisebot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rezadesky/highrisebot"
	"github.com/rezadesky/highrisebot/models"
)

func TestCoreClientCreationAndOptions(t *testing.T) {
	// Test validation
	invalidBot := highrisebot.New("", "")
	err := invalidBot.Connect()
	if !errors.Is(err, highrisebot.ErrMissingToken) {
		t.Fatalf("expected ErrMissingToken, got %v", err)
	}

	invalidBot2 := highrisebot.New("token_ok", "")
	err2 := invalidBot2.Connect()
	if !errors.Is(err2, highrisebot.ErrMissingRoomID) {
		t.Fatalf("expected ErrMissingRoomID, got %v", err2)
	}

	// Test functional options
	bot := highrisebot.New(
		"test_token",
		"test_room",
		highrisebot.WithGatewayURL("wss://custom.gateway.url"),
		highrisebot.WithWebAPIURL("https://custom.api.url"),
		highrisebot.WithHandshakeTimeout(5*time.Second),
		highrisebot.WithMaxReconnectDelay(10*time.Second),
		highrisebot.WithAutoReconnect(false),
	)

	if bot.Config.Token != "test_token" {
		t.Errorf("expected token test_token, got %s", bot.Config.Token)
	}
	if bot.Config.GatewayURL != "wss://custom.gateway.url" {
		t.Errorf("unexpected gateway URL: %s", bot.Config.GatewayURL)
	}
	if bot.Config.AutoReconnect != false {
		t.Errorf("expected AutoReconnect to be false")
	}

	// Test Initial Status
	if bot.Status() != highrisebot.StatusDisconnected {
		t.Errorf("expected initial status DISCONNECTED, got %s", bot.Status())
	}
}

func TestLifecycleHooksAndGracefulShutdown(t *testing.T) {
	bot := highrisebot.New(
		"test_token",
		"test_room",
		highrisebot.WithAutoReconnect(false),
	)

	beforeStartTriggered := false
	bot.BeforeStart(func(roomID string) {
		beforeStartTriggered = true
	})

	bot.OnReady(func(s *models.SessionMetadata) {})
	bot.OnDisconnect(func(err error) {})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Connect with cancellation context
	_ = bot.Connect(ctx)

	if !beforeStartTriggered {
		t.Errorf("expected BeforeStart hook to be triggered")
	}

	// Graceful disconnect test
	bot.Disconnect()
	if bot.Status() != highrisebot.StatusDisconnected {
		t.Errorf("expected status to be DISCONNECTED after Disconnect(), got %s", bot.Status())
	}
}
