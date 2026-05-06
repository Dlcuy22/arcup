// Package webhook provides notification mechanisms for the arcup lifecycle.
//
// Key Components:
//   - Notifier: Interface for sending notifications
//   - Manager: Holds multiple notifiers and broadcasts events
//   - Event: Data payload for a specific lifecycle event
//
// Dependencies:
//   - net/http: For sending webhook payloads
//
// Error Types:
//   - ErrInvalidPayload: Returned when the event payload cannot be marshaled
//
// Example:
//   mgr := webhook.NewManager()
//   mgr.Add(webhook.NewDiscord(webhookURL))
//   mgr.Notify(webhook.Event{Type: webhook.EventStarted})
package webhook

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type EventType string

const (
	EventStarted  EventType = "started"
	EventArchived EventType = "archived"
	EventUploaded EventType = "uploaded"
	EventFailed   EventType = "failed"
)

// Event holds metadata about the backup state.
type Event struct {
	Type     EventType
	Archive  string
	Size     int64
	Hash     string
	Duration time.Duration
	Error    error
}

// Notifier defines the interface for different webhook implementations.
type Notifier interface {
	Notify(event Event) error
}

// Manager manages multiple notifiers and broadcasts events to all of them.
type Manager struct {
	notifiers []Notifier
}

func NewManager() *Manager {
	return &Manager{}
}

// Add adds a new notifier to the manager.
func (m *Manager) Add(n Notifier) {
	if n != nil {
		m.notifiers = append(m.notifiers, n)
	}
}

// Notify broadcasts the event to all registered notifiers asynchronously.
func (m *Manager) Notify(event Event) {
	if len(m.notifiers) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, n := range m.notifiers {
		wg.Add(1)
		go func(notifier Notifier) {
			defer wg.Done()
			if err := notifier.Notify(event); err != nil {
				log.Warn().Err(err).Msg("webhook notification failed")
			}
		}(n)
	}
	wg.Wait()
}
