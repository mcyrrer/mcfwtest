package network

import (
	"fmt"
	"net"
)

// ResolveHost resolves a hostname to a list of IP addresses
// Returns the IP as-is if it's already an IP address
func ResolveHost(host string) ([]string, error) {
	// Check if it's already an IP address
	if ip := net.ParseIP(host); ip != nil {
		return []string{host}, nil
	}

	// Attempt DNS lookup
	ips, err := net.LookupHost(host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve hostname %q: %w", host, err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP addresses found for hostname %q", host)
	}

	return ips, nil
}
