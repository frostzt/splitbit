package services

import (
	"fmt"
	"log"
	"net"
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

// PingService performs health check on the provided service's health check route if the call fails
// it marks the [AliveStatus] as false otherwise marks it as true
func (s *Service) PingService() error {
	_, err := net.Dial("tcp", s.HealthCheckPath)
	if err != nil {
		log.Printf("Failed to connect to health check service at %s: %s\n", s.HealthCheckPath, err)
		s.AliveStatus = false // TODO: Need to implement some retry mechanism
		return err
	}

	s.AliveStatus = true
	return nil
}

// Address returns the network address in the host:port form
func (s *Service) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
