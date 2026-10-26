# Multibot Architecture

Need to run multiple bots in different rooms or manage a fleet of service bots? Use the built-in `MultiBotManager`.

---

## Example: Managing Multiple Bots

```go
package main

import (
	"log"
	"os"

	"github.com/rezadesky/highrisebot"
)

func main() {
	manager := highrisebot.NewMultiBotManager()

	// Bot 1: Welcome Bot in Room A
	botLobby := highrisebot.New(os.Getenv("TOKEN_1"), os.Getenv("ROOM_1"))
	botLobby.OnUserJoin(func(user *highrisebot.User, pos highrisebot.Position) {
		_ = botLobby.Chat("Welcome to the Lobby @" + user.Username)
	})

	// Bot 2: Game Master in Room B
	botGame := highrisebot.New(os.Getenv("TOKEN_2"), os.Getenv("ROOM_2"))
	botGame.OnChat(func(user *highrisebot.User, msg string) {
		if msg == "!start" {
			_ = botGame.Chat("Game starting in 5 seconds!")
		}
	})

	// Register bots into manager
	_ = manager.Register("lobby_bot", botLobby)
	_ = manager.Register("game_bot", botGame)

	// Start all bots concurrently
	log.Printf("Starting %d bots...", manager.Count())
	manager.StartAll()

	// Keep alive
	select {}
}
```

### Key Advantages:
- **Independent Failure Domains**: If one room disconnects, it reconnects automatically without affecting other bots.
- **Isolated Rate Limiters**: Each bot manages its own dedicated chat queue and WebSocket pipe.
