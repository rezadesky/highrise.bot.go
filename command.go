package highrisebot

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// CommandContext encapsulates data provided to a command handler execution.
type CommandContext struct {
	Ctx       context.Context
	User      User
	Message   string
	Command   string
	Args      []string
	Whisper   bool
	ReplyFunc func(msg string) error
	BotOwner  string
}

// Reply sends a response back to the user or room.
func (c *CommandContext) Reply(msg string) error {
	if c.ReplyFunc != nil {
		return c.ReplyFunc(msg)
	}
	return nil
}

// CommandHandlerFunc is the signature for command actions.
type CommandHandlerFunc func(c *CommandContext) error

// Command represents a single command definition.
type Command struct {
	Name        string
	Aliases     []string
	Description string
	Category    string
	OwnerOnly   bool
	MinArgs     int
	Cooldown    time.Duration
	Enabled     bool
	Handler     CommandHandlerFunc
	Subcommands map[string]*Command
}

// CommandRegistry manages command lookup, cooldowns, execution, and help generation.
type CommandRegistry struct {
	Prefix      string
	mu          sync.RWMutex
	commands    map[string]*Command
	aliases     map[string]string
	cooldowns   map[string]time.Time
	cooldownMu  sync.Mutex
	Middlewares []func(next CommandHandlerFunc) CommandHandlerFunc
}

// NewCommandRegistry creates a new Command Registry.
func NewCommandRegistry(prefix string) *CommandRegistry {
	if prefix == "" {
		prefix = "!"
	}
	return &CommandRegistry{
		Prefix:    prefix,
		commands:  make(map[string]*Command),
		aliases:   make(map[string]string),
		cooldowns: make(map[string]time.Time),
	}
}

// Register registers a command with optional subcommands.
func (r *CommandRegistry) Register(cmd *Command) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd.Enabled = true
	nameLower := strings.ToLower(cmd.Name)
	r.commands[nameLower] = cmd

	for _, alias := range cmd.Aliases {
		r.aliases[strings.ToLower(alias)] = nameLower
	}
}

// Execute parses a message string and invokes the matching command.
func (r *CommandRegistry) Execute(ctx context.Context, user User, message string, whisper bool, replyFn func(msg string) error, botOwner string) (bool, error) {
	if !strings.HasPrefix(message, r.Prefix) {
		return false, nil
	}

	trimmed := strings.TrimSpace(message[len(r.Prefix):])
	if trimmed == "" {
		return false, nil
	}

	parts := strings.Fields(trimmed)
	cmdName := strings.ToLower(parts[0])
	args := parts[1:]

	r.mu.RLock()
	realName, isAlias := r.aliases[cmdName]
	if isAlias {
		cmdName = realName
	}
	cmd, exists := r.commands[cmdName]
	r.mu.RUnlock()

	if !exists || !cmd.Enabled {
		return false, nil
	}

	if cmd.OwnerOnly && (botOwner == "" || user.ID != botOwner) {
		_ = replyFn("⛔ You do not have permission to execute this command.")
		return true, nil
	}

	if len(args) < cmd.MinArgs {
		_ = replyFn(fmt.Sprintf("⚠️ Usage: %s%s (requires at least %d args)", r.Prefix, cmd.Name, cmd.MinArgs))
		return true, nil
	}

	if cmd.Cooldown > 0 {
		key := fmt.Sprintf("%s:%s", user.ID, cmdName)
		r.cooldownMu.Lock()
		expire, onCooldown := r.cooldowns[key]
		if onCooldown && time.Now().Before(expire) {
			r.cooldownMu.Unlock()
			remaining := time.Until(expire).Round(time.Second)
			_ = replyFn(fmt.Sprintf("⏳ Please wait %v before using this command again.", remaining))
			return true, nil
		}
		r.cooldowns[key] = time.Now().Add(cmd.Cooldown)
		r.cooldownMu.Unlock()
	}

	cmdCtx := &CommandContext{
		Ctx:       ctx,
		User:      user,
		Message:   message,
		Command:   cmdName,
		Args:      args,
		Whisper:   whisper,
		ReplyFunc: replyFn,
		BotOwner:  botOwner,
	}

	handler := cmd.Handler
	for i := len(r.Middlewares) - 1; i >= 0; i-- {
		handler = r.Middlewares[i](handler)
	}

	err := handler(cmdCtx)
	return true, err
}

// Help generates an automatic formatted help text for all registered commands.
func (r *CommandRegistry) Help() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("📜 Available Commands:\n")
	for name, cmd := range r.commands {
		if !cmd.Enabled {
			continue
		}
		sb.WriteString(fmt.Sprintf("- %s%s: %s\n", r.Prefix, name, cmd.Description))
	}
	return sb.String()
}
