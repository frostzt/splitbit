package internals

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"syscall"
)

// SBTCPConn is a simple struct which wraps around the Conn
// helpful to provide Splitbit functionality
type SBTCPConn struct {
	net.Conn
}

func NewSplitbitTCPConn(conn net.Conn) *SBTCPConn {
	return &SBTCPConn{conn}
}

func (c *SBTCPConn) DialOriginalDestination(dontAssumeRemote bool) (*net.TCPConn, error) {
	rAddrPtr := c.RemoteAddr().(*net.TCPAddr)
	lAddrPtr := c.LocalAddr().(*net.TCPAddr)

	remoteSocketAddress, err := tcpAddrToSocketAddr(lAddrPtr)
	if err != nil {
		return nil, &net.OpError{Op: "dial", Err: fmt.Errorf("failed to parse local socket address: %w", err)}
	}

	localSocketAddress, err := tcpAddrToSocketAddr(rAddrPtr)
	if err != nil {
		return nil, &net.OpError{Op: "dial", Err: fmt.Errorf("failed to parse remote socket address: %w", err)}
	}

	// Create a new socket
	fd, err := syscall.Socket(tcpAddrFamily("tcp", lAddrPtr, rAddrPtr), syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, &net.OpError{Op: "dial", Err: fmt.Errorf("failed to create socket: %w", err)}
	}

	// Set SO_REUSEADDR to "ON"; this makes sure we're able to reconnect to this port without TCP_WAIT
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		syscall.Close(fd)
		return nil, &net.OpError{Op: "dial", Err: fmt.Errorf("socket option SO_REUSEADDR: %w", err)}
	}

	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.IP_TRANSPARENT, 1); err != nil {
		syscall.Close(fd)
		return nil, &net.OpError{Op: "dial", Err: fmt.Errorf("socket option IP_TRANSPARENT: %w", err)}
	}

	if err := syscall.SetNonblock(fd, true); err != nil {
		syscall.Close(fd)
		return nil, &net.OpError{Op: "dial", Err: fmt.Errorf("socket option SO_NONBLOCK: %w", err)}
	}

	if !dontAssumeRemote {
		if err := syscall.Bind(fd, localSocketAddress); err != nil {
			syscall.Close(fd)
			return nil, &net.OpError{Op: "dial", Err: fmt.Errorf("socket bind: %w", err)}
		}
	}

	if err := syscall

}

// tcpAddToSockerAddr will convert a TCPAddr
// into a Sockaddr that may be used when
// connecting and binding sockets
func tcpAddrToSocketAddr(addr *net.TCPAddr) (syscall.Sockaddr, error) {
	switch {
	case addr.IP.To4() != nil:
		ip := [4]byte{}
		copy(ip[:], addr.IP.To4())

		return &syscall.SockaddrInet4{Addr: ip, Port: addr.Port}, nil

	default:
		ip := [16]byte{}
		copy(ip[:], addr.IP.To16())

		zoneID, err := strconv.ParseUint(addr.Zone, 10, 32)
		if err != nil {
			return nil, err
		}

		return &syscall.SockaddrInet6{Addr: ip, Port: addr.Port, ZoneId: uint32(zoneID)}, nil
	}
}

// tcpAddrFamily will attempt to work
// out the address family based on the
// network and TCP addresses
func tcpAddrFamily(net string, localAddr, remoteAddr *net.TCPAddr) int {
	switch net[len(net)-1] {
	case '4':
		return syscall.AF_INET
	case '6':
		return syscall.AF_INET6
	}

	if (localAddr == nil || localAddr.IP.To4() != nil) &&
		(remoteAddr == nil || remoteAddr.IP.To4() != nil) {
		return syscall.AF_INET
	}
	return syscall.AF_INET6
}
