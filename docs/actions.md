# Room Actions & Controls

The SDK provides direct, type-safe methods to interact with players, moderate rooms, query inventory, and manage voice channels.

---

## Chat & Communication

All outgoing chat messages pass through an internal **rate-limited queue** with character limits (250 chars) to prevent spam-kicking from Highrise servers:

```go
// Send public room message
err := bot.Chat("Hello, Highrise!")

// Send private whisper
err := bot.Whisper(targetUserID, "This is a secret whisper.")
```

---

## Movement & Avatar Control

Control bot navigation and positions:

```go
// Walk to coordinates
err := bot.Walk(10.5, 0.0, 5.0, "FrontRight")

// Sit on a chair/sofa entity
err := bot.Sit("chair_entity_id", 0, "FrontLeft")

// Teleport a user or bot itself to specific coordinates
err := bot.Teleport(userID, 15.0, 0.0, 15.0, "FrontRight")

// Trigger an emote
err := bot.Emote("emote-lust")

// Send emoji reaction to a user
err := bot.React("heart", targetUserID)
```

---

## Room Moderation

Keep your room safe with moderator tools:

```go
// Kick user from room
err := bot.Kick(targetUserID)

// Mute user for 60 seconds
err := bot.Mute(targetUserID, 60)

// Ban user for 3600 seconds (1 hour)
err := bot.Ban(targetUserID, 3600)

// Remove ban from user
err := bot.Unban(targetUserID)
```

---

## Queries & Economy (RPC)

Query live room state synchronously:

```go
// Fetch active room users and coordinates
users, err := bot.GetRoomUsers()
for _, u := range users {
    log.Printf("- %s at (%.2f, %.2f, %.2f)", u.User.Username, u.Position.X, u.Position.Y, u.Position.Z)
}

// Check bot wallet balances
wallet, err := bot.GetWallet()
for _, c := range wallet {
    log.Printf("- Currency: %s | Amount: %d", c.Type, c.Amount)
}

// Fetch bot backpack inventory
inventory, err := bot.GetInventory()

// Change bot outfit
err = bot.SetOutfit(inventory)

// Tip gold bars to a user
// Valid bars: "gold_bar_1", "gold_bar_5", "gold_bar_10", "gold_bar_50", "gold_bar_100", "gold_bar_500", "gold_bar_1k"
err = bot.TipUser(targetUserID, "gold_bar_5")

// Buy item from shop
err = bot.BuyItem("item_clothing_id")
```

---

## Voice Chat Management

```go
// Check room voice channel status and remaining time
status, err := bot.GetVoiceStatus()
log.Printf("Voice seconds left: %.2f", status.SecondsLeft)

// Invite/add user to voice chat
err = bot.AddUserToVoice(targetUserID)

// Remove user from voice chat
err = bot.RemoveUserFromVoice(targetUserID)

// Purchase voice time for room
err = bot.BuyVoiceTime("bot_wallet_only")
```
