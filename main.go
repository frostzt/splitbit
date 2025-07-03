package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"

	"github.com/frostzt/splitbit/internals"
	"github.com/frostzt/splitbit/internals/services"
)

var (
	// tcpListener is the TCP socket which will listen to the TCP connections
	tcpListener net.Listener

	// availableService are a list of services provided/registered by the user
	availableServices []*services.Service

	// backendSelector selects one of the available services based on the algorithm
	backendSelector services.BackendSelector
)

func handleTCPConn(conn net.Conn) {
	log.Printf("Accepting TCP connection from %s with destination of %s", conn.RemoteAddr().String(), conn.LocalAddr().String())
	defer func() { _ = conn.Close() }()

	backend := backendSelector.SelectService()
	if backend == nil {
		log.Printf("No backend selected")
		return
	}

	remoteConn, err := net.Dial("tcp", backend.Address())
	if err != nil {
		log.Printf("failed to connect to backend: %v", err)
		return
	}

	defer func() { _ = remoteConn.Close() }()

	// Try and connect to the original destination
	var streamWait sync.WaitGroup
	streamWait.Add(2)

	streamConn := func(dst io.Writer, src io.Reader) {
		io.Copy(dst, src)
		streamWait.Done()
	}

	go streamConn(remoteConn, conn)
	go streamConn(conn, remoteConn)

	streamWait.Wait()
}

func listenTCPConn() {
	for {
		conn, err := tcpListener.Accept()

		log.Printf("Remote: %s → Local: %s", conn.RemoteAddr(), conn.LocalAddr())

		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Temporary() {
				log.Printf("temporary error: %v", err)
				continue
			}

			log.Fatalf("failed to accept tcp conn: %v", err)
		}

		go handleTCPConn(conn)
	}
}

func main() {
	var err error

	// Load configuration
	config, err := internals.LoadConfig("./splitbit-config.yml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Logger
	logger := internals.NewLogger(config.Env)

	logger.Info("Starting splitbit service with env %s", config.Env)

	// Initialize context for services registered
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Register backend services
	for _, service := range config.Backends {
		options := &services.ServiceOptions{
			Weight:          service.Weight,
			HealthCheckPath: service.HealthCheck,
		}

		svc := services.NewService(service.Host, service.Port, options, logger)
		availableServices = append(availableServices, svc)

		// Start health check loop
		go svc.PeriodicallyHealthCheckService(ctx)

		logger.Info("Registered service: %s", service.Name)
	}

	backendSelector = services.NewRoundRobinSelector(availableServices)

	tcpListener, err = internals.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 8080})
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	defer func() { _ = tcpListener.Close() }()
	go listenTCPConn()

	// Listen for interrupts
	interruptListener := make(chan os.Signal)
	signal.Notify(interruptListener, os.Interrupt)
	<-interruptListener

	logger.Warn("interrupt signal received, stopping...")
	cancel()
	os.Exit(0)
}
