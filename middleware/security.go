package middleware

import (
	"log"
	"sync"
	"time"

	"github.com/rezadesky/highrisebot/models"
)

// SecurityManager manages user allowlist, denylist, and rate limits.
type SecurityManager struct {
	mu        sync.RWMutex
	allowlist map[string]bool
	denylist  map[string]bool
	auditLogs []string
}

// NewSecurityManager creates a new security manager.
func NewSecurityManager() *SecurityManager {
	return &SecurityManager{
		allowlist: make(map[string]bool),
		denylist:  make(map[string]bool),
		auditLogs: make([]string, 0),
	}
}

// Deny adds a user to the blacklist.
func (sm *SecurityManager) Deny(userID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.denylist[userID] = true
}

// Allow removes a user from blacklist and adds to allowlist.
func (sm *SecurityManager) Allow(userID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.denylist, userID)
	sm.allowlist[userID] = true
}

// IsDenied checks if a user is blacklisted.
func (sm *SecurityManager) IsDenied(userID string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.denylist[userID]
}

// Audit logs critical actions taken by the bot.
func (sm *SecurityManager) Audit(action string, user models.User, details string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	entry := time.Now().Format(time.RFC3339) + " | " + action + " | User: " + user.Username + "(" + user.ID + ") | " + details
	sm.auditLogs = append(sm.auditLogs, entry)
	log.Printf("[AUDIT] %s", entry)
}
