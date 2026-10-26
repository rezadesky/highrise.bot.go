# Troubleshooting & FAQ

## Frequently Asked Questions

### 1. Where do I get my Bot API Token and Room ID?
- **Bot API Token**: Generated in the [Highrise Create Portal](https://create.highrise.game) under **Bots & API Keys**.
- **Room ID**: Copy the ID from your room URL. Make sure your bot has Owner, Designer, or Moderator permissions in the target room.

### 2. Can I run multiple bots at the same time?
Yes! Use the built-in `multibot.Manager`:
```go
mgr := multibot.NewManager()
mgr.Register("bot_lobby", bot1)
mgr.Register("bot_games", bot2)
mgr.StartAll()
```

### 3. Does it handle socket disconnects automatically?
Yes. The SDK monitors ping/pong frames and automatically reconnects using exponential backoff with full random jitter.

---

## Common Errors & Solutions

| Error Message | Cause | Solution |
| :--- | :--- | :--- |
| `Room not found.` | Room ID is incorrect or bot is not authorized. | Ensure the room ID matches the room where your bot has designer or moderator rights. |
| `protocol: authentication rejected` | Invalid or revoked API Token. | Regenerate a new API Token from [create.highrise.game](https://create.highrise.game). |
| `highrisebot: api token is required` | Passed empty token string. | Set the `HR_TOKEN` environment variable before booting the bot. |
| `chat: message exceeds maximum character limit` | Chat exceeds 250 characters. | Shorten message length or split long text into multiple chunks. |
| `gateway write buffer full` | Outgoing message buffer saturated. | Lower message broadcast frequency or use the built-in rate-limited chat queue. |
