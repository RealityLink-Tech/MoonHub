package devices

import (
	"testing"
)

func TestNewPairingManager(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	pm := NewPairingManager(store)
	if pm == nil {
		t.Fatal("expected pairing manager instance")
	}
}

func TestGenerateCodeOnce(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	pm := NewPairingManager(store)

	// Generate code
	code1, err := pm.GenerateCodeOnce()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	if len(code1) != 6 {
		t.Errorf("expected 6 character code, got %d", len(code1))
	}

	// Call again - should return same code
	code2, err := pm.GenerateCodeOnce()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	if code1 != code2 {
		t.Errorf("expected same code, got %s and %s", code1, code2)
	}
}

func TestGetCurrentCode(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	pm := NewPairingManager(store)

	// No code yet
	code := pm.GetCurrentCode()
	if code != "" {
		t.Errorf("expected empty code, got %s", code)
	}

	// Generate code
	generatedCode, err := pm.GenerateCodeOnce()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Get current code
	code = pm.GetCurrentCode()
	if code != generatedCode {
		t.Errorf("expected %s, got %s", generatedCode, code)
	}
}

func TestValidateCode(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	pm := NewPairingManager(store)

	// Generate code
	code, err := pm.GenerateCodeOnce()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	deviceInfo := &PairedDevice{
		ID:   "test-device-001",
		Name: "Test Device",
	}

	// Validate correct code
	result, err := pm.ValidateCode(code, deviceInfo)
	if err != nil {
		t.Fatalf("failed to validate code: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	if result.Token == "" {
		t.Error("expected token to be generated")
	}

	if result.Device == nil {
		t.Fatal("expected device info")
	}

	if result.Device.ID != "test-device-001" {
		t.Errorf("expected device ID test-device-001, got %s", result.Device.ID)
	}

	// Try to validate same code again (should fail - already used)
	result2, err := pm.ValidateCode(code, &PairedDevice{ID: "test-device-002"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result2.Success {
		t.Error("expected failure - code already used")
	}

	if result2.Error != "pairing code already used" {
		t.Errorf("expected 'pairing code already used' error, got: %s", result2.Error)
	}
}

func TestValidateCodeInvalid(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	pm := NewPairingManager(store)

	deviceInfo := &PairedDevice{
		ID:   "test-device-001",
		Name: "Test Device",
	}

	// Validate with no code generated
	result, err := pm.ValidateCode("WRONG1", deviceInfo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected failure - no code generated")
	}

	// Should return "no pairing code available" since no code was generated
	if result.Error != "no pairing code available" {
		t.Errorf("expected 'no pairing code available' error, got: %s", result.Error)
	}
}

func TestRegenerateCode(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	pm := NewPairingManager(store)

	// Generate initial code
	code1, err := pm.GenerateCodeOnce()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Regenerate code
	code2, err := pm.RegenerateCode()
	if err != nil {
		t.Fatalf("failed to regenerate code: %v", err)
	}

	if code1 == code2 {
		t.Error("expected different codes after regeneration")
	}

	// Current code should be new code
	currentCode := pm.GetCurrentCode()
	if currentCode != code2 {
		t.Errorf("expected current code %s, got %s", code2, currentCode)
	}
}

func TestIsPaired(t *testing.T) {
	store, err := NewDeviceStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create device store: %v", err)
	}

	pm := NewPairingManager(store)

	// Not paired initially
	if pm.IsPaired() {
		t.Error("expected not paired initially")
	}

	// Generate and use code
	code, err := pm.GenerateCodeOnce()
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	_, err = pm.ValidateCode(code, &PairedDevice{
		ID:   "test-device-001",
		Name: "Test Device",
	})
	if err != nil {
		t.Fatalf("failed to validate code: %v", err)
	}

	// Now paired
	if !pm.IsPaired() {
		t.Error("expected to be paired")
	}
}

func TestGeneratePairingCode(t *testing.T) {
	// Generate multiple codes to check format
	for i := 0; i < 100; i++ {
		code, err := generatePairingCode()
		if err != nil {
			t.Fatalf("failed to generate code: %v", err)
		}

		if len(code) != 6 {
			t.Errorf("expected 6 character code, got %d (%s)", len(code), code)
		}

		// Check format: 2 letters + 4 digits
		for j := 0; j < 2; j++ {
			c := code[j]
			if c < 'A' || c > 'Z' {
				t.Errorf("expected uppercase letter at position %d, got %c", j, c)
			}
		}

		for j := 2; j < 6; j++ {
			c := code[j]
			if c < '0' || c > '9' {
				t.Errorf("expected digit at position %d, got %c", j, c)
			}
		}
	}
}
