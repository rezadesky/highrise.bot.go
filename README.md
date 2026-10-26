<div align="center">

# highrise.bot.go 🚀

**An unofficial, blazingly fast, high-performance Highrise Virtual World Bot & Web API SDK written in Go.**

[![Go Reference](https://pkg.go.dev/badge/github.com/rezadesky/highrisebot.svg)](https://pkg.go.dev/github.com/rezadesky/highrisebot)
[![Documentation](https://img.shields.io/badge/Docs-Docsify-00add8.svg)](https://rezadesky.github.io/highrise.bot.go/)
[![Go Report Card](https://goreportcard.com/badge/github.com/rezadesky/highrise.bot.go)](https://goreportcard.com/report/github.com/rezadesky/highrise.bot.go)
[![CI Workflow](https://github.com/rezadesky/highrise.bot.go/actions/workflows/ci.yml/badge.svg)](https://github.com/rezadesky/highrise.bot.go/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

[Documentation](https://rezadesky.github.io/highrise.bot.go/) • [Quickstart](#-quickstart) • [Features](#-features) • [Examples](./examples/) • [Contributing](./CONTRIBUTING.md)

</div>

---

## ⚡ Highlights & Why Go?

Most Highrise bots are historically built using Python or Node.js. **`highrise.bot.go`** was created from the ground up to offer an enterprise-grade, memory-efficient alternative:

- 🏎️ **Extreme Resource Efficiency**: Consumes under **15 MB RAM** per running bot instance.
- 🚀 **High Concurrency**: Handles thousands of real-time room events effortlessly via lightweight Go goroutines.
- 📦 **Flat Idiomatic Go Design**: No convoluted package pollution—import `highrisebot` once and access everything.
- 🛡️ **Enterprise Resilience**: Auto-reconnect with randomized exponential jitter backoff, bounded channels, and zero-panic crash guards.
- 🌐 **100% Protocol Parity**: Real-time WebSocket room gateway, complete REST Web API client, inventory, wallet, and voice channels.

---

## 📦 Installation

Requires **Go 1.21** or later.

```bash
go get github.com/rezadesky/highrise.bot.go
```

---

## 🚀 Quickstart

Create a `main.go` file:

```go
package main

import (
	"log"
	"os"

	"github.com/rezadesky/highrisebot"
)

func main() {
	// Obtain credentials from https://create.highrise.game
	token := os.Getenv("HR_TOKEN")
	roomID := os.Getenv("HR_ROOM_ID")

	if token == "" || roomID == "" {
		log.Fatal("HR_TOKEN and HR_ROOM_ID must be set!")
	}

	// 1. Initialize client with functional options
	bot := highrisebot.New(token, roomID)

	// 2. Lifecycle: OnReady event
	bot.OnReady(func(session *highrisebot.SessionMetadata) {
		log.Printf("Bot online in room '%s' (Bot ID: %s)", session.RoomInfo.RoomName, session.UserID)
		_ = bot.Chat("Hello Highrise! Go Bot is running 🚀")
	})

	// 3. Room Interactions: OnChat event
	bot.OnChat(func(user *highrisebot.User, message string) {
		if message == "!ping" {
			_ = bot.Chat("Pong! 🏓")
		}
	})

	// 4. Welcome new players
	bot.OnUserJoin(func(user *highrisebot.User, pos highrisebot.Position) {
		_ = bot.Chat("Welcome @" + user.Username + " to the room! 🎉")
	})

	// 5. Run with automatic graceful shutdown on SIGINT / SIGTERM
	log.Println("Connecting bot to Highrise...")
	if err := bot.Run(); err != nil {
		log.Fatalf("Bot stopped: %v", err)
	}
}
```

Run your bot:

```bash
export HR_TOKEN="your_bot_api_token"
export HR_ROOM_ID="your_room_id"
go run main.go
```

---

## 🎮 Built-in Command Framework

Create interactive chat commands with per-user cooldowns, aliases, and permission controls:

```go
// Register a !dance command with 5-second cooldown
bot.Commands.Register(&highrisebot.Command{
    Name:        "dance",
    Aliases:     []string{"d"},
    Description: "Performs an emote animation",
    Cooldown:    5 * time.Second,
    Handler: func(c *highrisebot.CommandContext) error {
        _ = bot.Emote("emote-lust")
        return c.Reply("Dancing! 💃")
    },
})

// Owner-only moderation command
bot.Commands.Register(&highrisebot.Command{
    Name:        "kick",
    OwnerOnly:   true,
    MinArgs:     1,
    Handler: func(c *highrisebot.CommandContext) error {
        targetUserID := c.Args[0]
        return bot.Kick(targetUserID)
    },
})

// Auto-generated help command
bot.Commands.Register(&highrisebot.Command{
    Name: "help",
    Handler: func(c *highrisebot.CommandContext) error {
        return c.Reply(bot.Commands.Help())
    },
})
```

---

## 🛠️ API Reference Summary

### Event Callbacks
| Event | Signature | Description |
| :--- | :--- | :--- |
| `OnReady` | `func(*SessionMetadata)` | Triggered upon successful room connection & auth handshake. |
| `OnChat` | `func(*User, string)` | Triggered when a public chat message is sent. |
| `OnWhisper` | `func(*User, string)` | Triggered when a private whisper is sent to the bot. |
| `OnUserJoin` | `func(*User, Position)` | Triggered when a player enters the room. |
| `OnUserLeave` | `func(*User)` | Triggered when a player leaves the room. |
| `OnUserMove` | `func(*User, Position)` | Triggered when a player walks or teleports. |
| `OnEmote` | `func(*User, string, *User)` | Triggered when a player performs an emote animation. |
| `OnReaction` | `func(*User, string, *User)` | Triggered when a reaction emoji (heart, clap, etc.) is sent. |
| `OnTip` | `func(*User, *User, Item)` | Triggered when a tip is given. |
| `OnMessage` | `func(userID, convoID, isNew)` | Triggered when a direct inbox message arrives. |
| `OnDisconnect`| `func(error)` | Triggered if the socket connection drops. |

### Room Actions
| Method | Description |
| :--- | :--- |
| `bot.Chat(msg)` | Send message to room (rate-limited, max 250 chars). |
| `bot.Whisper(userID, msg)` | Send private whisper to user. |
| `bot.Walk(x, y, z, facing...)` | Move bot avatar coordinates. |
| `bot.Sit(entityID, anchorIx, facing...)` | Anchor avatar to a chair or seat. |
| `bot.Teleport(userID, x, y, z, facing...)` | Teleport target user or bot. |
| `bot.Emote(emoteID, targetUserID...)` | Trigger an emote animation. |
| `bot.React(reaction, targetUserID)` | Send emoji reaction. |
| `bot.Kick(userID)` | Kick user from room. |
| `bot.Mute(userID, seconds)` | Mute user for duration. |
| `bot.Ban(userID, seconds)` | Ban user from room. |
| `bot.Unban(userID)` | Lift room ban from user. |

### Real-Time Queries (RPC)
| Method | Description |
| :--- | :--- |
| `bot.GetRoomUsers()` | Retrieve all players in room with their live coordinates. |
| `bot.GetWallet()` | Inspect bot currency balances (Gold, Boost & Voice tokens). |
| `bot.GetInventory()` | Inspect clothing backpack items. |
| `bot.SetOutfit(items)` | Update bot avatar clothing items. |
| `bot.TipUser(userID, barAmount)` | Tip gold bars (`"gold_bar_1"`, `"gold_bar_5"`, `"gold_bar_10"`). |
| `bot.GetVoiceStatus()` | Inspect room voice channel status & remaining seconds. |
| `bot.AddUserToVoice(userID)` | Invite user to speak in voice chat. |
| `bot.GetConversations(notJoined, lastID)` | Retrieve inbox direct conversation threads. |

---

## 🤖 Multibot Fleet Management

Run and coordinate multiple bots in separate rooms from a single process:

```go
manager := highrisebot.NewMultiBotManager()

bot1 := highrisebot.New(token1, room1)
bot2 := highrisebot.New(token2, room2)

_ = manager.Register("lobby_bot", bot1)
_ = manager.Register("games_bot", bot2)

// Starts all bots concurrently with isolated failure domains
manager.StartAll()
```

---

## 🌐 Documentation & Community

Detailed documentation, guides, and tutorials are available on the official documentation site:
👉 **[https://rezadesky.github.io/highrise.bot.go/](https://rezadesky.github.io/highrise.bot.go/)**

---

## ⚠️ Disclaimer
This SDK is an **unofficial**, open-source community library. It is not affiliated with, endorsed, or sponsored by Pocket Worlds or Highrise Virtual Reality.

## 📄 License
Released under the [MIT License](LICENSE).
