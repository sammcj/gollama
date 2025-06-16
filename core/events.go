package core

import (
	"sync"
	"time"
)

// Event represents a system event
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
	Time time.Time   `json:"time"`
}

// Event type constants
const (
	EventModelUpdated     = "model_updated"
	EventModelDeleted     = "model_deleted"
	EventModelCreated     = "model_created"
	EventModelRunning     = "model_running"
	EventModelStopped     = "model_stopped"
	EventConfigUpdated    = "config_updated"
	EventServiceStarted   = "service_started"
	EventServiceStopped   = "service_stopped"
	EventError            = "error"
)

// EventBus manages event subscriptions and emissions
type EventBus struct {
	subscribers map[string][]chan Event
	mutex       sync.RWMutex
	closed      bool
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan Event),
	}
}

// Subscribe subscribes to events of a specific type
func (eb *EventBus) Subscribe(eventType string) <-chan Event {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if eb.closed {
		// Return a closed channel if the event bus is closed
		ch := make(chan Event)
		close(ch)
		return ch
	}

	ch := make(chan Event, 10) // Buffered channel to prevent blocking
	eb.subscribers[eventType] = append(eb.subscribers[eventType], ch)
	return ch
}

// Emit emits an event to all subscribers
func (eb *EventBus) Emit(event Event) {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	if eb.closed {
		return
	}

	if subscribers, exists := eb.subscribers[event.Type]; exists {
		for _, ch := range subscribers {
			select {
			case ch <- event:
			default:
				// Channel is full, skip this subscriber to prevent blocking
			}
		}
	}
}

// Close closes the event bus and all subscriber channels
func (eb *EventBus) Close() {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if eb.closed {
		return
	}

	eb.closed = true

	// Close all subscriber channels
	for _, subscribers := range eb.subscribers {
		for _, ch := range subscribers {
			close(ch)
		}
	}

	// Clear subscribers
	eb.subscribers = make(map[string][]chan Event)
}

// GetSubscriberCount returns the number of subscribers for a given event type
func (eb *EventBus) GetSubscriberCount(eventType string) int {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	if subscribers, exists := eb.subscribers[eventType]; exists {
		return len(subscribers)
	}
	return 0
}
