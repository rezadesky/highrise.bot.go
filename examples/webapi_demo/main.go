package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rezadesky/highrisebot/webapi"
)

func main() {
	client := webapi.NewClient("")
	ctx := context.Background()

	// 1. Fetch public rooms
	fmt.Println("Fetching rooms...")
	rooms, err := client.GetRooms(ctx, &webapi.PaginationOptions{Limit: 5})
	if err != nil {
		log.Fatalf("Failed to fetch rooms: %v", err)
	}

	for _, room := range rooms.Rooms {
		fmt.Printf("- Room ID: %s | Room Name: %s\n", room.RoomID, room.RoomName)
	}

	// 2. Search users
	fmt.Println("\nSearching users...")
	users, err := client.SearchUsers(ctx, "test", &webapi.PaginationOptions{Limit: 3})
	if err != nil {
		log.Fatalf("Failed to search users: %v", err)
	}

	for _, user := range users.Users {
		fmt.Printf("- User ID: %s | Username: %s\n", user.ID, user.Username)
	}
}
