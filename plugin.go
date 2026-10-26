package highrisebot

import (
	"context"
	"fmt"
	"sync"
)

// BotInterface exposes the bot features needed by plugins.
type BotInterface interface {
	Chat(msg string) error
	Emote(emoteID string, targetUserID ...string) error
}

// Plugin defines the interface for modular extensions.
type Plugin interface {
	Name() string
	Version() string
	Init(bot BotInterface) error
	Start(ctx context.Context) error
	Stop() error
}

// PluginManager registers and manages plugins.
type PluginManager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	bot     BotInterface
}

// NewPluginManager creates a plugin manager.
func NewPluginManager(bot BotInterface) *PluginManager {
	return &PluginManager{
		plugins: make(map[string]Plugin),
		bot:     bot,
	}
}

// Load initializes and starts a plugin.
func (m *PluginManager) Load(p Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := p.Name()
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already loaded", name)
	}

	if err := p.Init(m.bot); err != nil {
		return fmt.Errorf("plugin %s init failed: %w", name, err)
	}

	m.plugins[name] = p
	return nil
}

// Unload gracefully stops a plugin.
func (m *PluginManager) Unload(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	delete(m.plugins, name)
	return p.Stop()
}
