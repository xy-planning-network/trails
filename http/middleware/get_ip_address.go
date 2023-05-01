package middleware

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/xy-planning-network/trails"
)

// An ipRange is a range of IP addresses.
type ipRange struct {
	start net.IP
	end   net.IP
}

// isInRange checks whether the address is within the range.
func (r ipRange) isInRange(ipAddress net.IP) bool {
	if bytes.Compare(ipAddress, r.start) >= 0 && bytes.Compare(ipAddress, r.end) < 0 {
		return true
	}
	return false
}

// IANA defined IPv4 non-public ranges
var privateRanges = []ipRange{
	{start: net.ParseIP("10.0.0.0"), end: net.ParseIP("10.255.255.255")},
	{start: net.ParseIP("100.64.0.0"), end: net.ParseIP("100.127.255.255")},
	{start: net.ParseIP("172.16.0.0"), end: net.ParseIP("172.31.255.255")},
	{start: net.ParseIP("192.0.0.0"), end: net.ParseIP("192.0.0.255")},
	{start: net.ParseIP("192.168.0.0"), end: net.ParseIP("192.168.255.255")},
	{start: net.ParseIP("198.18.0.0"), end: net.ParseIP("198.19.255.255")},
}

// InjectIPAddress grabs the IP address in the *http.Request.Header
// and promotes it to *http.Request.Context under trails.IpAddrKey.
func InjectIPAddress() Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := GetIPAddress(r.Header)
			r = r.Clone(context.WithValue(r.Context(), trails.IpAddrKey, ip))
			h.ServeHTTP(w, r)
		})
	}
}

// GetIPAddress parses "X-Forward-For" and "X-Real-Ip" headers for the IP address
// from the request.
//
// GetIPAddress skips addresses from non-public ranges.
func GetIPAddress(hm http.Header) string {
	for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
		addresses := strings.Split(hm.Get(h), ",")
		// march from right to left until we get a public address
		// that will be the address right before our proxy.
		for i := len(addresses) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(addresses[i])
			realIP := net.ParseIP(ip)
			if !realIP.IsGlobalUnicast() || isPrivateSubnet(realIP) {
				continue
			}
			return ip
		}
	}
	return "0.0.0.0"
}

// isPrivateSubnet checks whether the IP address is in a private subnet.
//
// Only IPv4 subnets are supported.
func isPrivateSubnet(ipAddress net.IP) bool {
	if ipCheck := ipAddress.To4(); ipCheck != nil {
		for _, r := range privateRanges {
			if r.isInRange(ipAddress) {
				return true
			}
		}
	}
	return false
}
