// Package mdns provides mDNS service discovery for MoonHub devices.
package mdns

import (
	"net"
	"time"
)

const (
	// ServiceName is the mDNS service type for MoonHub devices.
	ServiceName = "_moonhub._tcp"

	// ServiceDomain is the mDNS domain for MoonHub devices.
	ServiceDomain = "local."

	// DefaultPort is the default HTTP API port.
	DefaultPort = 18800

	// DefaultDiscoverTimeout is the default timeout for device discovery.
	DefaultDiscoverTimeout = 5 * time.Second
)

// DeviceInfo represents a discovered MoonHub device.
type DeviceInfo struct {
	// ID is the unique device identifier (e.g., "YSHU-2026-ABC").
	ID string `json:"id"`

	// Name is the friendly device name (e.g., "MoonHub-LivingRoom").
	Name string `json:"name"`

	// Version is the MoonHub software version.
	Version string `json:"version"`

	// Addr is the device's IP address.
	Addr net.IP `json:"addr"`

	// Port is the HTTP API port.
	Port int `json:"port"`

	// Hostname is the device hostname.
	Hostname string `json:"hostname,omitempty"`

	// AgentID is the MoonHub agent identity ID (e.g., "mh_abcdef1234567890").
	AgentID string `json:"agent_id,omitempty"`

	// AgentName is the agent's friendly name.
	AgentName string `json:"agent_name,omitempty"`
}

// ServerConfig holds configuration for the mDNS server.
type ServerConfig struct {
	// DeviceID is the unique device identifier.
	DeviceID string

	// Name is the friendly device name.
	Name string

	// Version is the software version.
	Version string

	// Port is the HTTP API port.
	Port int

	// Iface is the network interface to broadcast on (optional).
	Iface *net.Interface

	// AgentID is the MoonHub agent identity to advertise.
	AgentID string

	// AgentName is the agent's friendly name.
	AgentName string
}

// ClientConfig holds configuration for the mDNS client.
type ClientConfig struct {
	// Timeout is the discovery timeout.
	Timeout time.Duration

	// Interface is the network interface to search on (optional).
	Interface *net.Interface
}
