package mdns

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/mdns"
)

// Server broadcasts MoonHub device presence via mDNS.
type Server struct {
	service *mdns.Server
	config  ServerConfig
	running bool
	stopCh  chan struct{}
}

// NewServer creates a new mDNS server instance.
func NewServer(config ServerConfig) *Server {
	if config.Port == 0 {
		config.Port = DefaultPort
	}
	return &Server{
		config: config,
		stopCh: make(chan struct{}),
	}
}

// Start begins broadcasting the MoonHub service via mDNS.
func (s *Server) Start(ctx context.Context) error {
	if s.running {
		return nil
	}

	// Create the mDNS service
	service, err := mdns.NewMDNSService(
		s.instanceName(),
		ServiceName,
		ServiceDomain,
		"",
		s.config.Port,
		nil,
		s.serviceTXTRecords(),
	)
	if err != nil {
		return fmt.Errorf("failed to create mDNS service: %w", err)
	}

	// Create the mDNS server
	server, err := mdns.NewServer(&mdns.Config{
		Zone:  service,
		Iface: s.config.Iface,
	})
	if err != nil {
		return fmt.Errorf("failed to create mDNS server: %w", err)
	}

	s.service = server
	s.running = true

	log.Printf("mDNS: broadcasting %s on port %d", s.instanceName(), s.config.Port)

	go func() {
		select {
		case <-ctx.Done():
			s.Stop()
		case <-s.stopCh:
			return
		}
	}()

	return nil
}

// Stop stops the mDNS server.
func (s *Server) Stop() {
	if !s.running {
		return
	}
	if s.service != nil {
		s.service.Shutdown()
		s.service = nil
	}
	s.running = false
	close(s.stopCh)
	log.Printf("mDNS: stopped broadcasting")
}

// instanceName returns the mDNS service instance name.
func (s *Server) instanceName() string {
	// Use hostname as instance name for uniqueness
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "moonhub"
	}
	// Sanitize hostname for mDNS compatibility
	hostname = sanitizeHostname(hostname)
	return fmt.Sprintf("MoonHub-%s (%s)", s.config.DeviceID, hostname)
}

// sanitizeHostname removes invalid characters from hostname.
func sanitizeHostname(h string) string {
	var result strings.Builder
	for _, r := range h {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// serviceTXTRecords returns TXT records for the mDNS service.
func (s *Server) serviceTXTRecords() []string {
	return []string{
		fmt.Sprintf("id=%s", s.config.DeviceID),
		fmt.Sprintf("name=%s", s.config.Name),
		fmt.Sprintf("version=%s", s.config.Version),
		fmt.Sprintf("port=%d", s.config.Port),
	}
}

// parseTXTRecords parses TXT records from mDNS response into a map.
func parseTXTRecords(txt []string) map[string]string {
	result := make(map[string]string)
	for _, record := range txt {
		parts := strings.SplitN(record, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

// parseDeviceFromTXT extracts device info from TXT records.
func parseDeviceFromTXT(txt []string, addr net.IP, port int) *DeviceInfo {
	records := parseTXTRecords(txt)
	id := records["id"]
	if id == "" {
		return nil
	}

	name := records["name"]
	if name == "" {
		name = "MoonHub Device"
	}

	version := records["version"]

	// Parse port from TXT if not provided
	if port == 0 {
		if portStr := records["port"]; portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
			}
		}
	}
	if port == 0 {
		port = DefaultPort
	}

	return &DeviceInfo{
		ID:      id,
		Name:    name,
		Version: version,
		Addr:    addr,
		Port:    port,
	}
}
