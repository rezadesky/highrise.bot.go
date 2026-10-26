package webapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rezadesky/highrisebot/webapi"
)

func TestGetUser(t *testing.T) {
	mockResponse := `{
		"user": {
			"id": "123456",
			"username": "rezadesky"
		},
		"bio": "Highrise bot developer",
		"num_followers": 42
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/123456" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := webapi.NewClient(server.URL)
	profile, err := client.GetUser(context.Background(), "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.User.Username != "rezadesky" {
		t.Errorf("expected username 'rezadesky', got '%s'", profile.User.Username)
	}
	if profile.NumFollowers != 42 {
		t.Errorf("expected 42 followers, got %d", profile.NumFollowers)
	}
}

func TestGetRooms(t *testing.T) {
	mockResponse := `{
		"rooms": [
			{"room_id": "room_1", "room_name": "Lounge"},
			{"room_id": "room_2", "room_name": "Dance Club"}
		],
		"total": 2
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := webapi.NewClient(server.URL)
	rooms, err := client.GetRooms(context.Background(), &webapi.PaginationOptions{Limit: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rooms.Rooms) != 2 {
		t.Errorf("expected 2 rooms, got %d", len(rooms.Rooms))
	}
	if rooms.Rooms[0].RoomName != "Lounge" {
		t.Errorf("expected room name 'Lounge', got '%s'", rooms.Rooms[0].RoomName)
	}
}
