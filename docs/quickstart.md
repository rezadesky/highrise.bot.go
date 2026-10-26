# Installation & Quickstart

## Prerequisites
- **Go 1.21+** installed on your machine ([Download Go](https://go.dev/dl/)).
- A **Highrise Bot Account & API Token** obtained from [Highrise Create](https://create.highrise.game).
- A **Room ID** where your bot account has Designer or Moderator permissions.

---

## Installation

Add `highrisebot` to your Go module using:

```bash
go get github.com/rezadesky/highrisebot
```

---

## Creating Your First Bot

Create a new file named `main.go`:

```go
package main

import (
	"log"
	"os"
	"strings"

	"github.com/rezadesky/highrisebot"
)

func main() {
	token := os.Getenv("HR_TOKEN")
	roomID := os.Getenv("HR_ROOM_ID")

	if token == "" || roomID == "" {
		log.Fatal("HR_TOKEN and HR_ROOM_ID environment variables must be set!")
	}

	// 1. Initialize Bot instance
	bot := highrisebot.New(token, roomID)

	// 2. Register Lifecycle hooks
	bot.OnReady(func(session *highrisebot.SessionMetadata) {
		log.Printf("Logged in as @%s (Bot ID: %s)", session.RoomInfo.RoomName, session.UserID)
		_ = bot.Chat("Hello Highrise! I am online.")
	})

	// 3. Listen to Chat events
	bot.OnChat(func(user *highrisebot.User, message string) {
		log.Printf("[%s]: %s", user.Username, message)

		if strings.EqualFold(message, "!hello") {
			_ = bot.Chat("Hello @" + user.Username + "! 👋")
		}
	})

	// 4. Welcome new users
	bot.OnUserJoin(func(user *highrisebot.User, pos highrisebot.Position) {
		_ = bot.Chat("Welcome @" + user.Username + " to the room! 🎉")
	})

	// 5. Start bot (blocks and automatically catches OS interrupt signals for graceful stop)
	log.Println("Starting bot...")
	if err := bot.Run(); err != nil {
		log.Fatalf("Bot stopped: %v", err)
	}
}
```

---

## Running the Bot

Export your credentials and run the program:

### On Linux / macOS / Bash:
```bash
export HR_TOKEN="your_api_token_here"
export HR_ROOM_ID="your_room_id_here"
go run main.go
```

### On Windows PowerShell:
```powershell
$env:HR_TOKEN="your_api_token_here"
$env:HR_ROOM_ID="your_room_id_here"
go run main.go
```
