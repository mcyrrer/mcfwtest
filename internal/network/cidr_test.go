package network

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandCIDR(t *testing.T) {
	tests := []struct {
		name        string
		cidr        string
		expected    []string
		expectError bool
	}{
		{
			name:     "single host /32",
			cidr:     "192.168.1.1/32",
			expected: []string{"192.168.1.1"},
		},
		{
			name:     "point-to-point /31",
			cidr:     "192.168.1.0/31",
			expected: []string{"192.168.1.0", "192.168.1.1"},
		},
		{
			name: "small range /30",
			cidr: "192.168.1.0/30",
			expected: []string{
				"192.168.1.1",
				"192.168.1.2",
			},
		},
		{
			name: "small range /29",
			cidr: "10.0.0.0/29",
			expected: []string{
				"10.0.0.1",
				"10.0.0.2",
				"10.0.0.3",
				"10.0.0.4",
				"10.0.0.5",
				"10.0.0.6",
			},
		},
		{
			name:        "invalid CIDR",
			cidr:        "not-a-cidr",
			expectError: true,
		},
		{
			name:        "too large /15",
			cidr:        "10.0.0.0/15",
			expectError: true,
		},
		{
			name:        "too large /8",
			cidr:        "10.0.0.0/8",
			expectError: true,
		},
		{
			name:        "CIDR without mask",
			cidr:        "192.168.1.1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandCIDR(tt.cidr)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestExpandCIDR_LargeRange(t *testing.T) {
	// Test /24 to ensure it works but don't check all IPs
	result, err := ExpandCIDR("192.168.1.0/24")
	assert.NoError(t, err)
	assert.Len(t, result, 254) // 256 - 2 (network and broadcast)
	assert.Equal(t, "192.168.1.1", result[0])
	assert.Equal(t, "192.168.1.254", result[253])
}

func TestExpandCIDR_Slash16(t *testing.T) {
	// Test /16 boundary (should work, but large)
	result, err := ExpandCIDR("10.0.0.0/16")
	assert.NoError(t, err)
	assert.Len(t, result, 65534) // 2^16 - 2
}

func TestIncrementIP(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple increment",
			input:    "192.168.1.1",
			expected: "192.168.1.2",
		},
		{
			name:     "byte overflow",
			input:    "192.168.1.255",
			expected: "192.168.2.0",
		},
		{
			name:     "double byte overflow",
			input:    "192.168.255.255",
			expected: "192.169.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(t, tt.input)
			result := incrementIP(ip)
			assert.Equal(t, tt.expected, result.String())
		})
	}
}

func parseIP(t *testing.T, s string) []byte {
	t.Helper()
	ip := make([]byte, 4)
	var a, b, c, d int
	_, err := fmt.Sscanf(s, "%d.%d.%d.%d", &a, &b, &c, &d)
	if err != nil {
		t.Fatalf("failed to parse IP %q: %v", s, err)
	}
	ip[0], ip[1], ip[2], ip[3] = byte(a), byte(b), byte(c), byte(d)
	return ip
}
