package api

import (
	"net"
	"net/http"
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

func requireLANClientLANAPI(w http.ResponseWriter, r *http.Request) bool {
	if IsLANScopeIP(ClientIPFromRequest(r)) {
		return true
	}
	writeAuthError(w, http.StatusForbidden, "access denied")
	return false
}
