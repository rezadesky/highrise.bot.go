# Event System & Callbacks

The SDK features an event-driven architecture with a concurrency-safe event dispatcher and worker pool.

---

## High-Level Event Handlers

Register event listeners directly on your `bot` instance:

### Chat & Messaging
```go
// Public room chat
bot.OnChat(func(user *highrisebot.User, message string) {
    log.Printf("[%s]: %s", user.Username, message)
})

// Private whisper
bot.OnWhisper(func(user *highrisebot.User, message string) {
    log.Printf("[Whisper from %s]: %s", user.Username, message)
})

// Direct Inbox Messages
bot.OnMessage(func(userID, conversationID string, isNew bool) {
    log.Printf("Received message from user %s in conversation %s", userID, conversationID)
})
```

### Player Lifecycle & Movement
```go
// User enters room
bot.OnUserJoin(func(user *highrisebot.User, pos highrisebot.Position) {
    log.Printf("%s joined at (%.2f, %.2f, %.2f)", user.Username, pos.X, pos.Y, pos.Z)
})

// User leaves room
bot.OnUserLeave(func(user *highrisebot.User) {
    log.Printf("%s left the room", user.Username)
})

// User moves or teleports
bot.OnUserMove(func(user *highrisebot.User, pos highrisebot.Position) {
    log.Printf("%s walked to (%.2f, %.2f, %.2f)", user.Username, pos.X, pos.Y, pos.Z)
})
```

### Social & Interactions
```go
// Emotes triggered by player
bot.OnEmote(func(user *highrisebot.User, emoteID string, receiver *highrisebot.User) {
    log.Printf("%s performed emote: %s", user.Username, emoteID)
})

// Reactions (love, clap, thumbs up)
bot.OnReaction(func(user *highrisebot.User, reaction string, receiver *highrisebot.User) {
    log.Printf("%s sent reaction %s", user.Username, reaction)
})

// Gold & Item Tips
bot.OnTip(func(sender *highrisebot.User, receiver *highrisebot.User, item highrisebot.Item) {
    log.Printf("%s tipped %s with item: %s", sender.Username, receiver.Username, item.ID)
})
```

---

## Generic Events & Filtering

You can also use the generic `On()` method with optional filters:

```go
// Only handle chat messages starting with "!vip"
registration := bot.On(highrisebot.EventChat, func(ctx context.Context, e any) error {
    chat := e.(*highrisebot.ChatEvent)
    log.Printf("VIP command received: %s", chat.Message)
    return nil
}, func(e any) bool {
    chat, ok := e.(*highrisebot.ChatEvent)
    return ok && strings.HasPrefix(chat.Message, "!vip")
})

// To stop listening at runtime:
registration.Unregister()
```

---

## Event Metrics & Stats

Inspect real-time event pipeline health:

```go
stats := bot.EventStats()
log.Printf("Received: %d | Dispatched: %d | Errors: %d | Dropped: %d",
    stats.TotalReceived, stats.TotalDispatched, stats.TotalErrors, stats.TotalDropped)
```
