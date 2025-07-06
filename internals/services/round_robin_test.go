package services

import (
	"testing"

	"github.com/frostzt/splitbit/internals"
)

func TestNewRoundRobinSelector(t *testing.T) {
	testService := &Service{
		Name: "TestService",
		Host: "Test",
		Port: 9990,
		FSM: &internals.StateMachine{
			PreviousState: "",
			CurrentState:  StateAlive,
			States:        nil,
		},
		HealthCheckPath:     "/health",
		HealthCheckDuration: 0,
		ConnectionCount:     0,
		Weight:              0,
		Logger:              nil,
		Metadata:            ServiceMetadata{},
	}

	testService2 := &Service{
		Name: "TestService2",
		Host: "Test2",
		Port: 9991,
		FSM: &internals.StateMachine{
			PreviousState: "",
			CurrentState:  StateAlive,
			States:        nil,
		},
		HealthCheckPath:     "/health",
		HealthCheckDuration: 0,
		ConnectionCount:     0,
		Weight:              0,
		Logger:              nil,
		Metadata:            ServiceMetadata{},
	}

	testService3 := &Service{
		Name: "TestService3",
		Host: "Test3",
		Port: 9992,
		FSM: &internals.StateMachine{
			PreviousState: "",
			CurrentState:  StateDown,
			States:        nil,
		},
		HealthCheckPath:     "/health",
		HealthCheckDuration: 0,
		ConnectionCount:     0,
		Weight:              0,
		Logger:              nil,
		Metadata:            ServiceMetadata{},
	}

	services := []*Service{testService, testService2, testService3}
	selector := NewRoundRobinSelector(services)

	for i := 0; i < 10; i++ {
		service := selector.SelectService()

		if service == nil {
			t.Errorf("service is nil")
		}

		if service.Host == testService3.Host {
			t.Errorf("a dead service was selected %s, current state: %s", service.Name, service.FSM.CurrentState)
		}
	}
}
