package highrisebot

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// MultiBotManager coordinates running and lifecycle of multiple bots in a single process.
type MultiBotManager struct {
	mu   sync.RWMutex
	bots map[string]*Bot
	wg   sync.WaitGroup
	ctx  context.Context
	stop context.CancelFunc
}

// NewMultiBotManager creates a multibot manager.
func NewMultiBotManager() *MultiBotManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &MultiBotManager{
		bots: make(map[string]*Bot),
		ctx:  ctx,
		stop: cancel,
	}
}

// Register registers a bot with a unique label identifier.
func (m *MultiBotManager) Register(name string, bot *Bot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.bots[name]; exists {
		return fmt.Errorf("bot %s is already registered", name)
	}
	m.bots[name] = bot
	return nil
}

// StartBot starts a specific registered bot asynchronously.
func (m *MultiBotManager) StartBot(name string) error {
	m.mu.RLock()
	bot, exists := m.bots[name]
	m.mu.RUnlock()

	if !exists {
		return errors.New("bot not found")
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		_ = bot.Start(m.ctx)
	}()
	return nil
}

// StartAll starts all registered bots concurrently.
func (m *MultiBotManager) StartAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name := range m.bots {
		_ = m.StartBot(name)
	}
}

// StopBot stops a specific bot gracefully.
func (m *MultiBotManager) StopBot(name string) {
	m.mu.RLock()
	bot, exists := m.bots[name]
	m.mu.RUnlock()

	if exists {
		bot.Stop()
	}
}

// StopAll stops all bots coordinatedly.
func (m *MultiBotManager) StopAll() {
	m.stop()
	m.mu.RLock()
	for _, b := range m.bots {
		b.Stop()
	}
	m.mu.RUnlock()
	m.wg.Wait()
}

// Count returns the number of registered bots.
func (m *MultiBotManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.bots)
}
