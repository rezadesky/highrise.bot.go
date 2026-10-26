package highrisebot_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	highrisebot "github.com/rezadesky/highrise.bot.go"
)

func TestCoreClientCreationAndOptions(t *testing.T) {
	invalidBot := highrisebot.New("", "")
	err := invalidBot.Connect()
	if !errors.Is(err, highrisebot.ErrMissingToken) {
		t.Fatalf("expected ErrMissingToken, got %v", err)
	}

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
	if bot.Status() != highrisebot.StatusDisconnected {
		t.Errorf("expected initial status DISCONNECTED, got %s", bot.Status())
	}
}

func TestChatQueueValidation(t *testing.T) {
	var count uint32
	q := highrisebot.NewChatQueue(50*time.Millisecond, func(m highrisebot.QueuedChatMessage) error {
		atomic.AddUint32(&count, 1)
		return nil
	})
	defer q.Close()

	if err := q.Enqueue("", false, ""); !errors.Is(err, highrisebot.ErrEmptyMessage) {
		t.Errorf("expected ErrEmptyMessage, got %v", err)
	}

	longMsg := strings.Repeat("x", 251)
	if err := q.Enqueue(longMsg, false, ""); !errors.Is(err, highrisebot.ErrMessageTooLong) {
		t.Errorf("expected ErrMessageTooLong, got %v", err)
	}
}

func TestCommandRegistry(t *testing.T) {
	reg := highrisebot.NewCommandRegistry("!")
	reg.Register(&highrisebot.Command{
		Name:    "ping",
		Aliases: []string{"p"},
		Handler: func(c *highrisebot.CommandContext) error {
			return c.Reply("Pong!")
		},
	})

	var replied string
	replyFn := func(msg string) error {
		replied = msg
		return nil
	}

	user := highrisebot.User{ID: "u1", Username: "john"}
	handled, err := reg.Execute(context.Background(), user, "!ping", false, replyFn, "owner_1")
	if !handled || err != nil || replied != "Pong!" {
		t.Fatalf("expected ping handled with Pong!, got handled=%v, reply=%s", handled, replied)
	}
}

func TestProtocolEncoding(t *testing.T) {
	enc := &highrisebot.ProtocolEncoder{}
	dec := &highrisebot.ProtocolDecoder{}

	data, err := enc.Encode(map[string]any{"_type": "TestEvent"})
	if err != nil {
		t.Fatal(err)
	}

	env, err := dec.DecodeEnvelope(data)
	if err != nil || env.Type != "TestEvent" {
		t.Fatalf("expected TestEvent envelope, got %v", env)
	}
}
