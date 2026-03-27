package requestmeta

import (
	"net"
	"net/http"
	"strings"
)

var cloudflareCIDRs = mustParseCIDRs([]string{
	"173.245.48.0/20",
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"141.101.64.0/18",
	"108.162.192.0/18",
	"190.93.240.0/20",
	"188.114.96.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",
	"162.158.0.0/15",
	"104.16.0.0/13",
	"104.24.0.0/14",
	"172.64.0.0/13",
	"131.0.72.0/22",
	"2400:cb00::/32",
	"2606:4700::/32",
	"2803:f800::/32",
	"2405:b500::/32",
	"2405:8100::/32",
	"2a06:98c0::/29",
	"2c0f:f248::/32",
})

func ClientIP(req *http.Request) string {
	if req == nil {
		return ""
	}
	remoteIP := remoteAddrIP(req.RemoteAddr)
	if !isCloudflareRemote(remoteIP) {
		return remoteIP
	}
	if ip := parseIP(strings.TrimSpace(req.Header.Get("CF-Connecting-IP"))); ip != "" {
		return ip
	}
	if ip := parseIP(strings.TrimSpace(req.Header.Get("X-Real-IP"))); ip != "" {
		return ip
	}
	if forwarded := strings.TrimSpace(req.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		for _, part := range parts {
			ip := parseIP(strings.TrimSpace(part))
			if ip == "" || isPrivateIP(ip) {
				continue
			}
			return ip
		}
	}
	return remoteIP
}

func remoteAddrIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err == nil {
		return parseIP(host)
	}
	return parseIP(strings.TrimSpace(remoteAddr))
}

func parseIP(raw string) string {
	parsed := net.ParseIP(strings.TrimSpace(raw))
	if parsed == nil {
		return ""
	}
	return parsed.String()
}

func isCloudflareRemote(ip string) bool {
	if ip == "" {
		return false
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, cidr := range cloudflareCIDRs {
		if cidr.Contains(parsed) {
			return true
		}
	}
	return false
}

func isPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast() || parsed.IsLinkLocalMulticast()
}

func mustParseCIDRs(raw []string) []*net.IPNet {
	cidrs := make([]*net.IPNet, 0, len(raw))
	for _, item := range raw {
		_, network, err := net.ParseCIDR(item)
		if err != nil {
			panic(err)
		}
		cidrs = append(cidrs, network)
	}
	return cidrs
}
