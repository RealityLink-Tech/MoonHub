package delegation

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/logger"
)

// Topic constants for intercom events
const (
	// Task lifecycle
	TopicTaskQueued    = "task:queued"
	TopicTaskCompleted = "task:completed"
	TopicTaskFailed    = "task:failed"

	// Agent lifecycle
	TopicAgentCreated   = "agent:created"
	TopicAgentDismissed = "agent:dismissed"
	TopicAgentSuspended = "agent:suspended"
	TopicAgentRevived   = "agent:revived"

	// Memory events
	TopicMemoryUpdated      = "memory:updated"
	TopicMemoryConsolidated = "memory:consolidated"

	// Blackboard collaboration
	TopicBlackboardProposal = "blackboard:proposal"
	TopicBlackboardResolved = "blackboard:resolved"

	// Nudge/reminder events
	TopicNudgeScheduled  = "nudge:scheduled"
	TopicNudgeDelivered  = "nudge:delivered"
	TopicNudgeSuppressed = "nudge:suppressed"
)

// EventHandler is a function that handles an intercom event
type EventHandler func(ctx context.Context, event IntercomEvent)

// UnsubscribeFunc is returned when subscribing to events
type UnsubscribeFunc func()

type intercomSub struct {
	id uint64
	h  EventHandler
}

// Intercom is a pub/sub event system for delegation communication
type Intercom struct {
	mu        sync.RWMutex
	handlers  map[string][]intercomSub
	wildcards []intercomSub
	recent    map[string][]IntercomEvent
	maxRecent int
	nextID    atomic.Uint64
}

// NewIntercom creates a new intercom event system
func NewIntercom(maxRecentEvents int) *Intercom {
	return &Intercom{
		handlers:  make(map[string][]intercomSub),
		recent:    make(map[string][]IntercomEvent),
		maxRecent: maxRecentEvents,
	}
}

// On subscribes to events on a specific topic
// Returns an unsubscribe function
func (i *Intercom) On(topic string, handler EventHandler) UnsubscribeFunc {
	sid := i.nextID.Add(1)
	sub := intercomSub{id: sid, h: handler}

	i.mu.Lock()
	i.handlers[topic] = append(i.handlers[topic], sub)
	i.mu.Unlock()

	logger.DebugCF("delegation", "Subscribed to topic", map[string]any{
		"topic": topic,
	})

	return func() {
		i.mu.Lock()
		defer i.mu.Unlock()
		subs := i.handlers[topic]
		for j, s := range subs {
			if s.id == sid {
				i.handlers[topic] = append(subs[:j], subs[j+1:]...)
				break
			}
		}
	}
}

// OnAny subscribes to all events
// Returns an unsubscribe function
func (i *Intercom) OnAny(handler EventHandler) UnsubscribeFunc {
	sid := i.nextID.Add(1)
	sub := intercomSub{id: sid, h: handler}

	i.mu.Lock()
	i.wildcards = append(i.wildcards, sub)
	i.mu.Unlock()

	return func() {
		i.mu.Lock()
		defer i.mu.Unlock()
		for j, s := range i.wildcards {
			if s.id == sid {
				i.wildcards = append(i.wildcards[:j], i.wildcards[j+1:]...)
				break
			}
		}
	}
}

// Emit publishes an event to all subscribers
func (i *Intercom) Emit(ctx context.Context, topic, userID string, data map[string]any) {
	event := IntercomEvent{
		Topic:     topic,
		UserID:    userID,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}

	// Store in recent events
	i.storeRecent(event)

	// Get handlers under read lock
	i.mu.RLock()
	subs := i.handlers[topic]
	handlers := make([]EventHandler, len(subs))
	for j, s := range subs {
		handlers[j] = s.h
	}
	wildSubs := i.wildcards
	wildcards := make([]EventHandler, len(wildSubs))
	for j, s := range wildSubs {
		wildcards[j] = s.h
	}
	i.mu.RUnlock()

	// Call topic-specific handlers
	for _, h := range handlers {
		i.callHandler(ctx, h, event)
	}

	// Call wildcard handlers
	for _, h := range wildcards {
		i.callHandler(ctx, h, event)
	}

	logger.DebugCF("delegation", "Emitted event", map[string]any{
		"topic":    topic,
		"user_id":  userID,
		"handlers": len(handlers) + len(wildcards),
	})
}

func (i *Intercom) callHandler(ctx context.Context, h EventHandler, event IntercomEvent) {
	defer func() {
		if r := recover(); r != nil {
			logger.ErrorCF("delegation", "Handler panic recovered", map[string]any{
				"topic": event.Topic,
				"panic": r,
			})
		}
	}()
	h(ctx, event)
}

func (i *Intercom) storeRecent(event IntercomEvent) {
	i.mu.Lock()
	defer i.mu.Unlock()

	events := i.recent[event.Topic]
	events = append(events, event)

	// Trim to max recent
	if len(events) > i.maxRecent {
		events = events[len(events)-i.maxRecent:]
	}
	i.recent[event.Topic] = events
}

// Recent returns the most recent events for a topic
func (i *Intercom) Recent(topic string, limit int) []IntercomEvent {
	i.mu.RLock()
	defer i.mu.RUnlock()

	events := i.recent[topic]
	if limit <= 0 || limit > len(events) {
		limit = len(events)
	}

	// Return most recent events (from the end)
	result := make([]IntercomEvent, limit)
	copy(result, events[len(events)-limit:])
	return result
}

// RecentAll returns recent events across all topics
func (i *Intercom) RecentAll(limit int) []IntercomEvent {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var all []IntercomEvent
	for _, events := range i.recent {
		all = append(all, events...)
	}

	// Sort by timestamp (newest first)
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[i].Timestamp < all[j].Timestamp {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}

	return all
}

// Clear removes all handlers and recent events
func (i *Intercom) Clear() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.handlers = make(map[string][]intercomSub)
	i.wildcards = nil
	i.recent = make(map[string][]IntercomEvent)
}

// Stats returns statistics about the intercom
func (i *Intercom) Stats() map[string]any {
	i.mu.RLock()
	defer i.mu.RUnlock()

	topicCounts := make(map[string]int)
	totalRecent := 0
	for topic, events := range i.recent {
		topicCounts[topic] = len(events)
		totalRecent += len(events)
	}

	return map[string]any{
		"topics":          len(i.handlers),
		"wildcard_count":  len(i.wildcards),
		"recent_events":   totalRecent,
		"recent_by_topic": topicCounts,
	}
}

// BlackboardProposalWithStatus extends BlackboardProposal with resolution status
type BlackboardProposalWithStatus struct {
	BlackboardProposal
	Resolved   bool
	Resolution string
	ResolvedAt time.Time
}

// Blackboard implements a simple blackboard pattern for collaboration
type Blackboard struct {
	mu         sync.RWMutex
	proposals  []BlackboardProposalWithStatus
	maxEntries int
	intercom   *Intercom // Optional intercom for emitting events
}

// NewBlackboard creates a new blackboard for collaboration
func NewBlackboard(maxEntries int) *Blackboard {
	return &Blackboard{
		proposals:  make([]BlackboardProposalWithStatus, 0),
		maxEntries: maxEntries,
	}
}

// SetIntercom sets the intercom instance for emitting events
func (b *Blackboard) SetIntercom(intercom *Intercom) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.intercom = intercom
}

// Post adds a proposal to the blackboard.
// userID is the delegation partition (IntercomEvent.UserID); subAgentID identifies the proposing sub-agent.
func (b *Blackboard) Post(userID, subAgentID, proposal string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.proposals = append(b.proposals, BlackboardProposalWithStatus{
		BlackboardProposal: BlackboardProposal{
			SubAgentID: subAgentID,
			Proposal:   proposal,
			Timestamp:  time.Now(),
		},
		Resolved: false,
	})

	// Trim old entries
	if len(b.proposals) > b.maxEntries {
		b.proposals = b.proposals[len(b.proposals)-b.maxEntries:]
	}

	logger.DebugCF("delegation", "Blackboard proposal posted", map[string]any{
		"sub_agent_id": subAgentID,
		"total":        len(b.proposals),
	})

	// Emit event if intercom is set
	if b.intercom != nil {
		ic := b.intercom
		go ic.Emit(context.Background(), TopicBlackboardProposal, userID, map[string]any{
			"sub_agent_id": subAgentID,
			"proposal":     proposal,
		})
	}
}

// Resolve marks a proposal as resolved.
func (b *Blackboard) Resolve(userID, subAgentID, resolution string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := range b.proposals {
		if b.proposals[i].SubAgentID == subAgentID && !b.proposals[i].Resolved {
			b.proposals[i].Resolved = true
			b.proposals[i].Resolution = resolution
			b.proposals[i].ResolvedAt = time.Now()
			propText := b.proposals[i].Proposal

			logger.DebugCF("delegation", "Blackboard proposal resolved", map[string]any{
				"sub_agent_id": subAgentID,
				"resolution":   resolution,
			})

			// Emit event if intercom is set
			if b.intercom != nil {
				ic := b.intercom
				go ic.Emit(context.Background(), TopicBlackboardResolved, userID, map[string]any{
					"sub_agent_id": subAgentID,
					"resolution":   resolution,
					"proposal":     propText,
				})
			}
			return
		}
	}
}

// GetAll returns all proposals
func (b *Blackboard) GetAll() []BlackboardProposalWithStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]BlackboardProposalWithStatus, len(b.proposals))
	copy(result, b.proposals)
	return result
}

// GetByAgent returns proposals from a specific agent
func (b *Blackboard) GetByAgent(subAgentID string) []BlackboardProposalWithStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []BlackboardProposalWithStatus
	for _, p := range b.proposals {
		if p.SubAgentID == subAgentID {
			result = append(result, p)
		}
	}
	return result
}

// GetUnresolved returns all unresolved proposals
func (b *Blackboard) GetUnresolved() []BlackboardProposalWithStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []BlackboardProposalWithStatus
	for _, p := range b.proposals {
		if !p.Resolved {
			result = append(result, p)
		}
	}
	return result
}

// GetResolved returns all resolved proposals
func (b *Blackboard) GetResolved() []BlackboardProposalWithStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []BlackboardProposalWithStatus
	for _, p := range b.proposals {
		if p.Resolved {
			result = append(result, p)
		}
	}
	return result
}

// Clear removes all proposals
func (b *Blackboard) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.proposals = b.proposals[:0]
}
