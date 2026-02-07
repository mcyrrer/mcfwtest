package probe

import (
	"context"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"
)

// TCPProber implements TCP connection probing
type TCPProber struct{}

// NewTCPProber creates a new TCP prober
func NewTCPProber() *TCPProber {
	return &TCPProber{}
}

// Probe attempts a TCP connection to the specified host and port
func (p *TCPProber) Probe(ctx context.Context, host string, port int, timeout time.Duration) Result {
	start := time.Now()

	// Create a dialer with the specified timeout
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := dialer.DialContext(ctx, "tcp", address)

	latency := time.Since(start)

	if err == nil {
		// Connection succeeded
		conn.Close()
		return Result{
			Status:  StatusOpen,
			Latency: latency,
			Message: "",
		}
	}

	// Analyze the error to determine if port is closed or filtered
	if isConnectionRefused(err) {
		return Result{
			Status:  StatusClosed,
			Latency: latency,
			Message: "connection refused",
		}
	}

	if isTimeout(err) {
		return Result{
			Status:  StatusClosed,
			Latency: latency,
			Message: fmt.Sprintf("timeout %v", timeout),
		}
	}

	// Other errors (network unreachable, no route to host, etc.)
	return Result{
		Status:  StatusError,
		Latency: latency,
		Message: err.Error(),
	}
}

// isConnectionRefused checks if the error is a connection refused error
func isConnectionRefused(err error) bool {
	if err == nil {
		return false
	}

	// Check for syscall.ECONNREFUSED
	if opErr, ok := err.(*net.OpError); ok {
		if syscallErr, ok := opErr.Err.(*syscall.Errno); ok {
			return *syscallErr == syscall.ECONNREFUSED
		}
	}

	// Fallback to string matching
	return strings.Contains(err.Error(), "connection refused")
}

// isTimeout checks if the error is a timeout error
func isTimeout(err error) bool {
	if err == nil {
		return false
	}

	// Check for net.Error with Timeout() == true
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}

	// Fallback to string matching
	return strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "i/o timeout")
}
