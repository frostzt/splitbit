package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/frostzt/splitbit/internals"
)

const (
	// StatePending represents that the service has just been registered and hasn't been probed yet
	StatePending internals.StateType = "PENDING"

	// StateAlive represents a service whose health check succeeded (for 3 subsequent calls)
	StateAlive internals.StateType = "ALIVE"

	// StateDown represents a service whose health check failed (for 3 subsequent calls)
	StateDown internals.StateType = "DOWN"

	// StateHalfOpen represents a service which has failed health check but is currently being tried again to recover
	StateHalfOpen internals.StateType = "HALF_OPEN"

	// EventSuccess triggers when a service has passed health check
	EventSuccess internals.EventType = "SUCCESS"

	// EventFailure triggers when a service has failed health check
	EventFailure internals.EventType = "FAILURE"

	// EventForceRecovery triggers when a service is probed again after it went down in an attempt
	// to being recovered
	EventForceRecovery internals.EventType = "RECOVERY"
)

// defaultHealthCheckDuration is the default time interval used in health checks
const defaultHealthCheckDuration = 5 * time.Second

type ServiceMetadata struct {
	// FailureCount tracks how many subsequent requests to this service has failed
	FailureCount int

	// LastRecoveryAttempt is the time at which this service was down and a forced recovery was initiated
	LastRecoveryAttempt time.Time
}

// Service corresponds to an Application server listening on the provided host and port
// services are registered at the very start of the load balancer
type Service struct {
	// Name of the service provided by the user
	Name string

	// Host URL of the service where the service is running
	Host string

	// Port on which the service is actively listening for connections
	Port int

	// FSM is the state machine which keeps track of the current state of this service
	FSM *internals.StateMachine

	// HealthCheckPath points to the health check path for this service
	HealthCheckPath string

	// HealthCheckDuration is the interval in which the proxy will hit the service
	HealthCheckDuration time.Duration

	// ConnectionCount tracks active count to this service
	ConnectionCount int

	// Weight for weighted-load balancing
	Weight int

	// Logger directly injected into service
	Logger *internals.Logger

	// Metadata contains information used by Splitbit to maintain this service
	Metadata ServiceMetadata
}

type ServiceOptions struct {
	Name            string
	HealthCheckPath string
	Weight          int
}

func NewService(host string, port int, opts *ServiceOptions, logger *internals.Logger) *Service {
	s := &Service{
		Name:                host,
		Host:                host,
		Port:                port,
		FSM:                 NewFSMForService(),
		HealthCheckPath:     "/health",
		HealthCheckDuration: defaultHealthCheckDuration,
		Weight:              0,
		Logger:              logger,
		Metadata: ServiceMetadata{
			FailureCount: 0,
		},
	}

	if opts != nil {
		if opts.Name != "" {
			s.Name = opts.Name
		}

		if opts.HealthCheckPath != "" {
			s.HealthCheckPath = opts.HealthCheckPath
		}

		if opts.Weight > 0 {
			s.Weight = opts.Weight
		}
	}

	return s
}

func NewFSMForService() *internals.StateMachine {
	return &internals.StateMachine{
		CurrentState: StatePending,
		States: internals.States{
			StatePending: internals.State{
				Events: internals.Events{
					EventSuccess: StateAlive,
					EventFailure: StateDown,
				},
			},
			StateAlive: internals.State{
				Action: &ServiceAliveAction{},
				Events: internals.Events{
					EventFailure: StateDown,
				},
			},
			StateDown: internals.State{
				Action: &ServiceDownAction{},
				Events: internals.Events{
					EventSuccess:       StateAlive,
					EventForceRecovery: StateHalfOpen,
				},
			},
			StateHalfOpen: internals.State{
				Events: internals.Events{
					EventFailure: StateDown,
					EventSuccess: StateAlive,
				},
			},
		},
	}
}

// HealthCheckService performs health check on the provided service's health check route if the call fails
// it marks the [AliveStatus] as false otherwise marks it as true
func (s *Service) HealthCheckService() error {
	url := fmt.Sprintf("http://%s:%d%s", s.Host, s.Port, s.HealthCheckPath)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Verify that the health check returned a 200 code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-200 health check %d", resp.StatusCode)
	}

	return nil
}

// PeriodicallyHealthCheckService will run health check onto every registered service every 5 seconds
// if the service fails the health check the service will be marked as `AliveStatus = false`
func (s *Service) PeriodicallyHealthCheckService(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := s.HealthCheckService()
			event := EventSuccess
			if err != nil {
				event = EventFailure
				s.Logger.Error("Health check failed for service %s: %s", s.Name, err)
			}

			// Do not resend Success events to state machine if the current state is already ALIVE
			if s.FSM.CurrentState == StateAlive && event == EventSuccess {
				continue
			}

			// Do not resend Failure events to state machine if the current state is already DOWN
			if s.FSM.CurrentState == StateDown && event == EventFailure {
				continue
			}

			// Update the state
			if err := s.FSM.SendEvent(event, &CommonActionCtx{svc: s}); err != nil {
				s.Logger.Warn("FSM rejected event %s for service %s, %v", event, s.Name, err)
			}
		}
	}
}

// Address returns the network address in the host:port form
func (s *Service) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
