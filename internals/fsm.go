package internals

import (
	"errors"
	"sync"
)

// CREDIT: https://venilnoronha.io/a-simple-state-machine-framework-in-go

// ErrEventRejected is returned when the state machine cannot process
// an event given its current state
var ErrEventRejected = errors.New("event rejected")

var ErrInvalidConfig = errors.New("invalid config")

const (
	// Default represents the default status of the machine
	Default StateType = ""

	// NOOP represents a no-op event
	NOOP EventType = "NOOP"
)

// StateType represents an extensible state type in the machine
type StateType string

// EventType represents an extensible event type in the machine
type EventType string

// EventContext contains any context to be passed to the action implementation
type EventContext any

// Action represents action to be executed in a state
type Action interface {
	Execute(eventCtx EventContext) EventType
}

// Events represents mapping of events and states
type Events map[EventType]StateType

// State binds a state to an action and the events that it can handle
type State struct {
	Action Action
	Events Events
}

// States represents a mapping of all the states and their implementation
type States map[StateType]State

// StateMachine is a state machine
type StateMachine struct {
	// PreviousState is the previous state this machine was in
	PreviousState StateType

	// CurrentState is the current state of this machine
	CurrentState StateType

	// States contains config for states and events handled by the machine
	States States

	// mutex to ensure that only one is processed at a time by the machine
	mutex sync.Mutex
}

// getNextState returns the next state that a given machine can transition to given its current
// state, or an error if the event can't be handled in the given state
func (s *StateMachine) getNextState(event EventType) (StateType, error) {
	if state, ok := s.States[s.CurrentState]; ok {
		if state.Events != nil {
			if next, ok := state.Events[event]; ok {
				return next, nil
			}
		}
	}

	return Default, ErrEventRejected
}

// SendEvent sends an event to the state machine
func (s *StateMachine) SendEvent(event EventType, eventCtx EventContext) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for {
		// Determine the next state given the current state of the machine
		nextState, err := s.getNextState(event)
		if err != nil {
			return ErrEventRejected
		}

		// Get the state definition for the next state
		state, ok := s.States[nextState]
		if !ok || state.Action == nil {
			return ErrInvalidConfig
		}

		// Transition
		s.PreviousState = s.CurrentState
		s.CurrentState = nextState

		nextEvent := state.Action.Execute(eventCtx)
		if nextEvent == NOOP {
			return nil
		}

		event = nextEvent
	}
}
