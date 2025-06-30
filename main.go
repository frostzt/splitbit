package main

import (
	"errors"
	"fmt"
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

	if err := backend.HealthCheckService(); err != nil {
		log.Printf("failed to ping service: %v", err)
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

	// TODO: Right now hardcoded need to move these to a YAML Configuration
	newService := services.NewService("localhost", 8000)
	availableServices = append(availableServices, newService)

	config, err := internals.LoadConfig("./example-splitbit-config.yml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	fmt.Printf("config: %+v\n", config)

	backendSelector = services.NewRoundRobinSelector(availableServices)

	tcpListener, err = internals.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 8080})
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
		return
	}

	defer tcpListener.Close()
	go listenTCPConn()

	// Listen for interrupts
	interruptListener := make(chan os.Signal)
	signal.Notify(interruptListener, os.Interrupt)
	<-interruptListener

	log.Printf("interrupt signal received, stopping")
	os.Exit(0)
}
