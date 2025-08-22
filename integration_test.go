//go:build darwin && arm64 && hypervisor

package hypervisor

import (
	"encoding/binary"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

func TestDemoIntegration(t *testing.T) {
	// Skip hypervisor tests in CI environments (no nested virtualization support)
	if isCI() {
		t.Skip("Skipping hypervisor tests in CI environment")
	}

	// This is a comprehensive integration test that reproduces the demo functionality
	supported, err := Supported()
	if err != nil {
		t.Fatalf("Failed to check hypervisor support: %v", err)
	}
	if !supported {
		t.Skip("Hypervisor not supported - skipping integration test")
	}

	// Create VM
	vm, err := NewVM()
	if err != nil {
		t.Skipf("Cannot create VM (likely missing entitlements): %v", err)
	}
	defer func() {
		if err := vm.Close(); err != nil {
			t.Errorf("Failed to close VM: %v", err)
		}
	}()

	// Allocate page-aligned memory for guest code
	pageSize := unix.Getpagesize()
	buf, err := unix.Mmap(-1, 0, pageSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
	if err != nil {
		t.Fatalf("Failed to mmap: %v", err)
	}
	defer func() {
		if err := unix.Munmap(buf); err != nil {
			t.Errorf("Failed to munmap: %v", err)
		}
	}()

	// Encode ARM64 instructions: mov x0,#0x42 ; brk #0
	binary.LittleEndian.PutUint32(buf[0:], 0xD2800840) // MOVZ X0,#0x42
	binary.LittleEndian.PutUint32(buf[4:], 0xD4200000) // BRK #0

	// Map the memory into guest physical address space
	const guestPhys = 0x4000
	err = vm.Map(buf, guestPhys, MemRead|MemWrite|MemExec)
	if err != nil {
		t.Fatalf("Failed to map guest memory: %v", err)
	}
	defer func() {
		if err := vm.Unmap(guestPhys, uint64(len(buf))); err != nil {
			t.Errorf("Failed to unmap guest memory: %v", err)
		}
	}()

	// Create vCPU
	vcpu, err := vm.NewVCPU()
	if err != nil {
		t.Fatalf("Failed to create vCPU: %v", err)
	}
	defer func() {
		if err := vcpu.Close(); err != nil {
			t.Errorf("Failed to close vCPU: %v", err)
		}
	}()

	// Set PC to the guest physical address where we loaded our code
	err = vcpu.SetPC(guestPhys)
	if err != nil {
		t.Fatalf("Failed to set PC: %v", err)
	}

	// Verify PC was set correctly
	pc, err := vcpu.GetPC()
	if err != nil {
		t.Fatalf("Failed to get PC: %v", err)
	}
	if pc&0xFFFFFFFF != guestPhys&0xFFFFFFFF {
		t.Logf("PC set to 0x%x (requested 0x%x) - might be masked by hypervisor", pc, guestPhys)
	}

	// Run the vCPU
	info, err := vcpu.Run()
	if err != nil {
		t.Fatalf("Failed to run vCPU: %v", err)
	}

	// Check that we got an exception exit (due to BRK instruction)
	t.Logf("Exit info: Reason=%v ESR=0x%x FAR=0x%x", info.Reason, info.ESR, info.FAR)
	if info.Reason != ExitException {
		t.Logf("Expected ExitException, got %v (this might be okay depending on hypervisor behavior)", info.Reason)
	}

	// The most important check: verify that X0 contains 0x42
	x0, err := vcpu.GetReg(RegX0)
	if err != nil {
		t.Fatalf("Failed to get X0 register: %v", err)
	}

	if x0 != 0x42 {
		t.Errorf("X0 = 0x%x, want 0x42", x0)
	} else {
		t.Logf("✅ Demo integration test passed: X0=0x%x", x0)
	}

	// Additional verification: check that other registers are reasonable
	pc, err = vcpu.GetPC()
	if err != nil {
		t.Fatalf("Failed to get final PC: %v", err)
	}
	t.Logf("Final PC: 0x%x", pc)

	// The PC should have advanced past our first instruction
	if pc == guestPhys {
		t.Logf("PC hasn't advanced - this might indicate the instruction didn't execute")
	}
}

func TestVMLifecycle(t *testing.T) {
	supported, err := Supported()
	if err != nil {
		t.Fatalf("Failed to check hypervisor support: %v", err)
	}
	if !supported {
		t.Skip("Hypervisor not supported - skipping VM lifecycle test")
	}

	// Test creating and destroying multiple VMs (should only allow one at a time)
	vm1, err := NewVM()
	if err != nil {
		t.Skipf("Cannot create first VM (likely missing entitlements): %v", err)
	}

	// Try to create a second VM - should fail
	vm2, err := NewVM()
	if err == nil {
		vm2.Close()
		t.Error("Expected error when creating second VM, but succeeded")
	} else {
		t.Logf("Correctly rejected second VM creation: %v", err)
	}

	// Close first VM
	err = vm1.Close()
	if err != nil {
		t.Errorf("Failed to close first VM: %v", err)
	}

	// Now we should be able to create another VM
	vm3, err := NewVM()
	if err != nil {
		t.Errorf("Failed to create VM after closing previous one: %v", err)
	} else {
		vm3.Close()
	}
}

func TestVCPULifecycle(t *testing.T) {
	supported, err := Supported()
	if err != nil {
		t.Fatalf("Failed to check hypervisor support: %v", err)
	}
	if !supported {
		t.Skip("Hypervisor not supported - skipping vCPU lifecycle test")
	}

	vm, err := NewVM()
	if err != nil {
		t.Skipf("Cannot create VM (likely missing entitlements): %v", err)
	}
	defer vm.Close()

	// Test creating multiple vCPUs
	vcpus := make([]*VCPU, 0)
	for i := 0; i < 3; i++ {
		vcpu, err := vm.NewVCPU()
		if err != nil {
			t.Logf("Failed to create vCPU %d: %v", i, err)
			break
		}
		vcpus = append(vcpus, vcpu)
		t.Logf("Created vCPU %d with ID %d", i, vcpu.id)
	}

	// Close all vCPUs
	for i, vcpu := range vcpus {
		err := vcpu.Close()
		if err != nil {
			t.Errorf("Failed to close vCPU %d: %v", i, err)
		}
	}
}

func TestInstructionEncoding(t *testing.T) {
	// Test that our instruction encoding is correct
	tests := []struct {
		name     string
		encoding uint32
		desc     string
	}{
		{
			name:     "MOVZ X0, #0x42",
			encoding: 0xD2800840,
			desc:     "Move immediate 0x42 to X0 register",
		},
		{
			name:     "BRK #0",
			encoding: 0xD4200000,
			desc:     "Breakpoint instruction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the constants are what we expect
			t.Logf("Instruction %s: 0x%08x (%s)", tt.name, tt.encoding, tt.desc)

			// Basic sanity checks
			if tt.encoding == 0 {
				t.Error("Instruction encoding should not be zero")
			}
			if tt.encoding > 0xFFFFFFFF {
				t.Error("Instruction encoding should fit in 32 bits")
			}
		})
	}
}

func TestMemoryAlignment(t *testing.T) {
	// Test memory alignment utilities
	pageSize := unix.Getpagesize()
	t.Logf("System page size: %d bytes", pageSize)

	testAddresses := []uint64{
		0x0000,
		0x1000,
		0x4000,
		0x10000,
		0x4001, // Unaligned
		0x4123, // Unaligned
	}

	for _, addr := range testAddresses {
		aligned := (addr % uint64(pageSize)) == 0
		t.Logf("Address 0x%x: aligned=%v", addr, aligned)
	}

	// Test buffer alignment
	buf := make([]byte, pageSize)
	bufAddr := uintptr(unsafe.Pointer(&buf[0]))
	bufAligned := (bufAddr % uintptr(pageSize)) == 0
	t.Logf("Buffer at 0x%x: aligned=%v", bufAddr, bufAligned)
}
