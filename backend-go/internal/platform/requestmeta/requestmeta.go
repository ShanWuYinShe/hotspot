package requestmeta

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
)

var defaultTrustedProxyCIDRs = mustParseCIDRs([]string{
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

var trustedProxyCIDRs atomic.Value

func init() {
	trustedProxyCIDRs.Store(cloneCIDRs(defaultTrustedProxyCIDRs))
}

func InitTrustedProxies(raw string) error {
	cidrs := cloneCIDRs(defaultTrustedProxyCIDRs)
	extra, err := parseCIDRList(raw)
	if err != nil {
		return err
	}
	cidrs = append(cidrs, extra...)
	trustedProxyCIDRs.Store(cidrs)
	return nil
}

func ClientIP(req *http.Request) string {
	if req == nil {
		return ""
	}
	remoteIP, remoteParsed := remoteAddrIP(req.RemoteAddr)
	if remoteIP == "" || !isTrustedProxyRemote(remoteParsed) {
		return remoteIP
	}
	if ip := headerIP(req.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if ip := headerIP(req.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if ip := forwardedIP(req.Header.Get("X-Forwarded-For")); ip != "" {
		return ip
	}
	return remoteIP
}

func ClientCountry(req *http.Request) string {
	if !trustedProxyRequest(req) {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(req.Header.Get("CF-IPCountry")))
}

func RequestRay(req *http.Request) string {
	if !trustedProxyRequest(req) {
		return ""
	}
	return strings.TrimSpace(req.Header.Get("CF-Ray"))
}

func ForwardedProto(req *http.Request) string {
	if !trustedProxyRequest(req) {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(req.Header.Get("X-Forwarded-Proto")))
}

func trustedProxyRequest(req *http.Request) bool {
	if req == nil {
		return false
	}
	_, remoteParsed := remoteAddrIP(req.RemoteAddr)
	return isTrustedProxyRemote(remoteParsed)
}

func remoteAddrIP(remoteAddr string) (string, net.IP) {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err == nil {
		return normalizedIP(host)
	}
	return normalizedIP(remoteAddr)
}

func headerIP(raw string) string {
	ip, _ := normalizedIP(raw)
	return ip
}

// forwardedIP 从 X-Forwarded-For 解析客户端 IP。XFF 最左侧是客户端自报
// 内容（可伪造：攻击者直发 XFF: 1.2.3.4，代理追加真实 IP 后变
// "1.2.3.4, real"），因此必须**从右往左**遍历：跳过可信代理网段，取第一
// 个不可信地址。此前取最左侧非私网 IP，伪造的左端可绕过 IP 限流并污染
// 会话审计。
func forwardedIP(raw string) string {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	for i := len(parts) - 1; i >= 0; i-- {
		ip, parsed := normalizedIP(parts[i])
		if ip == "" || isPrivateIP(parsed) {
			continue
		}
		if isTrustedProxyRemote(parsed) {
			continue
		}
		return ip
	}
	return ""
}

func normalizedIP(raw string) (string, net.IP) {
	parsed := net.ParseIP(strings.TrimSpace(raw))
	if parsed == nil {
		return "", nil
	}
	return parsed.String(), parsed
}

func isTrustedProxyRemote(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, cidr := range currentTrustedProxyCIDRs() {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

func currentTrustedProxyCIDRs() []*net.IPNet {
	cidrs, _ := trustedProxyCIDRs.Load().([]*net.IPNet)
	return cidrs
}

func mustParseCIDRs(raw []string) []*net.IPNet {
	cidrs, err := parseCIDRs(raw)
	if err != nil {
		panic(err)
	}
	return cidrs
}

func parseCIDRList(raw string) ([]*net.IPNet, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	return parseCIDRs(parts)
}

func parseCIDRs(raw []string) ([]*net.IPNet, error) {
	cidrs := make([]*net.IPNet, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		_, network, err := net.ParseCIDR(item)
		if err != nil {
			return nil, fmt.Errorf("parse trusted proxy cidr %q: %w", item, err)
		}
		cidrs = append(cidrs, network)
	}
	return cidrs, nil
}

func cloneCIDRs(src []*net.IPNet) []*net.IPNet {
	if len(src) == 0 {
		return nil
	}
	cloned := make([]*net.IPNet, len(src))
	copy(cloned, src)
	return cloned
}
