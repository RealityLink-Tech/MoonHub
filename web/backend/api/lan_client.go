package api

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// MOONHUB_TRUST_PROXY 设为 "1" 时，在反向代理后使用 X-Forwarded-For / X-Real-IP 解析客户端 IP。
// 默认关闭：仅信任 TCP 直连的 RemoteAddr，避免伪造头绕过局域网策略（云端入口预留，当前不启用）。
const envTrustProxy = "MOONHUB_TRUST_PROXY"

func trustProxyHeaders() bool {
	return os.Getenv(envTrustProxy) == "1"
}

// ClientIPFromRequest 返回用于访问策略的客户端 IP。
// 默认仅使用 RemoteAddr；开启 MOONHUB_TRUST_PROXY 后才会读取转发头。
func ClientIPFromRequest(r *http.Request) net.IP {
	if trustProxyHeaders() {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			for _, part := range strings.Split(xff, ",") {
				ip := net.ParseIP(strings.TrimSpace(part))
				if ip != nil {
					return ip
				}
			}
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			if ip := net.ParseIP(strings.TrimSpace(xri)); ip != nil {
				return ip
			}
		}
	}
	return parseRemoteAddrIP(r.RemoteAddr)
}

func parseRemoteAddrIP(remoteAddr string) net.IP {
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	// IPv6 zone (e.g. fe80::1%en0)
	if i := strings.IndexByte(host, '%'); i >= 0 {
		host = host[:i]
	}
	return net.ParseIP(host)
}

// IsLANScopeIP 判断 IP 是否属于当前阶段允许的「局域网直连」范围：回环、私网/CGNAT/ULA、链路本地（含 IPv4 APIPA）。
func IsLANScopeIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	if ip.IsPrivate() {
		return true
	}
	if ip.IsLinkLocalUnicast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// RFC 6598 CGNAT 100.64.0.0/10 (some constrained LAN / ISP shared address space)
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}
	return false
}

func requireLANClient(w http.ResponseWriter, r *http.Request) bool {
	if IsLANScopeIP(ClientIPFromRequest(r)) {
		return true
	}
	writeAuthError(w, http.StatusForbidden, "access denied")
	return false
}


// isLANOrigin checks whether an Origin header value refers to a LAN or
// loopback address. It parses the host from the URL and resolves it to an
// IP, then uses IsLANScopeIP. If the origin is empty (e.g. non-browser
// clients), it returns true so existing behaviour is preserved.
func isLANOrigin(origin string) bool {
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	// Fast-path for common textual forms.
	if host == "localhost" || host == "127.0.0.1" || host == "[::1]" || host == "0.0.0.0" {
		return true
	}
	// Try to parse as an IP directly.
	if ip := net.ParseIP(host); ip != nil {
		return IsLANScopeIP(ip)
	}
	// Resolve hostname (e.g. "my-pi.local").
	addrs, err := net.LookupHost(host)
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && IsLANScopeIP(ip) {
			return true
		}
	}
	return false
}

// writeCORSHeaders sets Access-Control-Allow-Origin to the requesting origin
// only when the origin is a LAN address. If the origin is not LAN-scoped the
// header is omitted, effectively blocking cross-origin access from public
// origins.
func writeCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if isLANOrigin(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
}
