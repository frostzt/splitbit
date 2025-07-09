package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

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

// handleTCPConn handles incoming TCP connections
func handleTCPConn(conn net.Conn, logger *internals.Logger) {
	logger.Info("Accepting TCP connection from %s with destination of %s", conn.RemoteAddr().String(), conn.LocalAddr().String())
	defer func() { _ = conn.Close() }()

	backend := backendSelector.SelectService()
	if backend == nil {
		logger.Warn("No backend selected")
		return
	}

	remoteConn, err := net.Dial("tcp", backend.Address())
	if err != nil {
		logger.Error("failed to connect to backend: %v", err)
		return
	}

	defer func() { _ = remoteConn.Close() }()

	// Try and connect to the original destination
	var streamWait sync.WaitGroup
	streamWait.Add(2)

	// TODO: Create transports that implement Writer interface to allow users to provide different writers
	monitor := &internals.Monitor{Logger: log.New(os.Stdout, "MONITOR: ", 0)}

	streamConn := func(dst io.Writer, src io.Reader) {
		defer streamWait.Done()

		buf := make([]byte, 1024)
		for {
			// Set deadline before reading
			internals.SetReadDeadline(src, 30*time.Second, logger)

			bytesRead, readErr := src.Read(buf)
			if readErr != nil {
				if errors.Is(readErr, io.EOF) {
					return
				}

				var netErr net.Error
				if errors.As(readErr, &netErr) && netErr.Timeout() {
					logger.Warn("read timeout from %s: %v\n", src.(*net.TCPConn).RemoteAddr().String(), readErr)
				} else {
					logger.Error("read error %v\n", readErr)
				}

				return
			}

			// Set deadline before writing
			internals.SetWriteDeadline(dst, 30*time.Second, logger)

			w := io.MultiWriter(dst, monitor)
			if _, writeError := w.Write(buf[:bytesRead]); writeError != nil {
				logger.Error("failed to write bytes: %v", writeError)
				return
			}
		}
	}

	logger.Debug("------------------- REMOTE CONN -------------------")
	logger.Debug("Local Address: %s", remoteConn.LocalAddr().String())
	logger.Debug("Remote Address: %s", remoteConn.RemoteAddr().String())

	logger.Debug("------------------- CONN -------------------")
	logger.Debug("Local Address: %s", conn.LocalAddr().String())
	logger.Debug("Remote Address: %s", conn.RemoteAddr().String())

	go streamConn(remoteConn, conn)
	go streamConn(conn, remoteConn)

	streamWait.Wait()
}

func listenTCPConn(logger *internals.Logger) {
	for {
		conn, err := tcpListener.Accept()

		logger.Info("Remote: %s → Local: %s", conn.RemoteAddr(), conn.LocalAddr())

		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Temporary() {
				logger.Debug("temporary error: %v", err)
				continue
			}

			logger.Fatal("failed to accept tcp conn: %v", err)
		}

		go handleTCPConn(conn, logger)
	}
}

func main() {
	var err error

	// Flag
	configFilePtr := flag.String("config", "./example-splitbit-config.yml", "a custom config file")
	flag.Parse()

	// Load configuration
	config, configErr := internals.LoadConfig(*configFilePtr)
	if configErr != nil {
		log.Fatalf("failed to load config: %v", configErr)
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

	tcpListener, err = internals.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: config.Port})
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	logger.Info("Splitbit ready to accept connection on %d", config.Port)

	defer func() { _ = tcpListener.Close() }()
	go listenTCPConn(logger)

	// Listen for interrupts
	interruptListener := make(chan os.Signal)
	signal.Notify(interruptListener, os.Interrupt)
	<-interruptListener

	logger.Warn("interrupt signal received, stopping...")
	cancel()
	os.Exit(0)
}
