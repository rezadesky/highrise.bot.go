# Command Framework

The SDK includes a built-in Command Framework that simplifies creating chat-driven commands (like `!ping`, `!dance`, `!help`).

---

## Defining Commands

Commands are registered on `bot.Commands`:

```go
bot.Commands.Register(&highrisebot.Command{
    Name:        "ping",
    Aliases:     []string{"p"},
    Description: "Replies with Pong!",
    Cooldown:    3 * time.Second, // Per-user cooldown
    Handler: func(c *highrisebot.CommandContext) error {
        return c.Reply("Pong! 🏓")
    },
})
```

---

## Owner-Only & Minimum Arguments

Secure your sensitive commands easily:

```go
bot.Commands.Register(&highrisebot.Command{
    Name:        "teleport",
    Description: "Teleports target user to your location",
    OwnerOnly:   true,  // Only room owner can run this
    MinArgs:     1,     // Requires at least 1 argument
    Handler: func(c *highrisebot.CommandContext) error {
        targetUsername := c.Args[0]
        return c.Reply("Teleporting @" + targetUsername)
    },
})
```

---

## Automatic Help Generator

Get an automatic, formatted list of all registered commands:

```go
bot.Commands.Register(&highrisebot.Command{
    Name:        "help",
    Description: "Displays list of available commands",
    Handler: func(c *highrisebot.CommandContext) error {
        return c.Reply(bot.Commands.Help())
    },
})
```
