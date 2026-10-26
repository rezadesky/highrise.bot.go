# highrisebot 🚀

[![Go Report Card](https://goreportcard.com/badge/github.com/rezadesky/highrisebot)](https://goreportcard.com/report/github.com/rezadesky/highrisebot)
[![Go Reference](https://pkg.go.dev/badge/github.com/rezadesky/highrisebot.svg)](https://pkg.go.dev/github.com/rezadesky/highrisebot)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

An unofficial, high-performance **Highrise Bot & Web API SDK** written in Go. Designed to be lightweight, concurrency-safe, with built-in automatic reconnects and an intuitive event-driven API (similar to DiscordGo).

---

## ✨ Features

- ⚡ **Event-Driven Architecture**: Intuitive callback listeners (`OnChat`, `OnUserJoin`, `OnEmote`, etc.).
- 🔄 **Built-in Auto Reconnect**: Seamlessly handles socket drops and reconnections with exponential backoff.
- 💬 **Complete Room Controls**: Chat, Whisper, Teleport, Walk, Emotes, Reactions, and Moderation (Kick/Mute/Ban).
- 🌐 **Integrated Web API Client**: Native support for Highrise REST Web API queries (Users, Rooms, Items).
- 🛡️ **Zero Bloat & Concurrency-Safe**: Backed by Gorilla WebSocket with thread-safe write channels.

---

## 📦 Installation

```bash
go get github.com/rezadesky/highrisebot
```

---

## 🚀 Quickstart

```go
package main

import (
	"log"
	"os"

	"github.com/rezadesky/highrisebot"
)

func main() {
	// Obtain your token and room ID from https://create.highrise.game
	token := os.Getenv("HR_TOKEN")
	roomID := os.Getenv("HR_ROOM_ID")

	bot := highrisebot.New(token, roomID)

	// Callback when bot successfully connects
	bot.OnReady(func(session *highrisebot.SessionMetadata) {
		log.Printf("Bot online! Bot User ID: %s", session.UserID)
		bot.Chat("Hello Highrise! Go Bot is running 🚀")
	})

	// Callback for chat messages
	bot.OnChat(func(user *highrisebot.User, message string) {
		if message == "!ping" {
			bot.Chat("Pong! 🏓")
		}
	})

	// Callback when a user joins the room
	bot.OnUserJoin(func(user *highrisebot.User, pos highrisebot.Position) {
		bot.Chat("Welcome @" + user.Username + "!")
	})

	// Start bot connection (blocking)
	if err := bot.Start(); err != nil {
		log.Fatal(err)
	}
}
```

---

## 🛠️ API Reference

### Event Listeners
- `bot.OnReady(func(session *highrisebot.SessionMetadata))`
- `bot.OnChat(func(user *highrisebot.User, message string))`
- `bot.OnWhisper(func(user *highrisebot.User, message string))`
- `bot.OnUserJoin(func(user *highrisebot.User, pos highrisebot.Position))`
- `bot.OnUserLeave(func(user *highrisebot.User))`
- `bot.OnUserMove(func(user *highrisebot.User, pos highrisebot.Position))`
- `bot.OnEmote(func(user *highrisebot.User, emoteID string, receiver *highrisebot.User))`
- `bot.OnReaction(func(user *highrisebot.User, reaction string, receiver *highrisebot.User))`
- `bot.OnTip(func(sender *highrisebot.User, receiver *highrisebot.User, item highrisebot.Item))`
- `bot.OnError(func(err error))`

### Room Actions
- `bot.Chat(message string)`
- `bot.Whisper(userID, message string)`
- `bot.Walk(x, y, z float64, facing ...string)`
- `bot.Teleport(userID string, x, y, z float64, facing ...string)`
- `bot.Emote(emoteID string, targetUserID ...string)`
- `bot.React(reaction, targetUserID string)`
- `bot.Kick(userID string)`
- `bot.Mute(userID string, durationSeconds int)`
- `bot.Ban(userID string, durationSeconds int)`

### Web API Client (Public REST Data)
```go
ctx := context.Background()

// Fetch user profile
userProfile, err := bot.WebAPI.GetUser(ctx, "USER_ID_HERE")

// Search users by username
users, err := bot.WebAPI.SearchUsers(ctx, "someone", nil)

// List public rooms
rooms, err := bot.WebAPI.GetRooms(ctx, &webapi.PaginationOptions{Limit: 10})
```

---

## 📄 License
Released under the [MIT License](LICENSE).
