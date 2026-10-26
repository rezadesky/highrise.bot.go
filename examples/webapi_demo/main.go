package main

import (
	"context"
	"fmt"
	"log"

	highrisebot "github.com/rezadesky/highrise.bot.go"
)

func main() {
	client := highrisebot.NewWebAPIClient("")
	ctx := context.Background()

	// 1. Fetch public rooms
	fmt.Println("Fetching rooms...")
	rooms, err := client.GetRooms(ctx, &highrisebot.PaginationOptions{Limit: 5})
	if err != nil {
		log.Fatalf("Failed to fetch rooms: %v", err)
	}

	for _, room := range rooms.Rooms {
		fmt.Printf("- Room ID: %s | Room Name: %s\n", room.RoomID, room.RoomName)
	}

	// 2. Search users
	fmt.Println("\nSearching users...")
	users, err := client.SearchUsers(ctx, "test", &highrisebot.PaginationOptions{Limit: 3})
	if err != nil {
		log.Fatalf("Failed to search users: %v", err)
	}

	for _, user := range users.Users {
		fmt.Printf("- User ID: %s | Username: %s\n", user.ID, user.Username)
	}
}
