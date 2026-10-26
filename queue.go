package highrisebot

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MaxChatMessageLength represents the maximum character limit for Highrise chat messages.
const MaxChatMessageLength = 250

var (
	// ErrMessageTooLong is returned when chat exceeds maximum characters.
	ErrMessageTooLong = errors.New("chat: message exceeds maximum character limit")

	// ErrEmptyMessage is returned when message content is blank.
	ErrEmptyMessage = errors.New("chat: message cannot be empty")

	// ErrEmptyRecipient is returned when whisper recipient is missing.
	ErrEmptyRecipient = errors.New("chat: recipient user ID cannot be empty")
)

// QueuedChatMessage represents a queued chat item.
type QueuedChatMessage struct {
	Content string
	Whisper bool
	UserID  string
	ErrChan chan error
}

// ChatQueue manages rate-limited outgoing messages with a configurable throttle delay.
type ChatQueue struct {
	ch      chan QueuedChatMessage
	delay   time.Duration
	sender  func(m QueuedChatMessage) error
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.Mutex
}

// NewChatQueue creates a new rate-limited chat queue.
func NewChatQueue(throttleDelay time.Duration, sender func(m QueuedChatMessage) error) *ChatQueue {
	if throttleDelay <= 0 {
		throttleDelay = 500 * time.Millisecond
	}
	ctx, cancel := context.WithCancel(context.Background())
	q := &ChatQueue{
		ch:      make(chan QueuedChatMessage, 128),
		delay:   throttleDelay,
		sender:  sender,
		ctx:     ctx,
		cancel:  cancel,
		running: true,
	}
	q.wg.Add(1)
	go q.loop()
	return q
}

// Enqueue validates and pushes a message to the rate-limited queue.
func (q *ChatQueue) Enqueue(content string, whisper bool, userID string) error {
	if len(content) == 0 {
		return ErrEmptyMessage
	}
	if len([]rune(content)) > MaxChatMessageLength {
		return ErrMessageTooLong
	}
	if whisper && userID == "" {
		return ErrEmptyRecipient
	}

	errChan := make(chan error, 1)
	msg := QueuedChatMessage{
		Content: content,
		Whisper: whisper,
		UserID:  userID,
		ErrChan: errChan,
	}

	select {
	case q.ch <- msg:
		return nil
	case <-q.ctx.Done():
		return errors.New("chat: queue is stopped")
	}
}

func (q *ChatQueue) loop() {
	defer q.wg.Done()
	ticker := time.NewTicker(q.delay)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case msg, ok := <-q.ch:
			if !ok {
				return
			}
			if q.sender != nil {
				_ = q.sender(msg)
			}
			select {
			case <-ticker.C:
			case <-q.ctx.Done():
				return
			}
		}
	}
}

// Close gracefully terminates the chat queue loop.
func (q *ChatQueue) Close() {
	q.mu.Lock()
	if !q.running {
		q.mu.Unlock()
		return
	}
	q.running = false
	q.mu.Unlock()

	q.cancel()
	q.wg.Wait()
}
