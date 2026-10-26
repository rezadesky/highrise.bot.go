package highrisebot_test

import (
	"context"
	"testing"
	"time"

	"github.com/rezadesky/highrisebot"
	"github.com/rezadesky/highrisebot/models"
)

func TestNewBotInstance(t *testing.T) {
	token := "dummy_token"
	roomID := "dummy_room_id"

	bot := highrisebot.New(token, roomID)
	if bot == nil {
		t.Fatal("expected bot instance to be created, got nil")
	}

	if bot.Token != token {
		t.Errorf("expected token %s, got %s", token, bot.Token)
	}

	if bot.RoomID != roomID {
		t.Errorf("expected roomID %s, got %s", roomID, bot.RoomID)
	}
}

func TestEventHandlersRegistration(t *testing.T) {
	bot := highrisebot.New("dummy_token", "dummy_room")

	readyCalled := false
	bot.OnReady(func(s *models.SessionMetadata) {
		readyCalled = true
	})

	chatCalled := false
	bot.OnChat(func(u *models.User, msg string) {
		chatCalled = true
	})

	// Verify that registering handlers does not panic and properties are set
	if bot.Gateway == nil {
		t.Fatal("expected gateway client initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Connect to non-existent endpoint should fail or timeout gracefully without panic
	_ = bot.Start(ctx)

	if readyCalled || chatCalled {
		t.Log("handlers registered successfully")
	}
}
