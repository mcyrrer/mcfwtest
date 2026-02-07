package probe

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTCPProber_Open(t *testing.T) {
	// Start a local TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	// Get the port
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	// Create prober
	prober := NewTCPProber()

	// Test
	ctx := context.Background()
	result := prober.Probe(ctx, "127.0.0.1", port, 2*time.Second)

	assert.Equal(t, StatusOpen, result.Status)
	assert.Greater(t, result.Latency, time.Duration(0))
	assert.Empty(t, result.Message)
}

func TestTCPProber_Closed(t *testing.T) {
	// Use a port that's unlikely to be open
	port := 54321

	// Create prober
	prober := NewTCPProber()

	// Test
	ctx := context.Background()
	result := prober.Probe(ctx, "127.0.0.1", port, 2*time.Second)

	assert.Equal(t, StatusClosed, result.Status)
	assert.Greater(t, result.Latency, time.Duration(0))
	assert.Contains(t, result.Message, "refused")
}

func TestTCPProber_Timeout(t *testing.T) {
	// Use a non-routable IP to trigger timeout (TEST-NET-1 from RFC 5737)
	host := "192.0.2.1"
	port := 80

	// Create prober
	prober := NewTCPProber()

	// Test with very short timeout
	ctx := context.Background()
	result := prober.Probe(ctx, host, port, 100*time.Millisecond)

	assert.Equal(t, StatusClosed, result.Status)
	assert.Contains(t, result.Message, "timeout")
}

func TestTCPProber_ContextCancellation(t *testing.T) {
	// Use a non-routable IP
	host := "192.0.2.1"
	port := 80

	// Create prober
	prober := NewTCPProber()

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	// Test
	result := prober.Probe(ctx, host, port, 2*time.Second)

	// Should fail due to context cancellation
	assert.NotEqual(t, StatusOpen, result.Status)
}

func TestIsConnectionRefused(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() error
		expected bool
	}{
		{
			name: "connection refused",
			setup: func() error {
				// Try to connect to a port that's definitely closed
				conn, err := net.Dial("tcp", "127.0.0.1:54321")
				if conn != nil {
					conn.Close()
				}
				return err
			},
			expected: true,
		},
		{
			name: "nil error",
			setup: func() error {
				return nil
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setup()
			result := isConnectionRefused(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTimeout(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() error
		expected bool
	}{
		{
			name: "timeout error",
			setup: func() error {
				dialer := &net.Dialer{
					Timeout: 1 * time.Millisecond,
				}
				conn, err := dialer.Dial("tcp", "192.0.2.1:80")
				if conn != nil {
					conn.Close()
				}
				return err
			},
			expected: true,
		},
		{
			name: "nil error",
			setup: func() error {
				return nil
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setup()
			result := isTimeout(err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
