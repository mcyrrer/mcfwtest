package probe

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// UDPProber implements UDP connection probing
type UDPProber struct{}

// NewUDPProber creates a new UDP prober
func NewUDPProber() *UDPProber {
	return &UDPProber{}
}

// Probe attempts a UDP probe to the specified host and port
// UDP probing is inherently less reliable than TCP:
// - If we receive ICMP "port unreachable", the port is closed
// - If we receive a UDP response, the port is open
// - If we get no response (timeout), the port is open|filtered (ambiguous)
func (p *UDPProber) Probe(ctx context.Context, host string, port int, timeout time.Duration) Result {
	start := time.Now()

	// Resolve the host to an IP address
	ips, err := net.LookupHost(host)
	if err != nil || len(ips) == 0 {
		return Result{
			Status:  StatusError,
			Latency: time.Since(start),
			Message: fmt.Sprintf("failed to resolve host: %v", err),
		}
	}

	// Use the first IP address
	targetIP := ips[0]

	// Create UDP connection
	addr := fmt.Sprintf("%s:%d", targetIP, port)
	conn, err := net.DialTimeout("udp", addr, timeout)
	if err != nil {
		return Result{
			Status:  StatusError,
			Latency: time.Since(start),
			Message: err.Error(),
		}
	}
	defer conn.Close()

	// Set deadline for the entire operation
	deadline := time.Now().Add(timeout)
	conn.SetDeadline(deadline)

	// Send a minimal UDP probe packet
	// Many services will ignore this, but it's enough to trigger ICMP unreachable
	probe := []byte("PROBE")
	_, err = conn.Write(probe)
	if err != nil {
		return Result{
			Status:  StatusError,
			Latency: time.Since(start),
			Message: err.Error(),
		}
	}

	// Try to read response (most UDP services won't respond to random data)
	response := make([]byte, 1024)
	conn.SetReadDeadline(deadline)
	_, err = conn.Read(response)

	latency := time.Since(start)

	if err == nil {
		// Got a response - port is definitely open
		return Result{
			Status:  StatusOpen,
			Latency: latency,
			Message: "",
		}
	}

	// Check if it's a timeout or other error
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		// No response and no ICMP unreachable - ambiguous
		// We treat this as "open|filtered" which we consider "open"
		return Result{
			Status:  StatusOpen,
			Latency: latency,
			Message: "no response (open|filtered)",
		}
	}

	// Check for ICMP port unreachable (connection refused for UDP)
	if opErr, ok := err.(*net.OpError); ok {
		if syscallErr, ok := opErr.Err.(*syscall.Errno); ok {
			if *syscallErr == syscall.ECONNREFUSED {
				// ICMP port unreachable received - port is closed
				return Result{
					Status:  StatusClosed,
					Latency: latency,
					Message: "icmp port unreachable",
				}
			}
		}
	}

	// Other error - treat as open|filtered (ambiguous)
	return Result{
		Status:  StatusOpen,
		Latency: latency,
		Message: "no definitive response (open|filtered)",
	}
}

// listenForICMP attempts to listen for ICMP port unreachable messages
// This is a more sophisticated approach but requires raw sockets (root/admin privileges)
// For now, we rely on the OS kernel to deliver ICMP errors via ECONNREFUSED
func listenForICMP(ctx context.Context, targetIP string, targetPort int, timeout time.Duration) (bool, error) {
	// Create ICMP listener
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return false, err
	}
	defer c.Close()

	// Set deadline
	deadline := time.Now().Add(timeout)
	c.SetReadDeadline(deadline)

	// Read ICMP messages
	rb := make([]byte, 1500)
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		n, peer, err := c.ReadFrom(rb)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return false, nil // Timeout - no ICMP unreachable
			}
			return false, err
		}

		// Parse ICMP message
		rm, err := icmp.ParseMessage(1, rb[:n])
		if err != nil {
			continue
		}

		// Check if it's a Destination Unreachable message
		if rm.Type == ipv4.ICMPTypeDestinationUnreachable {
			// This indicates the port is unreachable (closed)
			_ = peer // peer is the source of the ICMP message
			return true, nil
		}
	}
}
