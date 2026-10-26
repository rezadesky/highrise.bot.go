package events_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rezadesky/highrisebot/events"
	"github.com/rezadesky/highrisebot/models"
)

// Fixture payloads
var chatFixture = `{"_type":"ChatEvent","user":{"id":"user_1","username":"alice"},"message":"hello world"}`

func TestEventDispatcherMultiHandlerAndFiltering(t *testing.T) {
	d := events.NewDispatcher(events.DispatcherConfig{QueueCapacity: 64, Workers: 2})
	defer d.Close()

	var count uint32
	var filteredCount uint32

	// Handler 1: counts all chat events
	reg1 := d.Register(events.EventChat, func(ctx context.Context, e any) error {
		atomic.AddUint32(&count, 1)
		return nil
	})

	// Handler 2: only handles messages containing "secret"
	d.Register(events.EventChat, func(ctx context.Context, e any) error {
		atomic.AddUint32(&filteredCount, 1)
		return nil
	}, func(e any) bool {
		chat, ok := e.(*models.ChatEvent)
		return ok && chat.Message == "secret"
	})

	// Publish normal chat
	d.Publish(events.EventChat, &models.ChatEvent{Message: "hello world"})
	// Publish secret chat
	d.Publish(events.EventChat, &models.ChatEvent{Message: "secret"})

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadUint32(&count) != 2 {
		t.Errorf("expected 2 chat events handled, got %d", count)
	}
	if atomic.LoadUint32(&filteredCount) != 1 {
		t.Errorf("expected 1 filtered secret event handled, got %d", filteredCount)
	}

	// Test Unregister
	reg1.Unregister()
	d.Publish(events.EventChat, &models.ChatEvent{Message: "another message"})

	time.Sleep(50 * time.Millisecond)
	if atomic.LoadUint32(&count) != 2 {
		t.Errorf("expected count to remain 2 after unregister, got %d", count)
	}
}

func TestEventMiddlewareAndPanicRecovery(t *testing.T) {
	d := events.NewDispatcher()
	defer d.Close()

	var middlewareExecuted bool
	d.Use(func(next events.HandlerFunc) events.HandlerFunc {
		return func(ctx context.Context, e any) error {
			middlewareExecuted = true
			return next(ctx, e)
		}
	})

	// Handler that panics
	d.Register(events.EventUserJoin, func(ctx context.Context, e any) error {
		panic("simulated panic in handler")
	})

	// Handler that returns error
	d.Register(events.EventUserLeave, func(ctx context.Context, e any) error {
		return errors.New("simulated error")
	})

	d.Publish(events.EventUserJoin, &models.UserJoinedEvent{})
	d.Publish(events.EventUserLeave, &models.UserLeftEvent{})

	time.Sleep(100 * time.Millisecond)

	if !middlewareExecuted {
		t.Error("expected middleware to be executed")
	}

	stats := d.Stats()
	if stats.TotalErrors == 0 {
		t.Error("expected stats to record error & panic recovery")
	}
}
