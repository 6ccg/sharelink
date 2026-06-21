package security

import (
	"context"
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"172.32.0.1", false},
		{"192.168.1.1", true},
		{"169.254.1.1", true},
		{"8.8.8.8", false},
		{"::1", true},
		{"fe80::1", true},
		{"fc00::1", true},
		{"2001:db8::1", true},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Fatalf("failed to parse IP: %s", tt.ip)
		}
		result := IsPrivateIP(ip)
		if result != tt.expected {
			t.Errorf("IsPrivateIP(%s) = %v; want %v", tt.ip, result, tt.expected)
		}
	}
}

func TestCheckSSRF(t *testing.T) {
	ctx := context.Background()

	// Test with private IP directly
	_, err := CheckSSRF(ctx, "127.0.0.1")
	if err == nil {
		t.Error("expected error for private IP 127.0.0.1, got nil")
	}

	// Test with public IP directly
	ips, err := CheckSSRF(ctx, "8.8.8.8")
	if err != nil {
		t.Errorf("unexpected error for public IP 8.8.8.8: %v", err)
	}
	if len(ips) != 1 || !ips[0].Equal(net.ParseIP("8.8.8.8")) {
		t.Errorf("unexpected resolved IPs: %v", ips)
	}

	// Test with a private hostname (localhost should resolve to 127.0.0.1)
	_, err = CheckSSRF(ctx, "localhost")
	if err == nil {
		t.Error("expected error for localhost resolution, got nil")
	}
}
