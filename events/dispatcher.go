package events

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
	dispatcher *Dispatcher
}

// Unregister removes the handler registration from dispatcher.
func (hr *HandlerRegistration) Unregister() {
	if hr.dispatcher != nil {
		hr.dispatcher.removeHandler(hr.eventType, hr.id)
	}
}

// Stats tracks event processing counts and errors.
type Stats struct {
	TotalReceived   uint64
	TotalDispatched uint64
	TotalErrors     uint64
	TotalDropped    uint64
}

// Dispatcher manages registration, filtering, queuing, and concurrent dispatching of events.
type Dispatcher struct {
	mu          sync.RWMutex
	handlers    map[EventType][]*HandlerRegistration
	middlewares []Middleware
	nextID      uint64

	// Event Queue & Backpressure
	queue       chan QueuedEvent
	queueCap    int
	stats       Stats
	workers     int
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup

	// Unknown fallback handler
	onUnknown func(rawType string, payload []byte)
}

// QueuedEvent bundles event payload with its target type.
type QueuedEvent struct {
	Type EventType
	Data any
}

// DispatcherConfig configures event queue parameters.
type DispatcherConfig struct {
	QueueCapacity int // default 1024
	Workers       int // default 4 workers for parallel dispatching
}

// NewDispatcher creates an event dispatcher instance.
func NewDispatcher(cfg ...DispatcherConfig) *Dispatcher {
	qCap := 1024
	workers := 4
	if len(cfg) > 0 {
		if cfg[0].QueueCapacity > 0 {
			qCap = cfg[0].QueueCapacity
		}
		if cfg[0].Workers > 0 {
			workers = cfg[0].Workers
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	d := &Dispatcher{
		handlers:    make(map[EventType][]*HandlerRegistration),
		middlewares: make([]Middleware, 0),
		queue:       make(chan QueuedEvent, qCap),
		queueCap:    qCap,
		workers:     workers,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start worker pool for parallel independent events
	for i := 0; i < workers; i++ {
		d.wg.Add(1)
		go d.workerLoop()
	}

	return d
}

// Use adds middleware to the event pipeline.
func (d *Dispatcher) Use(mw Middleware) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.middlewares = append(d.middlewares, mw)
}

// Register registers a handler with optional filter.
func (d *Dispatcher) Register(evt EventType, fn HandlerFunc, filter ...FilterFunc) *HandlerRegistration {
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

func (d *Dispatcher) removeHandler(evt EventType, id uint64) {
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

// OnUnknown sets a fallback handler for unexpected or novel server frames.
func (d *Dispatcher) OnUnknown(fn func(rawType string, payload []byte)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onUnknown = fn
}

// TriggerUnknown invokes the fallback handler safely.
func (d *Dispatcher) TriggerUnknown(rawType string, payload []byte) {
	d.mu.RLock()
	fn := d.onUnknown
	d.mu.RUnlock()

	if fn != nil {
		fn(rawType, payload)
	}
}

// Publish enqueues an event. If the queue is full, it implements drop policy with metrics logging.
func (d *Dispatcher) Publish(evt EventType, data any) {
	atomic.AddUint64(&d.stats.TotalReceived, 1)

	select {
	case d.queue <- QueuedEvent{Type: evt, Data: data}:
	default:
		// Backpressure policy: Drop and increment metric to prevent deadlock
		atomic.AddUint64(&d.stats.TotalDropped, 1)
		log.Printf("[highrisebot/events] Event queue full (capacity %d), dropped event %s", d.queueCap, evt)
	}
}

func (d *Dispatcher) workerLoop() {
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

func (d *Dispatcher) dispatchDirect(evt EventType, data any) {
	d.mu.RLock()
	handlers := append([]*HandlerRegistration(nil), d.handlers[evt]...)
	mws := append([]Middleware(nil), d.middlewares...)
	d.mu.RUnlock()

	atomic.AddUint64(&d.stats.TotalDispatched, 1)

	for _, reg := range handlers {
		// Event Filtering
		if reg.filter != nil && !reg.filter(data) {
			continue
		}

		// Apply Middlewares
		finalHandler := reg.handler
		for i := len(mws) - 1; i >= 0; i-- {
			finalHandler = mws[i](finalHandler)
		}

		// Execute with error recovery
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
func (d *Dispatcher) Stats() Stats {
	return Stats{
		TotalReceived:   atomic.LoadUint64(&d.stats.TotalReceived),
		TotalDispatched: atomic.LoadUint64(&d.stats.TotalDispatched),
		TotalErrors:     atomic.LoadUint64(&d.stats.TotalErrors),
		TotalDropped:    atomic.LoadUint64(&d.stats.TotalDropped),
	}
}

// Close stops queue workers gracefully.
func (d *Dispatcher) Close() {
	d.cancel()
	d.wg.Wait()
}
