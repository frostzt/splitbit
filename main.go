package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/frostzt/splitbit/internals"
)

func handleConnection(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Println(fmt.Errorf("error closing connection: %s", err))
		}
	}()

	// Extract details from Conn
	localAddr := conn.LocalAddr().String()
	remoteAddr := conn.RemoteAddr().String()

	log.Println("Connection from " + remoteAddr + " to " + localAddr)

	// Read the request
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println(fmt.Errorf("error reading: %s", err))
		return
	}

	request := string(buffer[:n])
	log.Printf("Read %d bytes from %s\n", n, remoteAddr)

	isHttpRequest := internals.IsHTTPRequest(request)
	if isHttpRequest {
		log.Println("Unknown request type or non-http request")

		// Send raw TCP response for non-HTTP clients
		response := fmt.Sprintf("Hello from load balancer! Time: %s\n", time.Now().Format(time.RFC3339))
		_, err = conn.Write([]byte(response))
		if err != nil {
			fmt.Println(fmt.Errorf("error writing: %s", err))
		}

		return
	}

	// Send HTTP Response
	httpResponse := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Content-Type: text/plain\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"Hello from load balancer! Time: %s\n",
		len("Hello from load balancer! Time: "+time.Now().Format(time.RFC3339)+"\n"),
		time.Now().Format(time.RFC3339))

	// Write back the response
	_, err = conn.Write([]byte(httpResponse))
	if err != nil {
		fmt.Println(fmt.Errorf("error writing: %s", err))
	}
}

func main() {
	tcpListener, err := net.ListenTCP()
}
