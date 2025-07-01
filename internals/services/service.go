package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultHeartbeatInterval = 30 * time.Second

// Service corresponds to an Application server listening on the provided host and port
// services are registered at the very start of the load balancer
type Service struct {
	// Name of the service provided by the user
	Name string

	// Host URL of the service where the service is running
	Host string

	// Port on which the service is actively listening for connections
	Port int

	// AliveStatus is true if the last health check to the service was successful
	AliveStatus bool

	// HealthCheckPath points to the health check path for this service
	HealthCheckPath string

	// ConnectionCount tracks active count to this service
	ConnectionCount int

	// Weight for weighted-load balancing
	Weight int
}

type ServiceOptions struct {
	Name            string
	HealthCheckPath string
	Weight          int
}

func NewService(host string, port int, opts *ServiceOptions) *Service {
	s := &Service{
		Name:            host,
		Host:            host,
		Port:            port,
		AliveStatus:     true,
		HealthCheckPath: "/health",
		Weight:          0,
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

// HealthCheckService performs health check on the provided service's health check route if the call fails
// it marks the [AliveStatus] as false otherwise marks it as true
func (s *Service) HealthCheckService() error {
	url := fmt.Sprintf("http://%s:%d%s", s.Host, s.Port, s.HealthCheckPath)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		s.AliveStatus = false
		return err
	}
	defer resp.Body.Close()

	// Verify that the health check returned a 200 code
	if resp.StatusCode != http.StatusOK {
		s.AliveStatus = false
		return fmt.Errorf("non-200 health check %d", resp.StatusCode)
	}

	s.AliveStatus = true
	return nil
}

func (s *Service) Pinger(ctx context.Context, w io.Writer, reset <-chan time.Duration) {
	var interval time.Duration
	select {
	case <-ctx.Done():
		return

	case interval = <-reset:
	default:
	}

	if interval <= 0 {
		interval = defaultHeartbeatInterval
	}

	timer := time.NewTimer(interval)
	defer func() {
		if !timer.Stop() {
			<-timer.C
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case newInterval := <-reset:
			if !timer.Stop() {
				<-timer.C
			}
			if newInterval > 0 {
				interval = newInterval
			}

		case <-timer.C:
			if _, err := w.Write([]byte("ping")); err != nil {
				return
			}
		}

		_ = timer.Reset(interval)
	}
}

// Address returns the network address in the host:port form
func (s *Service) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
