package internals

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// SBTCPConn is a simple struct which wraps around the Conn
// helpful to provide Splitbit functionality
type SBTCPConn struct {
	net.Conn
}

// Listener is a simple TCP Listener
type Listener struct {
	base net.Listener
}

func (l *Listener) Addr() net.Addr {
	return l.base.Addr()
}

func (l *Listener) Accept() (net.Conn, error) {
	return l.AcceptSB()
}

func (l *Listener) AcceptSB() (*SBTCPConn, error) {
	tcpConn, err := l.base.(*net.TCPListener).AcceptTCP()
	if err != nil {
		return nil, err
	}

	return &SBTCPConn{tcpConn}, nil
}

func (l *Listener) Close() error {
	return l.base.Close()
}

func ListenTCP(network string, addr *net.TCPAddr) (net.Listener, error) {
	listener, err := net.ListenTCP(network, addr)
	if err != nil {
		return nil, err
	}

	fdSource, err := listener.File()
	if err != nil {
		return nil, &net.OpError{Op: "listen", Err: fmt.Errorf("failed to get file descriptor: %w", err)}
	}
	defer func() { _ = fdSource.Close() }()

	return &Listener{listener}, err
}

func NewSplitbitTCPConn(conn net.Conn) *SBTCPConn {
	return &SBTCPConn{conn}
}

// DialOriginalDestination will open a connection to the original destination that the original
// connection was trying to connect to
func (c *SBTCPConn) DialOriginalDestination(dontAssumeRemote bool) (*net.TCPConn, error) {
	rAddrPtr := c.RemoteAddr().(*net.TCPAddr)
	lAddrPtr := c.LocalAddr().(*net.TCPAddr)

	remoteSocketAddress, err := tcpAddrToSocketAddr(lAddrPtr)
	if err != nil {
		return nil, &net.OpError{Op: "dial", Err: fmt.Errorf("failed to parse local socket address: %w", err)}
	}

	localIP := rAddrPtr.IP
	bindAddr := &net.TCPAddr{IP: localIP, Port: 0} // OS will pick an available port
	localSocketAddress, err := tcpAddrToSocketAddr(bindAddr)
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

	if err := unix.SetsockoptInt(int(fd), unix.SOL_IP, unix.SO_REUSEPORT, 1); err != nil {
		syscall.Close(fd)
		return nil, &net.OpError{Op: "dial", Err: fmt.Errorf("socket option SO_REUSEPORT: %w", err)}
	}

	if err := syscall.SetsockoptInt(fd, syscall.SOL_IP, syscall.IP_TRANSPARENT, 1); err != nil {
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

	if err := syscall.Connect(fd, remoteSocketAddress); err != nil {
		var errno syscall.Errno
		if !errors.As(err, &errno) || !errors.Is(err, syscall.EINPROGRESS) {
			syscall.Close(fd)
			return nil, &net.OpError{Op: "dial", Err: fmt.Errorf("socket connect: %w", err)}
		}
	}

	fdFile := os.NewFile(uintptr(fd), fmt.Sprintf("net tcp dial %s", c.LocalAddr().String()))
	defer fdFile.Close()

	remoteConn, err := net.FileConn(fdFile)
	if err != nil {
		syscall.Close(fd)
		return nil, &net.OpError{Op: "dial", Err: fmt.Errorf("fd to conn: %w", err)}
	}

	return remoteConn.(*net.TCPConn), nil
}

// tcpAddToSockerAddr will convert a TCPAddr into a Sockaddr that may be used when
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

// tcpAddrFamily will attempt to work out the address family based on the
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

func SetReadDeadline(conn io.Reader, timeout time.Duration, logger *Logger) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		err := tcpConn.SetReadDeadline(time.Now().Add(timeout))
		if err != nil {
			logger.Error("failed to set read deadline: %v", err)
			return
		}
	} else {
		logger.Warn("src is not a *net.TCPConn, skipping setting read deadline")
	}
}

func SetWriteDeadline(conn io.Writer, timeout time.Duration, logger *Logger) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		err := tcpConn.SetWriteDeadline(time.Now().Add(timeout))
		if err != nil {
			logger.Error("failed to set write deadline: %v", err)
			return
		}
	} else {
		logger.Warn("src is not a *net.TCPConn, skipping setting write deadline")
	}
}
