package highrisebot

import (
	"sync"
)

// RoomTracker tracks the live state of all users and their positions in the room.
type RoomTracker struct {
	mu    sync.RWMutex
	users map[string]UserPosition
}

// NewRoomTracker creates a new room state tracker.
func NewRoomTracker() *RoomTracker {
	return &RoomTracker{
		users: make(map[string]UserPosition),
	}
}

// SetUsers bulk populates the active users list (e.g. from GetRoomUsers).
func (t *RoomTracker) SetUsers(userPositions []UserPosition) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.users = make(map[string]UserPosition, len(userPositions))
	for _, up := range userPositions {
		t.users[up.User.ID] = up
	}
}

// OnUserJoined handles user join state tracking.
func (t *RoomTracker) OnUserJoined(u User, pos Position) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.users[u.ID] = UserPosition{
		User:     u,
		Position: pos,
	}
}

// OnUserLeft handles user leave state tracking.
func (t *RoomTracker) OnUserLeft(userID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.users, userID)
}

// OnUserMoved updates tracked position for a user.
func (t *RoomTracker) OnUserMoved(userID string, pos Position) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if up, exists := t.users[userID]; exists {
		up.Position = pos
		t.users[userID] = up
	}
}

// GetUser retrieves a user by ID from tracked state.
func (t *RoomTracker) GetUser(userID string) (UserPosition, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	up, exists := t.users[userID]
	return up, exists
}

// GetAllUsers returns a slice of all tracked users currently in the room.
func (t *RoomTracker) GetAllUsers() []UserPosition {
	t.mu.RLock()
	defer t.mu.RUnlock()
	res := make([]UserPosition, 0, len(t.users))
	for _, up := range t.users {
		res = append(res, up)
	}
	return res
}

// Count returns the current active user count in the room.
func (t *RoomTracker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.users)
}
