//go:build darwin && arm64 && hypervisor

package hypervisor

import (
	"testing"
)

func TestMetrics(t *testing.T) {
	// Skip hypervisor tests in CI environments (no nested virtualization support)
	if isCI() {
		t.Skip("Skipping hypervisor tests in CI environment")
	}

	// Reset metrics for clean test
	ResetMetrics()

	// Verify initial state
	metrics := GetMetrics()
	if metrics.VMCreated != 0 {
		t.Errorf("Expected VMCreated=0, got %d", metrics.VMCreated)
	}

	// Create and test VM (if entitled)
	vm, err := NewVM()
	if err != nil {
		if err.Error() == "hv: access denied (HV_DENIED) - missing entitlement 'com.apple.security.hypervisor' or insufficient privileges" {
			t.Skip("Skipping metrics test: missing hypervisor entitlements")
		}
		t.Fatalf("Failed to create VM: %v", err)
	}
	defer vm.Close()

	// Check VM creation was recorded
	metrics = GetMetrics()
	if metrics.VMCreated != 1 {
		t.Errorf("Expected VMCreated=1, got %d", metrics.VMCreated)
	}
	if metrics.AvgVMCreateTimeNs == 0 {
		t.Errorf("Expected non-zero VM create time")
	}

	// Test VCPU metrics
	vcpu, err := vm.NewVCPU()
	if err != nil {
		t.Fatalf("Failed to create VCPU: %v", err)
	}
	defer vcpu.Close()

	metrics = GetMetrics()
	if metrics.VCPUCreated != 1 {
		t.Errorf("Expected VCPUCreated=1, got %d", metrics.VCPUCreated)
	}

	// Test register operation metrics
	_, err = vcpu.GetReg(RegX0)
	if err != nil {
		t.Fatalf("Failed to get register: %v", err)
	}

	metrics = GetMetrics()
	if metrics.RegisterOps != 1 {
		t.Errorf("Expected RegisterOps=1, got %d", metrics.RegisterOps)
	}

	t.Logf("Final metrics: %+v", metrics)
}
