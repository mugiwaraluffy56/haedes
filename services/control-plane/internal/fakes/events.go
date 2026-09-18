package fakes

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

type EventBus struct {
	mu           sync.Mutex
	clock        sandbox.Clock
	nextSequence int64
	nextSubID    int
	history      map[sandbox.CommandID][]sandbox.CommandEvent
	subscribers  map[sandbox.CommandID]map[int]chan sandbox.CommandEvent
}

func NewEventBus(clock sandbox.Clock) *EventBus {
	if clock == nil {
		clock = realClock{}
	}
	return &EventBus{
		clock:       clock,
		history:     make(map[sandbox.CommandID][]sandbox.CommandEvent),
		subscribers: make(map[sandbox.CommandID]map[int]chan sandbox.CommandEvent),
	}
}

func (bus *EventBus) Publish(commandID sandbox.CommandID, eventType, data string, result *sandbox.CommandResult) sandbox.CommandEvent {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.nextSequence++
	event := sandbox.CommandEvent{
		Sequence:  bus.nextSequence,
		Type:      eventType,
		CommandID: commandID,
		Data:      data,
		Result:    cloneCommandResult(result),
		At:        bus.clock.Now(),
	}
	bus.history[commandID] = append(bus.history[commandID], event)
	for _, subscriber := range bus.subscribers[commandID] {
		subscriber <- cloneCommandEvent(event)
	}
	return cloneCommandEvent(event)
}

func (bus *EventBus) Events(ctx context.Context, commandID sandbox.CommandID, lastEventID string) (<-chan sandbox.CommandEvent, error) {
	lastSequence, err := parseEventID(lastEventID)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	channel := make(chan sandbox.CommandEvent, 32)
	bus.mu.Lock()
	for _, event := range bus.history[commandID] {
		if event.Sequence > lastSequence {
			channel <- cloneCommandEvent(event)
		}
	}
	bus.nextSubID++
	subscriptionID := bus.nextSubID
	if bus.subscribers[commandID] == nil {
		bus.subscribers[commandID] = make(map[int]chan sandbox.CommandEvent)
	}
	bus.subscribers[commandID][subscriptionID] = channel
	bus.mu.Unlock()

	go func() {
		<-ctx.Done()
		bus.mu.Lock()
		delete(bus.subscribers[commandID], subscriptionID)
		close(channel)
		bus.mu.Unlock()
	}()
	return channel, nil
}

func parseEventID(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	sequence, err := strconv.ParseInt(value, 10, 64)
	if err != nil || sequence < 0 {
		return 0, fmt.Errorf("invalid event ID")
	}
	return sequence, nil
}

func cloneCommandEvent(event sandbox.CommandEvent) sandbox.CommandEvent {
	event.Result = cloneCommandResult(event.Result)
	return event
}

func cloneCommandResult(result *sandbox.CommandResult) *sandbox.CommandResult {
	if result == nil {
		return nil
	}
	clone := *result
	if result.ExitCode != nil {
		code := *result.ExitCode
		clone.ExitCode = &code
	}
	if result.FinishedAt != nil {
		finished := *result.FinishedAt
		clone.FinishedAt = &finished
	}
	return &clone
}
