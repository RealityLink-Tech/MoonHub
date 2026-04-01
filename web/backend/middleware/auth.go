package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
)

// BearerTokenAuth returns middleware that validates Bearer tokens against DeviceStore.
func BearerTokenAuth(deviceStore *devices.DeviceStore, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if len(auth) < 8 || auth[:7] != "Bearer " {
			writeAPIError(w, http.StatusUnauthorized, "missing or invalid token", "TOKEN_INVALID")
			return
		}
		token := strings.TrimSpace(auth[7:])

		validation := deviceStore.ValidateToken(token)
		if !validation.Valid {
			writeAPIError(w, http.StatusUnauthorized, "invalid or expired token", "TOKEN_INVALID")
			return
		}
		_ = deviceStore.UpdateLastSeen(validation.DeviceID, clientIP(r))
		next.ServeHTTP(w, r)
	})
}

func writeAPIError(w http.ResponseWriter, status int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error": map[string]string{
			"message": message,
			"code":    code,
		},
	})
}

// clientIP extracts the client IP from the request, stripping the port.
func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return host
}
