package provisioning

import (
	"context"
	"strings"
	"testing"
)

// MockConfigStore is a mock implementation of ConfigStore for testing
type MockConfigStore struct {
	data map[string]interface{}
}

func NewMockConfigStore() *MockConfigStore {
	return &MockConfigStore{
		data: make(map[string]interface{}),
	}
}

func (m *MockConfigStore) Get(key string) interface{} {
	return m.data[key]
}

func (m *MockConfigStore) Set(key string, value interface{}) {
	m.data[key] = value
}

func (m *MockConfigStore) Delete(key string) {
	delete(m.data, key)
}

func (m *MockConfigStore) Save() error {
	return nil
}

// MockCommandRunner returns mock results for commands
func MockCommandRunner(ctx context.Context, args ...string) CommandResult {
	if len(args) == 0 {
		return CommandResult{OK: false, Stderr: "no command"}
	}

	cmd := args[0]
	switch cmd {
	case "nmcli":
		if len(args) > 1 && args[1] == "--version" {
			return CommandResult{OK: true, Stdout: "nmcli tool, version 1.22.10"}
		}
		// nmcli -t -f STATE networking
		if len(args) >= 5 && args[1] == "-t" && args[2] == "-f" && args[3] == "STATE" && args[4] == "networking" {
			return CommandResult{OK: true, Stdout: "connected"}
		}
		// nmcli device wifi connect <ssid> ...
		if len(args) >= 5 && args[1] == "device" && args[2] == "wifi" && args[3] == "connect" {
			return CommandResult{OK: true}
		}
		// nmcli -t -f GENERAL.CONNECTION device show <iface> (post-connect autoconnect)
		if len(args) >= 7 && args[1] == "-t" && args[2] == "-f" && args[3] == "GENERAL.CONNECTION" &&
			args[4] == "device" && args[5] == "show" {
			return CommandResult{OK: true, Stdout: "SimulatedWifi\n"}
		}
		// nmcli -t -f GENERAL.STATE,GENERAL.CONNECTION,... device show <iface>
		if len(args) >= 7 && args[1] == "-t" && args[2] == "-f" &&
			strings.Contains(args[3], "GENERAL.STATE") &&
			args[4] == "device" && args[5] == "show" {
			return CommandResult{
				OK: true,
				Stdout: "GENERAL.STATE:connected\n" +
					"GENERAL.CONNECTION:SimulatedWifi\n" +
					"IP4.ADDRESS[0]:192.168.1.42/24\n" +
					"IP4.GATEWAY:192.168.1.1\n",
			}
		}
		// Handle: nmcli -t -f SSID,SIGNAL,SECURITY,IN-USE device wifi list --rescan yes
		if len(args) > 6 && args[1] == "-t" && args[6] == "list" {
			return CommandResult{
				OK:     true,
				Stdout: "MyNetwork:80:wpa2:*\nGuestNetwork:50:open:\nHiddenNet:90:wpa3:",
			}
		}
		// Handle: nmcli -t -f NAME connection show id <profile>
		if len(args) > 5 && args[4] == "show" && args[5] == "id" {
			return CommandResult{OK: true, Stdout: "MoonHub Hotspot"}
		}
		if len(args) > 1 && args[1] == "connection" {
			return CommandResult{OK: true}
		}
		return CommandResult{OK: true}
	case "systemctl":
		return CommandResult{OK: true, Stdout: "active"}
	case "ping":
		return CommandResult{OK: true}
	case "getent":
		return CommandResult{OK: true, Stdout: "1.2.3.4 api.openai.com"}
	default:
		return CommandResult{OK: false, Stderr: "command not found"}
	}
}

func TestNewDeviceManager(t *testing.T) {
	config := NewMockConfigStore()
	dm := NewDeviceManager(config, DeviceManagerOptions{
		CommandRunner: MockCommandRunner,
	})

	if dm == nil {
		t.Fatal("Expected DeviceManager to be created")
	}
	if dm.interfaceName() != DefaultNetworkInterface {
		t.Errorf("Expected default interface %s, got %s", DefaultNetworkInterface, dm.interfaceName())
	}
}

func TestNormalizeMode(t *testing.T) {
	tests := []struct {
		input    string
		expected DeviceMode
	}{
		{"provisioning", ModeProvisioning},
		{"connecting", ModeConnecting},
		{"onboarding", ModeOnboarding},
		{"ready", ModeReady},
		{"maintenance", ModeMaintenance},
		{"error", ModeError},
		{"unknown", ModeProvisioning},
		{"", ModeProvisioning},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeMode(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeMode(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeHotspotSuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"moonhub-device", "VICE"}, // Takes last 4 alphanumeric chars, uppercased
		{"test123", "T123"},        // Takes last 4 alphanumeric chars
		{"ab", "NODE"},             // Too short, uses default
		{"my-awesome-device", "VICE"},
		{"Device-123", "E123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SanitizeHotspotSuffix(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeHotspotSuffix(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDefaultHotspotSSID(t *testing.T) {
	ssid := DefaultHotspotSSID("test-device")
	expected := "MoonHub-VICE"
	if ssid != expected {
		t.Errorf("DefaultHotspotSSID() = %q, want %q", ssid, expected)
	}
}

func TestGetStatus(t *testing.T) {
	config := NewMockConfigStore()
	dm := NewDeviceManager(config, DeviceManagerOptions{
		CommandRunner: MockCommandRunner,
	})

	ctx := context.Background()
	status, err := dm.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if status.Hostname == "" {
		t.Error("Expected hostname to be set")
	}
	if status.Mode == "" {
		t.Error("Expected mode to be set")
	}
	_ = ctx // Use ctx to avoid unused variable error
}

func TestListWifiNetworks(t *testing.T) {
	config := NewMockConfigStore()
	dm := NewDeviceManager(config, DeviceManagerOptions{
		CommandRunner: MockCommandRunner,
	})

	ctx := context.Background()
	networks, err := dm.ListWifiNetworks(ctx)
	if err != nil {
		t.Fatalf("ListWifiNetworks() error = %v", err)
	}

	if len(networks) != 3 {
		t.Fatalf("Expected 3 networks, got %d", len(networks))
	}

	// Check first network
	if networks[0].SSID != "MyNetwork" {
		t.Errorf("Expected first SSID 'MyNetwork', got %q", networks[0].SSID)
	}
	if networks[0].Signal != 80 {
		t.Errorf("Expected signal 80, got %d", networks[0].Signal)
	}
	_ = ctx // Use ctx to avoid unused variable error
}

func TestEventBroadcaster(t *testing.T) {
	b := NewEventBroadcaster(10)

	// Test subscribe
	ch := b.Subscribe()
	if ch == nil {
		t.Fatal("Expected non-nil channel")
	}
	defer b.Unsubscribe(ch)

	// Test emit
	event := b.Emit(EventTypeStatus, EventLevelInfo, "Test message")
	if event == nil {
		t.Fatal("Expected non-nil event")
	}
	if event.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got %q", event.Message)
	}

	// Test recent events
	recent := b.GetRecentEvents()
	if len(recent) != 1 {
		t.Errorf("Expected 1 recent event, got %d", len(recent))
	}
}

func TestConfigStore(t *testing.T) {
	store := NewMockConfigStore()

	// Test Set and Get
	store.Set("test.key", "test-value")
	val := store.Get("test.key")
	if val != "test-value" {
		t.Errorf("Expected 'test-value', got %v", val)
	}

	// Test Delete
	store.Delete("test.key")
	val = store.Get("test.key")
	if val != nil {
		t.Errorf("Expected nil after delete, got %v", val)
	}
}

func TestDeriveRuntimeState(t *testing.T) {
	tests := []struct {
		name          string
		mode          DeviceMode
		network       DeviceNetworkSummary
		restart       bool
		services      DeviceServicesStatus
		expectedPhase DeviceRuntimePhase
	}{
		{
			name:          "ready state",
			mode:          ModeReady,
			network:       DeviceNetworkSummary{Connected: true, InternetReachable: true},
			restart:       false,
			services:      DeviceServicesStatus{NmcliAvailable: true, NetworkManagerAvailable: true},
			expectedPhase: PhaseReady,
		},
		{
			name:          "provisioning state",
			mode:          ModeProvisioning,
			network:       DeviceNetworkSummary{APEnabled: true},
			restart:       false,
			services:      DeviceServicesStatus{NmcliAvailable: true, NetworkManagerAvailable: true},
			expectedPhase: PhaseProvisioning,
		},
		{
			name:          "error state",
			mode:          ModeError,
			network:       DeviceNetworkSummary{},
			restart:       false,
			services:      DeviceServicesStatus{NmcliAvailable: true, NetworkManagerAvailable: true},
			expectedPhase: PhaseError,
		},
		{
			name:          "restarting state",
			mode:          ModeReady,
			network:       DeviceNetworkSummary{Connected: true},
			restart:       true,
			services:      DeviceServicesStatus{NmcliAvailable: true, NetworkManagerAvailable: true},
			expectedPhase: PhaseRestarting,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveRuntimeState(tt.mode, tt.network, tt.restart, tt.services)
			if result.Phase != tt.expectedPhase {
				t.Errorf("Expected phase %q, got %q", tt.expectedPhase, result.Phase)
			}
		})
	}
}

// TestConnectToWiFi_SimulatedSuccess runs a functional provisioning flow against MockCommandRunner (no real nmcli).
func TestConnectToWiFi_SimulatedSuccess(t *testing.T) {
	config := NewMockConfigStore()
	dm := NewDeviceManager(config, DeviceManagerOptions{
		Hostname:      "testhub",
		CommandRunner: MockCommandRunner,
	})
	ctx := context.Background()
	err := dm.ConnectToWiFi(ctx, "MyNetwork", "hunter2", false)
	if err != nil {
		t.Fatalf("ConnectToWiFi: %v", err)
	}
	if config.Get(KeyNetworkProvisioned) != true {
		t.Fatalf("KeyNetworkProvisioned want true, got %v", config.Get(KeyNetworkProvisioned))
	}
	if config.Get(KeyLastSSID) != "MyNetwork" {
		t.Fatalf("KeyLastSSID = %v", config.Get(KeyLastSSID))
	}
	if config.Get(KeyDeviceMode) != string(ModeOnboarding) {
		t.Fatalf("KeyDeviceMode = %v, want %q", config.Get(KeyDeviceMode), ModeOnboarding)
	}
}

func TestConnectToWiFi_SimulatedHiddenSSID(t *testing.T) {
	config := NewMockConfigStore()
	dm := NewDeviceManager(config, DeviceManagerOptions{
		CommandRunner: MockCommandRunner,
	})
	ctx := context.Background()
	err := dm.ConnectToWiFi(ctx, "HiddenNet", "pw", true)
	if err != nil {
		t.Fatalf("ConnectToWiFi hidden: %v", err)
	}
	if config.Get(KeyLastSSID) != "HiddenNet" {
		t.Fatalf("KeyLastSSID = %v", config.Get(KeyLastSSID))
	}
}

func TestConnectToWiFi_SimulatedNoInternet(t *testing.T) {
	config := NewMockConfigStore()
	runner := func(ctx context.Context, args ...string) CommandResult {
		if len(args) > 0 && args[0] == "ping" {
			return CommandResult{OK: false, Stderr: "Network unreachable"}
		}
		return MockCommandRunner(ctx, args...)
	}
	dm := NewDeviceManager(config, DeviceManagerOptions{CommandRunner: runner})
	ctx := context.Background()
	err := dm.ConnectToWiFi(ctx, "MyNetwork", "", false)
	if err == nil {
		t.Fatal("expected error when post-connect ping fails")
	}
}

func TestConnectToWiFi_EmptySSID(t *testing.T) {
	dm := NewDeviceManager(NewMockConfigStore(), DeviceManagerOptions{
		CommandRunner: MockCommandRunner,
	})
	err := dm.ConnectToWiFi(context.Background(), "   ", "", false)
	if err == nil {
		t.Fatal("expected error for empty SSID")
	}
}
