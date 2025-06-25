package services

import (
	"fmt"
	"net/http"
	"time"
)

// Service corresponds to an Application server listening on the provided host and port
// services are registered at the very start of the load balancer
type Service struct {
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

func NewService(host string, port int) *Service {
	return &Service{
		Host:            host,
		Port:            port,
		AliveStatus:     true,
		HealthCheckPath: "/health",
		Weight:          0,
	}
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

// Address returns the network address in the host:port form
func (s *Service) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
