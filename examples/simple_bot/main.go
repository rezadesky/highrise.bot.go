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
		log.Println("Please set HR_TOKEN and HR_ROOM_ID environment variables.")
		return
	}

	bot := highrisebot.New(token, roomID)

	// Triggered when bot successfully connects to the room
	bot.OnReady(func(session *highrisebot.SessionMetadata) {
		log.Printf("Bot is online! Bot User ID: %s | Room ID: %s", session.UserID, session.RoomInfo.RoomID)
		bot.Chat("Hello everyone! Highrise Go Bot is now active 🚀")
	})

	// Triggered when a public chat message is sent
	bot.OnChat(func(user *highrisebot.User, message string) {
		log.Printf("[%s]: %s", user.Username, message)

		if strings.EqualFold(message, "!ping") {
			bot.Chat("Pong! 🏓")
		} else if strings.EqualFold(message, "!dance") {
			bot.Emote("emote-lust")
		}
	})

	// Triggered when a private whisper is sent to the bot
	bot.OnWhisper(func(user *highrisebot.User, message string) {
		log.Printf("[Whisper from %s]: %s", user.Username, message)
		bot.Whisper(user.ID, "Hello! This is a private reply from the bot 🤫")
	})

	// Triggered when a user enters the room
	bot.OnUserJoin(func(user *highrisebot.User, pos highrisebot.Position) {
		bot.Chat("Welcome @" + user.Username + " to the room!")
	})

	// Triggered when a connection or server error occurs
	bot.OnError(func(err error) {
		log.Printf("[ERROR] %v", err)
	})

	log.Println("Connecting bot to Highrise...")
	if err := bot.Start(); err != nil {
		log.Fatalf("Bot stopped: %v", err)
	}
}
