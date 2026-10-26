package command_test

import (
	"context"
	"testing"
	"time"

	"github.com/rezadesky/highrisebot/command"
	"github.com/rezadesky/highrisebot/models"
)

func TestCommandRegistrationAndExecution(t *testing.T) {
	reg := command.NewRegistry("!")

	var replyReceived string
	replyMock := func(msg string) error {
		replyReceived = msg
		return nil
	}

	// Register Ping Command
	reg.Register(&command.Command{
		Name:        "ping",
		Aliases:     []string{"p"},
		Description: "Replies with Pong",
		Handler: func(c *command.Context) error {
			return c.Reply("Pong!")
		},
	})

	user := models.User{ID: "u_1", Username: "alice"}

	// 1. Direct Command
	handled, err := reg.Execute(context.Background(), user, "!ping", false, replyMock, "owner_123")
	if !handled || err != nil || replyReceived != "Pong!" {
		t.Fatalf("expected ping reply, got handled=%v, err=%v, reply=%s", handled, err, replyReceived)
	}

	// 2. Alias Command
	replyReceived = ""
	handledAlias, _ := reg.Execute(context.Background(), user, "!p", false, replyMock, "owner_123")
	if !handledAlias || replyReceived != "Pong!" {
		t.Fatalf("expected alias reply Pong!, got %s", replyReceived)
	}

	// 3. Cooldown test
	reg.Register(&command.Command{
		Name:     "slow",
		Cooldown: 2 * time.Second,
		Handler: func(c *command.Context) error {
			return c.Reply("slow response")
		},
	})

	_, _ = reg.Execute(context.Background(), user, "!slow", false, replyMock, "owner_123")
	replyReceived = ""
	_, _ = reg.Execute(context.Background(), user, "!slow", false, replyMock, "owner_123")
	if replyReceived == "slow response" {
		t.Error("expected command to be blocked by cooldown")
	}
}
