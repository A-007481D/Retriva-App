// Package security provides protections against SSRF, directory traversal, and malicious URLs.
package security

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

var blockedCIDRs = []string{
	"10.0.0.0/8",      // RFC1918
	"172.16.0.0/12",   // RFC1918
	"192.168.0.0/16",  // RFC1918
	"127.0.0.0/8",     // Loopback
	"0.0.0.0/8",       // Current network
	"169.254.0.0/16",  // Link-local (Cloud Metadata endpoints)
	"100.64.0.0/10",   // Shared address space (RFC6598)
	"192.0.0.0/24",    // IETF Protocol Assignments
	"192.0.2.0/24",    // TEST-NET-1
	"198.18.0.0/15",   // Network Interconnect Device Benchmark Testing
	"198.51.100.0/24", // TEST-NET-2
	"203.0.113.0/24",  // TEST-NET-3
	"224.0.0.0/4",     // Multicast
	"240.0.0.0/4",     // Reserved
	"::1/128",         // Loopback IPv6
	"fc00::/7",        // Unique local address IPv6
	"fe80::/10",       // Link-local IPv6
}

var parsedBlockedNets []*net.IPNet

func init() {
	for _, cidr := range blockedCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("invalid blocked CIDR %q: %v", cidr, err))
		}
		parsedBlockedNets = append(parsedBlockedNets, ipNet)
	}
}

var blockedHostnames = []string{
	"localhost",
	"metadata.google.internal",
	"169.254.169.254",
	"instance-data",
}

// CheckSSRF verifies that the hostname resolves to a public IP address
// and is not in the blocked hostnames list.
func CheckSSRF(ctx context.Context, rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	hostname := u.Hostname()
	if hostname == "" {
		return fmt.Errorf("missing hostname")
	}

	hostnameLower := strings.ToLower(hostname)
	for _, blocked := range blockedHostnames {
		if hostnameLower == blocked {
			return fmt.Errorf("blocked hostname: %s", hostname)
		}
	}

	// Resolve the IPs. Use a timeout to prevent DNS tarpits.
	resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIP(resolveCtx, "ip", hostname)
	if err != nil {
		return fmt.Errorf("dns lookup failed: %w", err)
	}

	if len(ips) == 0 {
		return fmt.Errorf("no IP addresses found for hostname")
	}

	for _, ip := range ips {
		if !isPublicIP(ip) {
			return fmt.Errorf("resolves to non-public IP: %s", ip.String())
		}
	}

	return nil
}

// isPublicIP checks if the IP is globally routable and not in any blocked CIDR.
// Note: Go 1.18+ has ip.IsPrivate() and ip.IsLoopback(), but we use our explicit
// CIDR list for comprehensive coverage including cloud metadata endpoints.
func isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return false
	}
	for _, net := range parsedBlockedNets {
		if net.Contains(ip) {
			return false
		}
	}
	return true
}
