package highrisebot

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
)

// EventType defines strings for Highrise events.
type EventType string

const (
	EventReady     EventType = "ready"
	EventChat      EventType = "chat"
	EventWhisper   EventType = "whisper"
	EventUserJoin  EventType = "user_join"
	EventUserLeave EventType = "user_leave"
	EventUserMove  EventType = "user_move"
	EventEmote     EventType = "emote"
	EventReaction  EventType = "reaction"
	EventTip       EventType = "tip"
	EventVoice     EventType = "voice"
	EventMessage   EventType = "message"
	EventUnknown   EventType = "unknown"
)

// ChatEvent represents a public room chat event.
type ChatEvent struct {
	User      User   `json:"user"`
	Message   string `json:"message"`
	Whisper   bool   `json:"whisper,omitempty"`
	MessageID string `json:"message_id,omitempty"`
}

// WhisperEvent represents a private direct whisper sent to the bot.
type WhisperEvent struct {
	User    User   `json:"user"`
	Message string `json:"message"`
}

// UserJoinedEvent represents when a user enters the room.
type UserJoinedEvent struct {
	User     User     `json:"user"`
	Position Position `json:"position"`
}

// UserLeftEvent represents when a user leaves the room.
type UserLeftEvent struct {
	User User `json:"user"`
}

// UserMovedEvent represents when a user walks or teleports to a new coordinate.
type UserMovedEvent struct {
	User     User     `json:"user"`
	Position Position `json:"position"`
}

// EmoteEvent represents when a user performs an emote.
type EmoteEvent struct {
	User     User   `json:"user"`
	EmoteID  string `json:"emote_id"`
	Receiver *User  `json:"receiver,omitempty"`
}

// ReactionEvent represents when a reaction emoji is sent towards a user.
type ReactionEvent struct {
	User     User   `json:"user"`
	Reaction string `json:"reaction"`
	Receiver User   `json:"receiver"`
}

// TipReactionEvent represents when a tip is given to another user.
type TipReactionEvent struct {
	Sender   User `json:"sender"`
	Receiver User `json:"receiver"`
	Item     Item `json:"item"`
}

// VoiceEvent represents voice chat state updates.
type VoiceEvent struct {
	Users  []User `json:"users"`
	Status string `json:"status"`
}

// MessageEvent represents incoming direct inbox messages sent to the bot.
type MessageEvent struct {
	UserID            string `json:"user_id"`
	ConversationID    string `json:"conversation_id"`
	IsNewConversation bool   `json:"is_new_conversation"`
}

// HandlerFunc is the function signature for generic event handlers.
type HandlerFunc func(ctx context.Context, event any) error

// FilterFunc determines whether an event should reach the handler.
type FilterFunc func(event any) bool

// Middleware wraps a HandlerFunc for logging, recovery, metrics, etc.
type Middleware func(HandlerFunc) HandlerFunc

// HandlerRegistration represents an active handler that can be unregistered.
type HandlerRegistration struct {
	id         uint64
	eventType  EventType
	handler    HandlerFunc
	filter     FilterFunc
	dispatcher *EventDispatcher
}

// Unregister removes the handler registration from dispatcher.
func (hr *HandlerRegistration) Unregister() {
	if hr.dispatcher != nil {
		hr.dispatcher.removeHandler(hr.eventType, hr.id)
	}
}

// EventStats tracks event processing counts and errors.
type EventStats struct {
	TotalReceived   uint64
	TotalDispatched uint64
	TotalErrors     uint64
	TotalDropped    uint64
}

// EventDispatcher manages registration, filtering, queuing, and concurrent dispatching of events.
type EventDispatcher struct {
	mu          sync.RWMutex
	handlers    map[EventType][]*HandlerRegistration
	middlewares []Middleware
	nextID      uint64

	queue    chan QueuedEvent
	queueCap int
	stats    EventStats
	workers  int
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	onUnknown func(rawType string, payload []byte)
}

// QueuedEvent bundles event payload with its target type.
type QueuedEvent struct {
	Type EventType
	Data any
}

// NewEventDispatcher creates an event dispatcher instance.
func NewEventDispatcher(queueCapacity, workers int) *EventDispatcher {
	if queueCapacity <= 0 {
		queueCapacity = 1024
	}
	if workers <= 0 {
		workers = 4
	}

	ctx, cancel := context.WithCancel(context.Background())
	d := &EventDispatcher{
		handlers:    make(map[EventType][]*HandlerRegistration),
		middlewares: make([]Middleware, 0),
		queue:       make(chan QueuedEvent, queueCapacity),
		queueCap:    queueCapacity,
		workers:     workers,
		ctx:         ctx,
		cancel:      cancel,
	}

	for i := 0; i < workers; i++ {
		d.wg.Add(1)
		go d.workerLoop()
	}

	return d
}

// Use adds middleware to the event pipeline.
func (d *EventDispatcher) Use(mw Middleware) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.middlewares = append(d.middlewares, mw)
}

// Register registers a handler with optional filter.
func (d *EventDispatcher) Register(evt EventType, fn HandlerFunc, filter ...FilterFunc) *HandlerRegistration {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.nextID++
	reg := &HandlerRegistration{
		id:         d.nextID,
		eventType:  evt,
		handler:    fn,
		dispatcher: d,
	}
	if len(filter) > 0 {
		reg.filter = filter[0]
	}

	d.handlers[evt] = append(d.handlers[evt], reg)
	return reg
}

func (d *EventDispatcher) removeHandler(evt EventType, id uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	list := d.handlers[evt]
	for i, reg := range list {
		if reg.id == id {
			d.handlers[evt] = append(list[:i], list[i+1:]...)
			break
		}
	}
}

// OnUnknown sets a fallback handler for unexpected server frames.
func (d *EventDispatcher) OnUnknown(fn func(rawType string, payload []byte)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onUnknown = fn
}

// TriggerUnknown invokes the fallback handler safely.
func (d *EventDispatcher) TriggerUnknown(rawType string, payload []byte) {
	d.mu.RLock()
	fn := d.onUnknown
	d.mu.RUnlock()

	if fn != nil {
		fn(rawType, payload)
	}
}

// Publish enqueues an event.
func (d *EventDispatcher) Publish(evt EventType, data any) {
	atomic.AddUint64(&d.stats.TotalReceived, 1)

	select {
	case d.queue <- QueuedEvent{Type: evt, Data: data}:
	default:
		atomic.AddUint64(&d.stats.TotalDropped, 1)
		log.Printf("[highrisebot/events] Event queue full (cap %d), dropped %s", d.queueCap, evt)
	}
}

func (d *EventDispatcher) workerLoop() {
	defer d.wg.Done()
	for {
		select {
		case <-d.ctx.Done():
			return
		case qEvent, ok := <-d.queue:
			if !ok {
				return
			}
			d.dispatchDirect(qEvent.Type, qEvent.Data)
		}
	}
}

func (d *EventDispatcher) dispatchDirect(evt EventType, data any) {
	d.mu.RLock()
	handlers := append([]*HandlerRegistration(nil), d.handlers[evt]...)
	mws := append([]Middleware(nil), d.middlewares...)
	d.mu.RUnlock()

	atomic.AddUint64(&d.stats.TotalDispatched, 1)

	for _, reg := range handlers {
		if reg.filter != nil && !reg.filter(data) {
			continue
		}

		finalHandler := reg.handler
		for i := len(mws) - 1; i >= 0; i-- {
			finalHandler = mws[i](finalHandler)
		}

		go func(h HandlerFunc) {
			defer func() {
				if r := recover(); r != nil {
					atomic.AddUint64(&d.stats.TotalErrors, 1)
					log.Printf("[highrisebot/events] Panic in handler for %s: %v", evt, r)
				}
			}()

			if err := h(d.ctx, data); err != nil {
				atomic.AddUint64(&d.stats.TotalErrors, 1)
				log.Printf("[highrisebot/events] Error handling %s: %v", evt, err)
			}
		}(finalHandler)
	}
}

// Stats returns a copy of current event statistics.
func (d *EventDispatcher) Stats() EventStats {
	return EventStats{
		TotalReceived:   atomic.LoadUint64(&d.stats.TotalReceived),
		TotalDispatched: atomic.LoadUint64(&d.stats.TotalDispatched),
		TotalErrors:     atomic.LoadUint64(&d.stats.TotalErrors),
		TotalDropped:    atomic.LoadUint64(&d.stats.TotalDropped),
	}
}

// Close stops queue workers gracefully.
func (d *EventDispatcher) Close() {
	d.cancel()
	d.wg.Wait()
}
