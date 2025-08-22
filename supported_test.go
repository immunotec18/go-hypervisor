//go:build darwin && arm64

package hypervisor

import (
	"testing"
)

func TestSupported(t *testing.T) {
	t.Run("should return result without error", func(t *testing.T) {
		// Skip hypervisor tests in CI environments
		if isCI() {
			t.Skip("Skipping hypervisor tests in CI environment")
		}

		supported, err := Supported()
		if err != nil {
			t.Fatalf("Supported() returned error: %v", err)
		}

		t.Logf("Hypervisor support: %v", supported)
		if !supported {
			t.Skip("Hypervisor not supported on this system - skipping remaining tests")
		}
	})
}

func TestSupportedConsistency(t *testing.T) {
	t.Run("should return consistent results", func(t *testing.T) {
		// Skip hypervisor tests in CI environments
		if isCI() {
			t.Skip("Skipping hypervisor tests in CI environment")
		}

		results := make([]bool, 5)
		for i := 0; i < 5; i++ {
			supported, err := Supported()
			if err != nil {
				t.Fatalf("Supported() call %d returned error: %v", i, err)
			}
			results[i] = supported
		}

		// All results should be identical
		first := results[0]
		for i, result := range results {
			if result != first {
				t.Errorf("Inconsistent result at call %d: got %v, want %v", i, result, first)
			}
		}
	})
}
