package mdns

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/hashicorp/mdns"
)

// Client discovers MoonHub devices on the local network via mDNS.
type Client struct {
	timeout   time.Duration
	iface     *net.Interface
	entriesCh chan *mdns.ServiceEntry
}

// NewClient creates a new mDNS client for device discovery.
func NewClient(opts ...ClientConfig) *Client {
	var config ClientConfig
	if len(opts) > 0 {
		config = opts[0]
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultDiscoverTimeout
	}

	return &Client{
		timeout:   config.Timeout,
		iface:     config.Interface,
		entriesCh: make(chan *mdns.ServiceEntry, 10),
	}
}

// Discover scans the local network for MoonHub devices.
func (c *Client) Discover(ctx context.Context) ([]DeviceInfo, error) {
	// Create context with timeout
	discoverCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Start mDNS query in a goroutine
	go func() {
		params := &mdns.QueryParam{
			Service:             ServiceName,
			Domain:              ServiceDomain,
			Timeout:             c.timeout,
			Entries:             c.entriesCh,
			WantUnicastResponse: false,
			DisableIPv6:         false,
		}
		if c.iface != nil {
			params.Interface = c.iface
		}

		if err := mdns.Query(params); err != nil {
			log.Printf("mDNS query error: %v", err)
		}
	}()

	// Collect responses
	var devices []DeviceInfo
	seen := make(map[string]bool)

	for {
		select {
		case <-discoverCtx.Done():
			return devices, nil
		case entry := <-c.entriesCh:
			if entry == nil {
				continue
			}

			// Parse device info from the entry
			device := c.parseEntry(entry)
			if device == nil {
				continue
			}

			// Deduplicate by device ID
			if seen[device.ID] {
				continue
			}
			seen[device.ID] = true
			devices = append(devices, *device)
		}
	}
}

// DiscoverOne discovers a single MoonHub device by ID.
func (c *Client) DiscoverOne(ctx context.Context, deviceID string) (*DeviceInfo, error) {
	devices, err := c.Discover(ctx)
	if err != nil {
		return nil, err
	}

	for _, device := range devices {
		if device.ID == deviceID {
			return &device, nil
		}
	}

	return nil, nil
}

// parseEntry converts an mDNS entry to DeviceInfo.
func (c *Client) parseEntry(entry *mdns.ServiceEntry) *DeviceInfo {
	// Get IPv4 address
	addr := entry.AddrV4
	if addr == nil || addr.IsLoopback() || len(addr) == 0 {
		// Try IPv6 if no IPv4
		if entry.AddrV6IPAddr != nil && entry.AddrV6IPAddr.IP != nil {
			addr = entry.AddrV6IPAddr.IP
		}
	}
	if addr == nil || addr.IsLoopback() {
		return nil
	}

	return parseDeviceFromTXT(entry.InfoFields, addr, entry.Port)
}
