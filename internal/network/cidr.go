package network

import (
	"fmt"
	"net"
)

// ExpandCIDR expands a CIDR range into individual usable host IPs
// Returns an error if the CIDR is invalid or larger than /16
func ExpandCIDR(cidr string) ([]string, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}

	// Check prefix length to prevent resource exhaustion
	ones, _ := ipNet.Mask.Size()
	if ones < 16 {
		return nil, fmt.Errorf("CIDR range %q is too large (prefix /%d), maximum allowed is /16", cidr, ones)
	}

	var ips []string

	// Special case for /32 (single host)
	if ones == 32 {
		return []string{ip.String()}, nil
	}

	// Special case for /31 (point-to-point link, both IPs are usable)
	if ones == 31 {
		return []string{
			ip.String(),
			incrementIP(ip).String(),
		}, nil
	}

	// For other ranges, skip network and broadcast addresses
	// Start from network address + 1
	currentIP := incrementIP(ipNet.IP)

	// Calculate broadcast address
	broadcastIP := make(net.IP, len(ipNet.IP))
	copy(broadcastIP, ipNet.IP)
	for i := range broadcastIP {
		broadcastIP[i] |= ^ipNet.Mask[i]
	}

	// Collect all usable IPs (excluding broadcast)
	for !currentIP.Equal(broadcastIP) && ipNet.Contains(currentIP) {
		ips = append(ips, currentIP.String())
		currentIP = incrementIP(currentIP)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("CIDR range %q contains no usable host addresses", cidr)
	}

	return ips, nil
}

// incrementIP returns the next IP address
func incrementIP(ip net.IP) net.IP {
	// Make a copy to avoid modifying the original
	result := make(net.IP, len(ip))
	copy(result, ip)

	// Increment from the rightmost byte
	for i := len(result) - 1; i >= 0; i-- {
		result[i]++
		if result[i] != 0 {
			break
		}
	}

	return result
}
