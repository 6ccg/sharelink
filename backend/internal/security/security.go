package security

import (
	"context"
	"fmt"
	"net"
)

var privateIPBlocks []*net.IPNet

// AllowLoopback allows loopback (127.0.0.1) addresses for testing purposes
var AllowLoopback bool

func init() {
	for _, cidr := range []string{
		"0.0.0.0/8",       // current network
		"10.0.0.0/8",      // private
		"100.64.0.0/10",   // carrier-grade NAT
		"127.0.0.0/8",     // loopback
		"169.254.0.0/16",  // link-local
		"172.16.0.0/12",   // private
		"192.0.0.0/24",    // IETF protocol assignments
		"192.0.2.0/24",    // documentation
		"192.168.0.0/16",  // private
		"198.18.0.0/15",   // benchmarking
		"198.51.100.0/24", // documentation
		"203.0.113.0/24",  // documentation
		"224.0.0.0/4",     // multicast
		"240.0.0.0/4",     // reserved
		"::/128",          // unspecified
		"::1/128",         // IPv6 loopback
		"64:ff9b::/96",    // IPv4/IPv6 translation
		"100::/64",        // discard-only
		"2001::/23",       // IETF protocol assignments
		"2001:db8::/32",   // documentation
		"fc00::/7",        // IPv6 unique local
		"fe80::/10",       // IPv6 link-local
		"ff00::/8",        // IPv6 multicast
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err == nil {
			privateIPBlocks = append(privateIPBlocks, block)
		}
	}
}

// IsPrivateIP returns true if the given IP address is in a private, loopback, or reserved range.
func IsPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return !AllowLoopback
	}
	if ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			if AllowLoopback && ip.IsLoopback() {
				continue
			}
			return true
		}
	}
	return false
}

// CheckSSRF resolves the given host and checks if any resolved IP is a private/reserved IP.
// If any IP is private, it returns an error. Otherwise, it returns the resolved IPs.
func CheckSSRF(ctx context.Context, host string) ([]net.IP, error) {
	// Check if host is already an IP address
	if ip := net.ParseIP(host); ip != nil {
		if IsPrivateIP(ip) {
			return nil, fmt.Errorf("private IP address blocked: %s", ip.String())
		}
		return []net.IP{ip}, nil
	}

	// Resolve domain
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host %s: %w", host, err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP address found for host: %s", host)
	}

	// Verify all resolved IPs
	for _, ip := range ips {
		if IsPrivateIP(ip) {
			return nil, fmt.Errorf("SSRF blocked: host %s resolves to private IP %s", host, ip.String())
		}
	}

	return ips, nil
}
