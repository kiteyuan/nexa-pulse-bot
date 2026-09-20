package kernel

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

// AbsoluteHTTPURL validates an absolute http(s) URL with a host and no userinfo.
// It does not resolve DNS or reject private addresses — use SafeHTTPURL for outbound fetches.
func AbsoluteHTTPURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.User != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("地址无效")
	}
	u.Fragment = ""
	return u.String(), nil
}

// PublicHTTPURL is an alias of AbsoluteHTTPURL kept for older call sites.
func PublicHTTPURL(raw string) (string, error) {
	return AbsoluteHTTPURL(raw)
}

// SafeHTTPURL validates AbsoluteHTTPURL and rejects hosts that resolve to
// loopback, private, link-local, or cloud metadata addresses.
func SafeHTTPURL(raw string) (string, error) {
	cleaned, err := AbsoluteHTTPURL(raw)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(cleaned)
	if err != nil {
		return "", errors.New("地址无效")
	}
	host := u.Hostname()
	if host == "" {
		return "", errors.New("地址无效")
	}
	if isBlockedHostName(host) {
		return "", errors.New("禁止访问内网或元数据地址")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return "", errors.New("禁止访问内网或元数据地址")
		}
		return cleaned, nil
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return "", errors.New("地址无法解析")
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return "", errors.New("禁止访问内网或元数据地址")
		}
	}
	return cleaned, nil
}

func isBlockedHostName(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	switch h {
	case "localhost", "metadata.google.internal", "metadata":
		return true
	}
	return false
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// 0.0.0.0/8, CGNAT 100.64/10, benchmarking 198.18/15
		if ip4[0] == 0 {
			return true
		}
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
		if ip4[0] == 198 && (ip4[1] == 18 || ip4[1] == 19) {
			return true
		}
		// AWS/GCP/Azure metadata
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		return false
	}
	// IPv6 ULA fc00::/7 and AWS IMDS fd00:ec2::254
	if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
		return true
	}
	return false
}
