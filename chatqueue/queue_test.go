package chatqueue_test

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rezadesky/highrisebot/chatqueue"
)

func TestChatQueueValidationAndThrottling(t *testing.T) {
	var sentCount uint32
	q := chatqueue.NewQueue(50*time.Millisecond, func(m chatqueue.Message) error {
		atomic.AddUint32(&sentCount, 1)
		return nil
	})
	defer q.Close()

	// 1. Empty message error
	errEmpty := q.Enqueue("", false, "")
	if !errors.Is(errEmpty, chatqueue.ErrEmptyMessage) {
		t.Errorf("expected ErrEmptyMessage, got %v", errEmpty)
	}

	// 2. Overlength message error
	longMsg := strings.Repeat("a", 251)
	errLong := q.Enqueue(longMsg, false, "")
	if !errors.Is(errLong, chatqueue.ErrMessageTooLong) {
		t.Errorf("expected ErrMessageTooLong, got %v", errLong)
	}

	// 3. Missing recipient on whisper
	errWhisper := q.Enqueue("secret", true, "")
	if !errors.Is(errWhisper, chatqueue.ErrEmptyRecipient) {
		t.Errorf("expected ErrEmptyRecipient, got %v", errWhisper)
	}

	// 4. Valid Enqueue with throttling
	start := time.Now()
	_ = q.Enqueue("msg 1", false, "")
	_ = q.Enqueue("msg 2", false, "")

	time.Sleep(150 * time.Millisecond)
	if atomic.LoadUint32(&sentCount) != 2 {
		t.Errorf("expected 2 sent messages, got %d", sentCount)
	}
	if time.Since(start) < 50*time.Millisecond {
		t.Errorf("expected delay throttling between messages")
	}
}
