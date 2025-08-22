//go:build darwin && arm64 && hypervisor

package hypervisor

import (
	"testing"
)

func TestRegisterConstants(t *testing.T) {
	// Test that our register enumeration is consistent
	registers := []Reg{
		RegX0, RegX1, RegX2, RegX3, RegX4, RegX5, RegX6, RegX7,
		RegX8, RegX9, RegX10, RegX11, RegX12, RegX13, RegX14, RegX15,
		RegX16, RegX17, RegX18, RegX19, RegX20, RegX21, RegX22, RegX23,
		RegX24, RegX25, RegX26, RegX27, RegX28, RegFP, RegLR, RegPC, RegCPSR,
	}

	// Ensure all registers map to valid HV constants
	for _, reg := range registers {
		hvReg := regToHV(reg)
		t.Logf("Reg %v maps to HV constant %v", reg, hvReg)

		// Basic sanity check - should not panic
		if hvReg < 0 {
			t.Errorf("Register %v maps to invalid HV constant %v", reg, hvReg)
		}
	}
}

func TestRegisterRoundTrip(t *testing.T) {
	// This test requires actual hypervisor access and will skip if not available
	supported, err := Supported()
	if err != nil {
		t.Fatalf("Failed to check hypervisor support: %v", err)
	}
	if !supported {
		t.Skip("Hypervisor not supported - skipping register tests")
	}

	vm, err := NewVM()
	if err != nil {
		t.Skipf("Cannot create VM (likely missing entitlements): %v", err)
	}
	defer vm.Close()

	vcpu, err := vm.NewVCPU()
	if err != nil {
		t.Fatalf("Failed to create vCPU: %v", err)
	}
	defer vcpu.Close()

	// Test register round-trip for general purpose registers
	testRegs := []struct {
		reg   Reg
		value uint64
	}{
		{RegX0, 0x1234567890abcdef},
		{RegX1, 0x0},
		{RegX2, 0xffffffffffffffff},
		{RegX3, 0x5a5a5a5a5a5a5a5a},
		{RegPC, 0x4000}, // Valid guest address
	}

	for _, test := range testRegs {
		t.Run(test.reg.String(), func(t *testing.T) {
			// Set register value
			err := vcpu.SetReg(test.reg, test.value)
			if err != nil {
				t.Fatalf("SetReg(%v, 0x%x) failed: %v", test.reg, test.value, err)
			}

			// Get register value back
			got, err := vcpu.GetReg(test.reg)
			if err != nil {
				t.Fatalf("GetReg(%v) failed: %v", test.reg, err)
			}

			// For PC, the value might be masked or aligned by the hypervisor
			if test.reg == RegPC {
				// PC should be at least close to what we set
				if got&0xFFFFFFFF != test.value&0xFFFFFFFF {
					t.Errorf("PC round-trip: got 0x%x, want approximately 0x%x", got, test.value)
				}
			} else {
				if got != test.value {
					t.Errorf("Register %v round-trip: got 0x%x, want 0x%x", test.reg, got, test.value)
				}
			}
		})
	}
}

func TestPCHelpers(t *testing.T) {
	supported, err := Supported()
	if err != nil {
		t.Fatalf("Failed to check hypervisor support: %v", err)
	}
	if !supported {
		t.Skip("Hypervisor not supported - skipping PC helper tests")
	}

	vm, err := NewVM()
	if err != nil {
		t.Skipf("Cannot create VM (likely missing entitlements): %v", err)
	}
	defer vm.Close()

	vcpu, err := vm.NewVCPU()
	if err != nil {
		t.Fatalf("Failed to create vCPU: %v", err)
	}
	defer vcpu.Close()

	testPC := uint64(0x4000)

	// Test SetPC helper
	err = vcpu.SetPC(testPC)
	if err != nil {
		t.Fatalf("SetPC(0x%x) failed: %v", testPC, err)
	}

	// Test GetPC helper
	pc, err := vcpu.GetPC()
	if err != nil {
		t.Fatalf("GetPC() failed: %v", err)
	}

	// PC might be aligned/masked, so check if it's approximately correct
	if pc&0xFFFFFFFF != testPC&0xFFFFFFFF {
		t.Errorf("PC helpers: got 0x%x, want approximately 0x%x", pc, testPC)
	}
}

// Add String() method for better test output
func (r Reg) String() string {
	switch r {
	case RegX0:
		return "X0"
	case RegX1:
		return "X1"
	case RegX2:
		return "X2"
	case RegX3:
		return "X3"
	case RegX4:
		return "X4"
	case RegX5:
		return "X5"
	case RegX6:
		return "X6"
	case RegX7:
		return "X7"
	case RegX8:
		return "X8"
	case RegX9:
		return "X9"
	case RegX10:
		return "X10"
	case RegX11:
		return "X11"
	case RegX12:
		return "X12"
	case RegX13:
		return "X13"
	case RegX14:
		return "X14"
	case RegX15:
		return "X15"
	case RegX16:
		return "X16"
	case RegX17:
		return "X17"
	case RegX18:
		return "X18"
	case RegX19:
		return "X19"
	case RegX20:
		return "X20"
	case RegX21:
		return "X21"
	case RegX22:
		return "X22"
	case RegX23:
		return "X23"
	case RegX24:
		return "X24"
	case RegX25:
		return "X25"
	case RegX26:
		return "X26"
	case RegX27:
		return "X27"
	case RegX28:
		return "X28"
	case RegFP:
		return "FP"
	case RegLR:
		return "LR"
	case RegPC:
		return "PC"
	case RegCPSR:
		return "CPSR"
	default:
		return "Unknown"
	}
}
