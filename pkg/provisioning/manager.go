package provisioning

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// ConfigStore provides an interface for reading and writing configuration values.
type ConfigStore interface {
	Get(key string) interface{}
	Set(key string, value interface{})
	Delete(key string)
	Save() error
}

// DeviceManagerOptions contains configuration options for the device manager.
type DeviceManagerOptions struct {
	Hostname           string
	NetworkInterface   string
	AllowSystemControl bool
	CommandRunner      CommandRunner
}

// DeviceManager handles device provisioning and network management.
type DeviceManager struct {
	config     ConfigStore
	options    DeviceManagerOptions
	network    *NetworkOperator
	events     *EventBroadcaster
	mu         sync.RWMutex
	recoveryMu sync.Mutex
	running    bool
	cancel     context.CancelFunc
}

// NewDeviceManager creates a new device manager instance.
func NewDeviceManager(config ConfigStore, opts ...DeviceManagerOptions) *DeviceManager {
	var options DeviceManagerOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	if options.Hostname == "" {
		options.Hostname = DefaultHostname()
	}
	if options.NetworkInterface == "" {
		options.NetworkInterface = DefaultNetworkInterface
	}

	dm := &DeviceManager{
		config:  config,
		options: options,
		events:  NewEventBroadcaster(25),
	}

	dm.network = NewNetworkOperator(options.NetworkInterface, options.CommandRunner)

	return dm
}

// Start begins the device manager background operations.
func (dm *DeviceManager) Start(ctx context.Context) {
	dm.mu.Lock()
	if dm.running {
		dm.mu.Unlock()
		return
	}
	dm.running = true
	dm.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	dm.cancel = cancel

	// Emit startup event
	status, _ := dm.GetStatus()
	dm.events.EmitWithStatus(EventTypeSystem, EventLevelInfo, "Device manager started", status)

	// Start monitoring loop
	go dm.monitorLoop(ctx)
}

// Stop halts the device manager.
func (dm *DeviceManager) Stop() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if !dm.running {
		return
	}

	dm.running = false
	if dm.cancel != nil {
		dm.cancel()
	}
}

// GetEvents returns the event broadcaster for SSE subscriptions.
func (dm *DeviceManager) GetEvents() *EventBroadcaster {
	return dm.events
}

// GetStatus returns the complete device status.
func (dm *DeviceManager) GetStatus() (DeviceStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nmcliOK, nmServiceOK := dm.network.CheckServices(ctx)
	diagnostics := dm.network.DiagnoseNetwork(ctx)

	mode := NormalizeMode(getString(dm.config, KeyDeviceMode))
	services := DeviceServicesStatus{
		NmcliAvailable:          nmcliOK,
		NetworkManagerAvailable: nmServiceOK,
	}

	network := DeviceNetworkSummary{
		Provisioned:       getBool(dm.config, KeyNetworkProvisioned),
		Connected:         diagnostics.Connected,
		LastSSID:          getString(dm.config, KeyLastSSID),
		HotspotSSID:       dm.hotspotSSID(),
		HotspotProfile:    dm.hotspotProfile(),
		InterfaceName:     dm.interfaceName(),
		APEnabled:         getBool(dm.config, KeyAPEnabled),
		ActiveConnection:  diagnostics.ActiveConnection,
		IPv4Address:       diagnostics.IPv4Address,
		Gateway:           diagnostics.Gateway,
		InternetReachable: diagnostics.InternetReachable,
		LastError:         getString(dm.config, KeyLastError),
	}

	restartPending := getBool(dm.config, KeyRestartPending)
	runtime := deriveRuntimeState(mode, network, restartPending, services)
	recovery := dm.getRecoveryStatus()

	return DeviceStatus{
		Mode:              mode,
		Hostname:          dm.hostname(),
		Network:           network,
		Runtime:           runtime,
		Recovery:          recovery,
		RestartPending:    restartPending,
		LastRestartReason: getString(dm.config, KeyRestartReason),
		Services:          services,
	}, nil
}

// ListWifiNetworks scans for available WiFi networks.
func (dm *DeviceManager) ListWifiNetworks(ctx context.Context) ([]WifiNetwork, error) {
	return dm.network.ScanWifiNetworks(ctx)
}

// ListSavedNetworks returns saved WiFi profiles.
func (dm *DeviceManager) ListSavedNetworks(ctx context.Context) ([]SavedNetworkProfile, error) {
	return dm.network.ListSavedNetworks(ctx, dm.hotspotProfile())
}

// ConnectToWiFi connects to a WiFi network.
func (dm *DeviceManager) ConnectToWiFi(ctx context.Context, ssid, password string, hidden bool) error {
	ssid = strings.TrimSpace(ssid)
	if ssid == "" {
		return fmt.Errorf("WiFi SSID is required")
	}

	// Emit action started event with status
	status, _ := dm.GetStatus()
	dm.events.EmitAction("network.connect", ActionPhaseStarted, EventLevelInfo,
		fmt.Sprintf("Connecting to WiFi %s", ssid),
		WithDetails(map[string]interface{}{"ssid": ssid, "hidden": hidden}),
		WithStatus(&status))

	dm.setMode(ModeConnecting)

	err := dm.network.ConnectToWiFi(ctx, ssid, password, hidden)
	if err != nil {
		dm.rollbackToProvisioning(ctx, err.Error())
		status, _ := dm.GetStatus()
		dm.events.EmitAction("network.connect", ActionPhaseFailed, EventLevelError,
			err.Error(),
			WithDetails(map[string]interface{}{"ssid": ssid}),
			WithStatus(&status))
		return err
	}

	// Verify internet connectivity
	diagnostics := dm.network.DiagnoseNetwork(ctx)
	if !diagnostics.InternetReachable {
		err := fmt.Errorf("connected to WiFi but internet access is not available")
		dm.rollbackToProvisioning(ctx, err.Error())
		status, _ := dm.GetStatus()
		dm.events.EmitAction("network.connect", ActionPhaseFailed, EventLevelError,
			err.Error(),
			WithDetails(map[string]interface{}{"ssid": ssid}),
			WithStatus(&status))
		return err
	}

	// Update state
	dm.config.Set(KeyNetworkProvisioned, true)
	dm.config.Set(KeyLastSSID, ssid)
	dm.config.Set(KeyAPEnabled, false)
	dm.config.Set(KeyDeviceMode, string(ModeOnboarding))
	dm.config.Delete(KeyLastError)

	status, _ = dm.GetStatus()
	dm.events.EmitAction("network.connect", ActionPhaseCompleted, EventLevelSuccess,
		fmt.Sprintf("Connected to WiFi %s", ssid),
		WithDetails(map[string]interface{}{"ssid": ssid}),
		WithStatus(&status))

	return nil
}

// ForgetNetwork removes a saved network profile.
func (dm *DeviceManager) ForgetNetwork(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("network name is required")
	}

	dm.events.EmitAction("network.forget", ActionPhaseStarted, EventLevelInfo,
		fmt.Sprintf("Removing saved network %s", name),
		WithDetails(map[string]interface{}{"name": name}))

	err := dm.network.ForgetNetwork(ctx, name)
	if err != nil {
		dm.events.EmitAction("network.forget", ActionPhaseFailed, EventLevelError,
			err.Error(),
			WithDetails(map[string]interface{}{"name": name}))
		return err
	}

	// Clear last SSID if it was the forgotten network
	if getString(dm.config, KeyLastSSID) == name {
		dm.config.Delete(KeyLastSSID)
	}

	dm.events.EmitAction("network.forget", ActionPhaseCompleted, EventLevelSuccess,
		fmt.Sprintf("Removed saved network %s", name),
		WithDetails(map[string]interface{}{"name": name}))

	return nil
}

// EnableHotspot activates the device hotspot.
func (dm *DeviceManager) EnableHotspot(ctx context.Context) error {
	dm.events.EmitAction("hotspot.enable", ActionPhaseStarted, EventLevelInfo,
		"Enabling device hotspot")

	// Ensure profile exists
	if err := dm.network.EnsureHotspotProfile(ctx, dm.hotspotProfile(), dm.hotspotSSID(), dm.hotspotPassword()); err != nil {
		dm.setModeWithError(ModeError, err.Error())
		dm.events.EmitAction("hotspot.enable", ActionPhaseFailed, EventLevelError, err.Error())
		return err
	}

	if err := dm.network.EnableHotspot(ctx, dm.hotspotProfile()); err != nil {
		dm.setModeWithError(ModeError, err.Error())
		dm.events.EmitAction("hotspot.enable", ActionPhaseFailed, EventLevelError, err.Error())
		return err
	}

	dm.config.Set(KeyAPEnabled, true)
	dm.config.Set(KeyDeviceMode, string(ModeProvisioning))
	dm.config.Delete(KeyLastError)

	status, _ := dm.GetStatus()
	dm.events.EmitAction("hotspot.enable", ActionPhaseCompleted, EventLevelSuccess,
		"Device hotspot enabled",
		WithStatus(&status))

	return nil
}

// DisableHotspot deactivates the device hotspot.
func (dm *DeviceManager) DisableHotspot(ctx context.Context) error {
	dm.events.EmitAction("hotspot.disable", ActionPhaseStarted, EventLevelInfo,
		"Disabling device hotspot")

	err := dm.network.DisableHotspot(ctx, dm.hotspotProfile())
	if err != nil {
		dm.events.EmitAction("hotspot.disable", ActionPhaseFailed, EventLevelError, err.Error())
		return err
	}

	dm.config.Set(KeyAPEnabled, false)

	status, _ := dm.GetStatus()
	dm.events.EmitAction("hotspot.disable", ActionPhaseCompleted, EventLevelSuccess,
		"Device hotspot disabled",
		WithStatus(&status))

	return nil
}

// DiagnoseNetwork runs network diagnostics.
func (dm *DeviceManager) DiagnoseNetwork(ctx context.Context) NetworkDiagnostics {
	return dm.network.DiagnoseNetwork(ctx)
}

// TestInternet tests internet connectivity.
func (dm *DeviceManager) TestInternet(ctx context.Context) bool {
	return dm.network.TestInternet(ctx)
}

// RequestRestart requests a system or application restart.
func (dm *DeviceManager) RequestRestart(ctx context.Context, reason, scope string) error {
	if scope != "device" {
		scope = "app"
	}

	dm.config.Set(KeyRestartPending, true)
	dm.config.Set(KeyRestartReason, reason)

	dm.events.EmitAction("system.restart", ActionPhaseStarted, EventLevelInfo,
		fmt.Sprintf("Restarting %s", scope),
		WithDetails(map[string]interface{}{"reason": reason, "scope": scope}))

	if !dm.options.AllowSystemControl {
		dm.events.Emit(EventTypeSystem, EventLevelInfo,
			fmt.Sprintf("Restart requested (dry-run mode): %s", reason),
			WithDetails(map[string]interface{}{"scope": scope}))
		return nil
	}

	var err error
	if scope == "device" {
		err = dm.runCommand(ctx, "systemctl", "reboot")
	} else {
		err = dm.runCommand(ctx, "systemctl", "restart", "moonhub")
	}

	if err != nil {
		dm.events.EmitAction("system.restart", ActionPhaseFailed, EventLevelError, err.Error())
		return err
	}

	dm.events.EmitAction("system.restart", ActionPhaseCompleted, EventLevelSuccess,
		fmt.Sprintf("%s restart requested", scope))
	return nil
}

// RollbackToProvisioning returns device to provisioning mode with hotspot.
func (dm *DeviceManager) RollbackToProvisioning(ctx context.Context, reason string) error {
	dm.config.Set(KeyNetworkProvisioned, false)
	dm.setMode(ModeProvisioning)

	if err := dm.EnableHotspot(ctx); err != nil {
		dm.setModeWithError(ModeError, err.Error())
		return err
	}

	dm.config.Set(KeyLastError, reason)
	return nil
}

// FactoryReset completely resets the device to initial provisioning state.
// It clears all WiFi credentials, device configuration, and returns to hotspot mode.
func (dm *DeviceManager) FactoryReset(ctx context.Context) error {
	dm.events.EmitAction("factory.reset", ActionPhaseStarted, EventLevelWarning,
		"Starting factory reset - all data will be cleared")

	// 1. Forget all saved WiFi networks
	savedNetworks, err := dm.ListSavedNetworks(ctx)
	if err != nil {
		dm.events.Emit(EventTypeSystem, EventLevelWarning,
			fmt.Sprintf("Could not list saved networks: %v", err))
	} else {
		for _, net := range savedNetworks {
			if err := dm.network.ForgetNetwork(ctx, net.Name); err != nil {
				dm.events.Emit(EventTypeSystem, EventLevelWarning,
					fmt.Sprintf("Could not forget network %s: %v", net.Name, err))
			}
		}
	}

	// 2. Clear all device configuration keys
	deviceKeys := []string{
		KeyDeviceMode,
		KeyNetworkProvisioned,
		KeyLastSSID,
		KeyAPEnabled,
		KeyLastError,
		KeyAuthCode,
		KeyDeviceId,
		KeyRecoveryEnabled,
		KeyRecoveryFailureThreshold,
		KeyRecoveryCooldownMs,
		KeyRecoveryFailureStreak,
		KeyRecoveryLastAttemptAt,
		KeyRecoveryLastRecoveredAt,
		KeyRecoveryLastResult,
		KeyRecoveryCooldownUntil,
		KeyRecoveryActiveLock,
		KeyRecoveryLastReason,
	}

	for _, key := range deviceKeys {
		dm.config.Delete(key)
	}

	// 3. Clear onboarding status
	dm.config.Delete(KeyOnboardingCompleted)

	// 4. Enable hotspot for initial provisioning
	if err := dm.EnableHotspot(ctx); err != nil {
		dm.events.EmitAction("factory.reset", ActionPhaseFailed, EventLevelError,
			fmt.Sprintf("Failed to enable hotspot: %s", err.Error()))
		return err
	}

	// 5. Set mode to provisioning
	dm.config.Set(KeyDeviceMode, string(ModeProvisioning))

	status, _ := dm.GetStatus()
	dm.events.EmitAction("factory.reset", ActionPhaseCompleted, EventLevelSuccess,
		"Factory reset completed - device is ready for initial setup",
		WithStatus(&status))

	return nil
}

// Private helpers

func (dm *DeviceManager) hostname() string {
	return dm.options.Hostname
}

func (dm *DeviceManager) interfaceName() string {
	return dm.options.NetworkInterface
}

func (dm *DeviceManager) hotspotSSID() string {
	if ssid := getString(dm.config, KeyHotspotSSID); ssid != "" {
		return ssid
	}
	return DefaultHotspotSSID(dm.hostname())
}

func (dm *DeviceManager) hotspotProfile() string {
	if profile := getString(dm.config, KeyHotspotProfile); profile != "" {
		return profile
	}
	return DefaultHotspotProfile
}

func (dm *DeviceManager) hotspotPassword() string {
	if pw := getString(dm.config, KeyHotspotPassword); pw != "" {
		return pw
	}
	pw := fmt.Sprintf("MoonHub-%s-wifi", SanitizeHotspotSuffix(dm.hostname()))
	dm.config.Set(KeyHotspotPassword, pw)
	return pw
}

func (dm *DeviceManager) setMode(mode DeviceMode) {
	dm.config.Set(KeyDeviceMode, string(mode))
	if mode != ModeError {
		dm.config.Delete(KeyLastError)
	}
}

func (dm *DeviceManager) setModeWithError(mode DeviceMode, errMsg string) {
	dm.config.Set(KeyDeviceMode, string(mode))
	dm.config.Set(KeyLastError, errMsg)
}

func (dm *DeviceManager) rollbackToProvisioning(ctx context.Context, reason string) {
	_ = dm.RollbackToProvisioning(ctx, reason)
}

func (dm *DeviceManager) getRecoveryStatus() DeviceRecoveryStatus {
	cooldownUntil := getInt64(dm.config, KeyRecoveryCooldownUntil)
	activeLock := getRecoveryLock(dm.config, KeyRecoveryActiveLock)

	return DeviceRecoveryStatus{
		Enabled:               getBoolWithDefault(dm.config, KeyRecoveryEnabled, true),
		FailureThreshold:      getIntWithDefault(dm.config, KeyRecoveryFailureThreshold, DefaultRecoveryFailureThreshold),
		CooldownMs:            getIntWithDefault(dm.config, KeyRecoveryCooldownMs, DefaultRecoveryCooldownMs),
		DegradedNetworkStreak: getInt(dm.config, KeyRecoveryFailureStreak),
		LastAttemptAt:         getInt64(dm.config, KeyRecoveryLastAttemptAt),
		LastRecoveredAt:       getInt64(dm.config, KeyRecoveryLastRecoveredAt),
		LastResult:            getRecoveryResult(dm.config, KeyRecoveryLastResult),
		CooldownUntil:         cooldownUntil,
		ActiveLock:            activeLock,
		LastReason:            getString(dm.config, KeyRecoveryLastReason),
		InCooldown:            activeLock == LockCooldown && cooldownUntil > time.Now().UnixMilli(),
	}
}

func (dm *DeviceManager) runCommand(ctx context.Context, args ...string) error {
	result := dm.network.Run(ctx, args...)
	if !result.OK {
		return fmt.Errorf("%s", result.Stderr)
	}
	return nil
}

func (dm *DeviceManager) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dm.checkNetworkHealth(ctx)
		}
	}
}

func (dm *DeviceManager) checkNetworkHealth(ctx context.Context) {
	mode := NormalizeMode(getString(dm.config, KeyDeviceMode))
	if mode == ModeProvisioning || mode == ModeMaintenance {
		return
	}

	diagnostics := dm.network.DiagnoseNetwork(ctx)
	if !diagnostics.Connected || !diagnostics.InternetReachable {
		// Increment failure streak
		streak := getInt(dm.config, KeyRecoveryFailureStreak) + 1
		dm.config.Set(KeyRecoveryFailureStreak, streak)

		// Emit degraded event
		dm.events.Emit(EventTypeStatus, EventLevelWarning,
			"Network connectivity degraded",
			WithDetails(map[string]interface{}{
				"connected":        diagnostics.Connected,
				"internet":         diagnostics.InternetReachable,
				"dnsResolved":      diagnostics.DNSResolved,
				"activeConnection": diagnostics.ActiveConnection,
				"failureStreak":    streak,
			}))

		// Check if recovery should be triggered
		recovery := dm.getRecoveryStatus()
		if recovery.Enabled && streak >= recovery.FailureThreshold {
			go dm.attemptRecovery(ctx)
		}
	} else {
		// Reset failure streak on successful check
		dm.config.Set(KeyRecoveryFailureStreak, 0)
	}
}

func (dm *DeviceManager) attemptRecovery(ctx context.Context) error {
	dm.recoveryMu.Lock()
	defer dm.recoveryMu.Unlock()

	recovery := dm.getRecoveryStatus()

	// Check if recovery is enabled
	if !recovery.Enabled {
		return fmt.Errorf("recovery is disabled")
	}

	// Check cooldown
	if recovery.InCooldown {
		return fmt.Errorf("recovery is in cooldown until %d", recovery.CooldownUntil)
	}

	// Check if already in recovery
	if recovery.ActiveLock == LockRecovery {
		return fmt.Errorf("recovery already in progress")
	}

	// Set recovery lock
	dm.config.Set(KeyRecoveryActiveLock, string(LockRecovery))
	dm.config.Set(KeyRecoveryLastAttemptAt, time.Now().UnixMilli())

	status, _ := dm.GetStatus()
	dm.events.EmitAction("recovery.start", ActionPhaseStarted, EventLevelInfo,
		"Starting network recovery",
		WithDetails(map[string]interface{}{
			"failureStreak": recovery.DegradedNetworkStreak,
		}),
		WithStatus(&status))

	defer func() {
		if getRecoveryLock(dm.config, KeyRecoveryActiveLock) == LockRecovery {
			dm.config.Set(KeyRecoveryActiveLock, string(LockNone))
		}
	}()

	// Strategy 1: Try to reconnect to last known WiFi
	lastSSID := getString(dm.config, KeyLastSSID)
	if lastSSID != "" {
		dm.events.Emit(EventTypeSystem, EventLevelInfo,
			fmt.Sprintf("Attempting to reconnect to %s", lastSSID))

		reconnectErr := dm.network.ReconnectSavedNetwork(ctx, lastSSID)
		if reconnectErr != nil {
			dm.events.Emit(EventTypeSystem, EventLevelWarning,
				fmt.Sprintf("Reconnect to %s failed: %v", lastSSID, reconnectErr))
		} else {
			time.Sleep(3 * time.Second)
			diagnostics := dm.network.DiagnoseNetwork(ctx)
			if diagnostics.InternetReachable {
				dm.config.Set(KeyRecoveryLastResult, string(RecoveryRejoinSucceeded))
				dm.config.Set(KeyRecoveryLastRecoveredAt, time.Now().UnixMilli())
				dm.config.Set(KeyRecoveryFailureStreak, 0)
				dm.setCooldown()

				status, _ := dm.GetStatus()
				dm.events.EmitAction("recovery.complete", ActionPhaseCompleted, EventLevelSuccess,
					fmt.Sprintf("Reconnected to %s successfully", lastSSID),
					WithStatus(&status))
				return nil
			}
			dm.events.Emit(EventTypeSystem, EventLevelWarning,
				fmt.Sprintf("Reconnected to %s but internet is still unreachable", lastSSID))
		}
	}

	// Strategy 2: Enable hotspot as fallback
	dm.events.Emit(EventTypeSystem, EventLevelInfo,
		"Enabling hotspot as fallback")

	err := dm.EnableHotspot(ctx)
	if err != nil {
		dm.config.Set(KeyRecoveryLastResult, string(RecoveryFailed))
		dm.config.Set(KeyRecoveryLastReason, err.Error())

		dm.events.EmitAction("recovery.failed", ActionPhaseFailed, EventLevelError,
			fmt.Sprintf("Recovery failed: %s", err.Error()))

		return err
	}

	dm.config.Set(KeyRecoveryLastResult, string(RecoveryHotspotRestored))
	dm.config.Set(KeyRecoveryLastRecoveredAt, time.Now().UnixMilli())
	dm.config.Set(KeyRecoveryFailureStreak, 0)
	dm.setCooldown()

	dm.config.Set(KeyNetworkProvisioned, false)
	dm.config.Set(KeyDeviceMode, string(ModeProvisioning))

	dm.config.Set(KeyLastError, "Network recovery: hotspot enabled as fallback")

	status, _ = dm.GetStatus()
	dm.events.EmitAction("recovery.complete", ActionPhaseCompleted, EventLevelSuccess,
		"Hotspot enabled as recovery fallback",
		WithStatus(&status))

	return nil
}

// TriggerRecovery manually triggers the network recovery process.
func (dm *DeviceManager) TriggerRecovery(ctx context.Context) error {
	return dm.attemptRecovery(ctx)
}

func (dm *DeviceManager) setCooldown() {
	cooldownMs := getIntWithDefault(dm.config, KeyRecoveryCooldownMs, DefaultRecoveryCooldownMs)
	cooldownUntil := time.Now().Add(time.Duration(cooldownMs) * time.Millisecond).UnixMilli()
	dm.config.Set(KeyRecoveryCooldownUntil, cooldownUntil)
	dm.config.Set(KeyRecoveryActiveLock, string(LockCooldown))
}

// Config helper functions

func getString(c ConfigStore, key string) string {
	if v := c.Get(key); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(c ConfigStore, key string) bool {
	if v := c.Get(key); v != nil {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getBoolWithDefault(c ConfigStore, key string, def bool) bool {
	if v := c.Get(key); v != nil {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func getInt(c ConfigStore, key string) int {
	if v := c.Get(key); v != nil {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return 0
}

func getIntWithDefault(c ConfigStore, key string, def int) int {
	if v := c.Get(key); v != nil {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return def
}

func getInt64(c ConfigStore, key string) int64 {
	if v := c.Get(key); v != nil {
		switch n := v.(type) {
		case int64:
			return n
		case int:
			return int64(n)
		case float64:
			return int64(n)
		}
	}
	return 0
}

func getRecoveryLock(c ConfigStore, key string) DeviceRecoveryLock {
	v := getString(c, key)
	switch v {
	case string(LockCooldown), string(LockRecovery):
		return DeviceRecoveryLock(v)
	default:
		return LockNone
	}
}

func getRecoveryResult(c ConfigStore, key string) DeviceRecoveryResult {
	v := getString(c, key)
	switch v {
	case string(RecoveryIdle), string(RecoveryPending), string(RecoveryCooldown),
		string(RecoveryHotspotRestored), string(RecoveryRejoinSucceeded),
		string(RecoveryRejoinFailed), string(RecoveryFailed):
		return DeviceRecoveryResult(v)
	default:
		return RecoveryIdle
	}
}

// deriveRuntimeState calculates runtime state from device state
func deriveRuntimeState(mode DeviceMode, network DeviceNetworkSummary, restartPending bool, services DeviceServicesStatus) DeviceRuntimeStatus {
	var warnings []string
	var phase DeviceRuntimePhase

	if restartPending {
		phase = PhaseRestarting
		warnings = append(warnings, "Restart is pending.")
	} else if mode == ModeError {
		phase = PhaseError
		warnings = append(warnings, "Device is in error mode.")
	} else if mode == ModeMaintenance {
		phase = PhaseMaintenance
	} else if mode == ModeProvisioning {
		phase = PhaseProvisioning
	} else if mode == ModeConnecting {
		phase = PhaseConnecting
	} else if mode == ModeOnboarding {
		phase = PhaseOnboarding
	} else {
		phase = PhaseReady
	}

	if !services.NmcliAvailable {
		warnings = append(warnings, "nmcli is unavailable.")
	}
	if !services.NetworkManagerAvailable {
		warnings = append(warnings, "NetworkManager is unavailable.")
	}
	if network.Provisioned && !network.Connected && !network.APEnabled {
		warnings = append(warnings, "Configured network is not connected.")
	}
	if network.Connected && !network.InternetReachable {
		warnings = append(warnings, "Internet connectivity is unavailable.")
	}

	health := HealthOK
	if phase == PhaseError || !services.NetworkManagerAvailable || !services.NmcliAvailable {
		health = HealthError
	} else if len(warnings) > 0 {
		health = HealthWarning
	}

	if health != HealthOK && (phase == PhaseReady || phase == PhaseOnboarding) {
		phase = PhaseDegraded
	}

	summary := deriveSummary(phase, warnings)

	return DeviceRuntimeStatus{
		Phase:    phase,
		Health:   health,
		Summary:  summary,
		Warnings: warnings,
	}
}

func deriveSummary(phase DeviceRuntimePhase, warnings []string) string {
	switch phase {
	case PhaseReady:
		return "Device is ready."
	case PhaseDegraded:
		if len(warnings) > 0 {
			return warnings[0]
		}
		return "Device needs attention."
	case PhaseRestarting:
		return "Restart is pending."
	case PhaseProvisioning:
		return "Waiting for initial WiFi provisioning."
	case PhaseConnecting:
		return "Connecting to the configured WiFi."
	case PhaseOnboarding:
		return "Waiting for guided setup to finish."
	case PhaseMaintenance:
		return "Device is in maintenance mode."
	default:
		return "Device requires recovery."
	}
}

// JSONConfigStore is a simple file-based config store implementation
type JSONConfigStore struct {
	path string
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewJSONConfigStore creates a new JSON file-based config store
func NewJSONConfigStore(path string) (*JSONConfigStore, error) {
	store := &JSONConfigStore{
		path: path,
		data: make(map[string]interface{}),
	}
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return store, nil
}

func (s *JSONConfigStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.data)
}

// Save persists the store to disk.
func (s *JSONConfigStore) Save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

// Get retrieves a value from the store
func (s *JSONConfigStore) Get(key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[key]
}

// Set stores a value in the store
func (s *JSONConfigStore) Set(key string, value interface{}) {
	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()
	_ = s.Save()
}

// Delete removes a key from the store
func (s *JSONConfigStore) Delete(key string) {
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()
	_ = s.Save()
}

// Auth code management

// GetOrCreateAuthCode returns the existing auth code or generates a new one.
func (dm *DeviceManager) GetOrCreateAuthCode() (code string, deviceId string, err error) {
	code = getString(dm.config, KeyAuthCode)
	deviceId = getString(dm.config, KeyDeviceId)

	if code == "" {
		code, err = generateAuthCode()
		if err != nil {
			return "", "", err
		}
		dm.config.Set(KeyAuthCode, code)
	}

	if deviceId == "" {
		deviceId = generateDeviceId(dm.hostname())
		dm.config.Set(KeyDeviceId, deviceId)
	}

	return code, deviceId, nil
}

// RegenerateAuthCode generates a new auth code.
func (dm *DeviceManager) RegenerateAuthCode() (code string, deviceId string, err error) {
	code, err = generateAuthCode()
	if err != nil {
		return "", "", err
	}
	dm.config.Set(KeyAuthCode, code)

	deviceId = getString(dm.config, KeyDeviceId)
	if deviceId == "" {
		deviceId = generateDeviceId(dm.hostname())
		dm.config.Set(KeyDeviceId, deviceId)
	}

	return code, deviceId, nil
}

// generateAuthCode generates a random 6-digit auth code using crypto/rand.
func generateAuthCode() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	n := 100000 + int(binary.BigEndian.Uint32(b[:])%900000)
	return fmt.Sprintf("%06d", n), nil
}

// generateDeviceId generates a device identifier.
func generateDeviceId(hostname string) string {
	year := time.Now().Year()
	var suffix string
	if len(hostname) > 3 {
		suffix = strings.ToUpper(hostname[len(hostname)-3:])
	} else {
		suffix = strings.ToUpper(hostname)
	}
	return fmt.Sprintf("YSHU-%d-%s", year, suffix)
}
