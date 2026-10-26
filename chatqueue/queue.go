package chatqueue

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

// Message represents a queued chat item.
type Message struct {
	Content string
	Whisper bool
	UserID  string
	ErrChan chan error
}

// Queue manages rate-limited outgoing messages with a configurable throttle delay.
type Queue struct {
	ch      chan Message
	delay   time.Duration
	sender  func(m Message) error
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.Mutex
}

// NewQueue creates a new rate-limited chat queue.
func NewQueue(throttleDelay time.Duration, sender func(m Message) error) *Queue {
	if throttleDelay <= 0 {
		throttleDelay = 500 * time.Millisecond // default throttle to avoid server rate-limits
	}
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{
		ch:      make(chan Message, 128),
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
func (q *Queue) Enqueue(content string, whisper bool, userID string) error {
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
	msg := Message{
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

func (q *Queue) loop() {
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
			// Enforce delay between messages to prevent spam kicking
			select {
			case <-ticker.C:
			case <-q.ctx.Done():
				return
			}
		}
	}
}

// Close gracefully terminates the chat queue loop.
func (q *Queue) Close() {
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
