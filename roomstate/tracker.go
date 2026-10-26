package roomstate

import (
	"sync"

	"github.com/rezadesky/highrisebot/models"
)

// Tracker tracks the live state of all users and their positions in the room.
type Tracker struct {
	mu    sync.RWMutex
	users map[string]models.UserPosition
}

// NewTracker creates a new room state tracker.
func NewTracker() *Tracker {
	return &Tracker{
		users: make(map[string]models.UserPosition),
	}
}

// SetUsers bulk populates the active users list (e.g. from GetRoomUsers).
func (t *Tracker) SetUsers(userPositions []models.UserPosition) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.users = make(map[string]models.UserPosition, len(userPositions))
	for _, up := range userPositions {
		t.users[up.User.ID] = up
	}
}

// OnUserJoined handles user join state tracking.
func (t *Tracker) OnUserJoined(u models.User, pos models.Position) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.users[u.ID] = models.UserPosition{
		User:     u,
		Position: pos,
	}
}

// OnUserLeft handles user leave state tracking.
func (t *Tracker) OnUserLeft(userID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.users, userID)
}

// OnUserMoved updates tracked position for a user.
func (t *Tracker) OnUserMoved(userID string, pos models.Position) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if up, exists := t.users[userID]; exists {
		up.Position = pos
		t.users[userID] = up
	}
}

// GetUser retrieves a user by ID from tracked state.
func (t *Tracker) GetUser(userID string) (models.UserPosition, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	up, exists := t.users[userID]
	return up, exists
}

// GetAllUsers returns a slice of all tracked users currently in the room.
func (t *Tracker) GetAllUsers() []models.UserPosition {
	t.mu.RLock()
	defer t.mu.RUnlock()
	res := make([]models.UserPosition, 0, len(t.users))
	for _, up := range t.users {
		res = append(res, up)
	}
	return res
}

// Count returns the current active user count in the room.
func (t *Tracker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.users)
}
