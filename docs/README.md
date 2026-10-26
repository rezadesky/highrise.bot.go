# highrise.bot.go 🚀

<div class="badge-bar">
  <a href="https://pkg.go.dev/github.com/rezadesky/highrisebot"><img src="https://pkg.go.dev/badge/github.com/rezadesky/highrisebot.svg" alt="Go Reference"></a>
  <a href="https://github.com/rezadesky/highrise.bot.go/actions"><img src="https://github.com/rezadesky/highrise.bot.go/workflows/CI/badge.svg" alt="CI Status"></a>
  <a href="https://goreportcard.com/report/github.com/rezadesky/highrise.bot.go"><img src="https://goreportcard.com/badge/github.com/rezadesky/highrise.bot.go" alt="Go Report Card"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="MIT License"></a>
</div>

**highrise.bot.go** is an unofficial, production-ready Golang SDK built specifically for creating, managing, and scaling high-performance bots in the **Highrise Virtual Reality** metaverse.

---

## 💡 Why Go for Highrise Bots?

Compared to traditional Python or Node.js bot scripts, **Go** provides major advantages for Highrise bots:
* 🏎️ **Ultra Low Memory & Latency**: Consumes less than 15MB RAM per bot instance.
* ⚡ **High Concurrency**: Thousands of concurrent room events handled effortlessly via lightweight goroutines.
* 🛡️ **Zero Runtime Crashes**: Strong type safety, panic recovery middleware, and built-in channel backpressure.
* 📦 **Single Flat Package**: No dependency hell or convoluted imports—everything lives cleanly under `highrisebot.*`.

---

## 🌟 Key Capabilities

| Feature | Description |
| :--- | :--- |
| **Real-time Gateway** | Full WebSocket communication over TLS with automatic ping/pong keep-alive. |
| **Resilient Reconnecting** | Smart exponential backoff with full randomized jitter to prevent server overload. |
| **Comprehensive Room Actions** | Chat, Whisper, Emote, Reaction, Walk, Sit, Teleport, Kick, Ban, and Unban. |
| **Economy & Inventory** | Inspect wallet (Gold, Tokens), buy shop items, fetch backpack inventory, and update outfit styles. |
| **Voice Chat Integration** | Inspect room voice statuses, invite users to speak, and purchase voice time. |
| **Command Framework** | Built-in prefix routing (`!command`), aliases, per-user cooldowns, and automatic help text generator. |
| **Direct Messages (Inbox)** | Fetch conversation threads, read inbox DMs, and reply via private messages. |
| **Multibot Architecture** | Coordinate and supervise dozens of separate bots in a single application process. |

---

## ⚡ Quick Example

```go
package main

import (
	"log"
	"os"

	"github.com/rezadesky/highrisebot"
)

func main() {
	bot := highrisebot.New(os.Getenv("HR_TOKEN"), os.Getenv("HR_ROOM_ID"))

	// Callback when bot is successfully logged in
	bot.OnReady(func(session *highrisebot.SessionMetadata) {
		log.Printf("Bot online in room: %s", session.RoomInfo.RoomName)
		bot.Chat("Hello Highrise! Go Bot is running 🚀")
	})

	// Callback for chat messages
	bot.OnChat(func(user *highrisebot.User, message string) {
		if message == "!ping" {
			bot.Chat("Pong! 🏓")
		}
	})

	// Run with graceful shutdown
	log.Fatal(bot.Run())
}
```

---

## 📖 Explore the Docs
- 🚀 [Installation & Quickstart Guide](quickstart.md)
- ⚙️ [Configuration Options](configuration.md)
- 📡 [Event System & Callbacks](events.md)
- 🎮 [Room Actions & Controls](actions.md)
- 🤖 [Multibot Manager](multibot.md)
