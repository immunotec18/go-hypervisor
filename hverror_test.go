//go:build darwin && arm64

package hypervisor

import (
	"strings"
	"testing"
)

func TestHVError(t *testing.T) {
	tests := []struct {
		name     string
		code     uint32
		expected string
	}{
		{
			name:     "HV_SUCCESS",
			code:     HV_SUCCESS,
			expected: "hv: success",
		},
		{
			name:     "HV_ERROR",
			code:     HV_ERROR,
			expected: "hv: general error (HV_ERROR) - check system requirements and API usage",
		},
		{
			name:     "HV_BUSY",
			code:     HV_BUSY,
			expected: "hv: resource busy (HV_BUSY) - another operation is in progress",
		},
		{
			name:     "HV_BAD_ARGUMENT",
			code:     HV_BAD_ARGUMENT,
			expected: "hv: invalid argument (HV_BAD_ARGUMENT) - check parameter values and alignment",
		},
		{
			name:     "HV_ILLEGAL_GUEST_STATE",
			code:     HV_ILLEGAL_GUEST_STATE,
			expected: "hv: illegal guest state (HV_ILLEGAL_GUEST_STATE) - guest CPU state is invalid",
		},
		{
			name:     "HV_NO_RESOURCES",
			code:     HV_NO_RESOURCES,
			expected: "hv: insufficient resources (HV_NO_RESOURCES) - system memory or limits exceeded",
		},
		{
			name:     "HV_NO_DEVICE",
			code:     HV_NO_DEVICE,
			expected: "hv: device not found (HV_NO_DEVICE) - hardware virtualization unavailable",
		},
		{
			name:     "HV_DENIED",
			code:     HV_DENIED,
			expected: "hv: access denied (HV_DENIED) - missing entitlement 'com.apple.security.hypervisor' or insufficient privileges",
		},
		{
			name:     "HV_EXISTS",
			code:     HV_EXISTS,
			expected: "hv: resource exists (HV_EXISTS) - VM or vCPU already created",
		},
		{
			name:     "HV_UNSUPPORTED",
			code:     HV_UNSUPPORTED,
			expected: "hv: operation unsupported (HV_UNSUPPORTED) - feature not available on this hardware/OS",
		},
		{
			name:     "Unknown error code",
			code:     0x12345678,
			expected: "hv: unknown error code 0x12345678 - consult Apple Hypervisor.framework documentation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HVError{Code: tt.code}
			got := err.Error()
			if got != tt.expected {
				t.Errorf("HVError{Code: 0x%08x}.Error() = %q, want %q", tt.code, got, tt.expected)
			}
		})
	}
}

func TestHvErrLogic(t *testing.T) {
	t.Run("can create HVError directly", func(t *testing.T) {
		err := HVError{Code: HV_ERROR}
		errMsg := err.Error()
		if !strings.Contains(errMsg, "HV_ERROR") {
			t.Errorf("Error message %q should contain 'HV_ERROR'", errMsg)
		}
	})

	t.Run("different error codes produce different messages", func(t *testing.T) {
		err1 := HVError{Code: HV_ERROR}
		err2 := HVError{Code: HV_BUSY}

		if err1.Error() == err2.Error() {
			t.Error("Different error codes should produce different messages")
		}
	})
}

func TestErrorConstants(t *testing.T) {
	// Verify that our constants match the expected Apple error codes
	expectedCodes := map[string]uint32{
		"HV_SUCCESS":             0x00000000,
		"HV_ERROR":               0xFAE94001,
		"HV_BUSY":                0xFAE94002,
		"HV_BAD_ARGUMENT":        0xFAE94003,
		"HV_ILLEGAL_GUEST_STATE": 0xFAE94004,
		"HV_NO_RESOURCES":        0xFAE94005,
		"HV_NO_DEVICE":           0xFAE94006,
		"HV_DENIED":              0xFAE94007,
		"HV_EXISTS":              0xFAE94008,
		"HV_UNSUPPORTED":         0xFAE9400F,
	}

	actualCodes := map[string]uint32{
		"HV_SUCCESS":             HV_SUCCESS,
		"HV_ERROR":               HV_ERROR,
		"HV_BUSY":                HV_BUSY,
		"HV_BAD_ARGUMENT":        HV_BAD_ARGUMENT,
		"HV_ILLEGAL_GUEST_STATE": HV_ILLEGAL_GUEST_STATE,
		"HV_NO_RESOURCES":        HV_NO_RESOURCES,
		"HV_NO_DEVICE":           HV_NO_DEVICE,
		"HV_DENIED":              HV_DENIED,
		"HV_EXISTS":              HV_EXISTS,
		"HV_UNSUPPORTED":         HV_UNSUPPORTED,
	}

	for name, expected := range expectedCodes {
		actual, exists := actualCodes[name]
		if !exists {
			t.Errorf("Missing constant %s", name)
			continue
		}
		if actual != expected {
			t.Errorf("Constant %s = 0x%08x, want 0x%08x", name, actual, expected)
		}
	}
}
