//go:build darwin && arm64

package hypervisor

/*
#cgo darwin LDFLAGS: -framework Hypervisor
#include <Hypervisor/hv_vcpu.h>
#include <Hypervisor/hv_vcpu_types.h>
*/
import "C"

import "fmt"

func (c *VCPU) GetReg(r Reg) (uint64, error) {
	if c == nil {
		return 0, fmt.Errorf("hv: VCPU is nil")
	}

	// Security: Lock to prevent use-after-free
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return 0, fmt.Errorf("hv: VCPU is closed")
	}

	// Security: Enhanced register bounds validation
	if r < RegX0 || r > RegCPSR {
		return 0, fmt.Errorf("hv: invalid register %d (must be %d-%d)", r, RegX0, RegCPSR)
	}

	var val C.ulonglong
	var ret C.hv_return_t

	// Use system register API for SP
	if r == RegSP {
		ret = C.hv_vcpu_get_sys_reg(C.hv_vcpu_t(c.id), C.HV_SYS_REG_SP_EL0, &val)
	} else {
		// Security: Additional validation for register mapping
		hvReg := regToHV(r)
		if hvReg == C.HV_REG_X0 && r != RegX0 {
			return 0, fmt.Errorf("hv: register mapping failed for %d", r)
		}
		ret = C.hv_vcpu_get_reg(C.hv_vcpu_t(c.id), hvReg, &val)
	}

	if err := hvErr(ret); err != nil {
		recordResourceError()
		return 0, fmt.Errorf("failed to get register %d: %w", r, err)
	}

	recordRegisterOp()
	return uint64(val), nil
}

func (c *VCPU) SetReg(r Reg, v uint64) error {
	if c == nil {
		return fmt.Errorf("hv: VCPU is nil")
	}

	// Security: Lock to prevent use-after-free
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return fmt.Errorf("hv: VCPU is closed")
	}

	// Security: Enhanced register bounds validation
	if r < RegX0 || r > RegCPSR {
		return fmt.Errorf("hv: invalid register %d (must be %d-%d)", r, RegX0, RegCPSR)
	}

	var ret C.hv_return_t

	// Use system register API for SP
	if r == RegSP {
		ret = C.hv_vcpu_set_sys_reg(C.hv_vcpu_t(c.id), C.HV_SYS_REG_SP_EL0, C.ulonglong(v))
	} else {
		// Security: Additional validation for register mapping
		hvReg := regToHV(r)
		if hvReg == C.HV_REG_X0 && r != RegX0 {
			return fmt.Errorf("hv: register mapping failed for %d", r)
		}
		ret = C.hv_vcpu_set_reg(C.hv_vcpu_t(c.id), hvReg, C.ulonglong(v))
	}

	if err := hvErr(ret); err != nil {
		recordResourceError()
		return fmt.Errorf("failed to set register %d: %w", r, err)
	}

	recordRegisterOp()
	return nil
}

func (c *VCPU) GetPC() (uint64, error) { return c.GetReg(RegPC) }
func (c *VCPU) SetPC(v uint64) error   { return c.SetReg(RegPC, v) }

// RegBatch represents a batch of register operations for performance
type RegBatch map[Reg]uint64

// GetRegs retrieves multiple registers in a single call (performance optimization)
// Note: Currently implemented as individual calls, but foundation for batching
func (c *VCPU) GetRegs(regs []Reg) (RegBatch, error) {
	if c == nil {
		return nil, fmt.Errorf("hv: VCPU is nil")
	}

	batch := make(RegBatch, len(regs))
	for _, reg := range regs {
		val, err := c.GetReg(reg)
		if err != nil {
			return nil, err
		}
		batch[reg] = val
	}
	return batch, nil
}

// SetRegs sets multiple registers in a single call (performance optimization)
// Note: Currently implemented as individual calls, but foundation for batching
func (c *VCPU) SetRegs(batch RegBatch) error {
	if c == nil {
		return fmt.Errorf("hv: VCPU is nil")
	}

	for reg, val := range batch {
		if err := c.SetReg(reg, val); err != nil {
			return err
		}
	}
	return nil
}

// regToHV maps our Reg enum to the Hypervisor framework hv_reg_t constants.
func regToHV(r Reg) C.hv_reg_t {
	switch r {
	case RegX0:
		return C.HV_REG_X0
	case RegX1:
		return C.HV_REG_X1
	case RegX2:
		return C.HV_REG_X2
	case RegX3:
		return C.HV_REG_X3
	case RegX4:
		return C.HV_REG_X4
	case RegX5:
		return C.HV_REG_X5
	case RegX6:
		return C.HV_REG_X6
	case RegX7:
		return C.HV_REG_X7
	case RegX8:
		return C.HV_REG_X8
	case RegX9:
		return C.HV_REG_X9
	case RegX10:
		return C.HV_REG_X10
	case RegX11:
		return C.HV_REG_X11
	case RegX12:
		return C.HV_REG_X12
	case RegX13:
		return C.HV_REG_X13
	case RegX14:
		return C.HV_REG_X14
	case RegX15:
		return C.HV_REG_X15
	case RegX16:
		return C.HV_REG_X16
	case RegX17:
		return C.HV_REG_X17
	case RegX18:
		return C.HV_REG_X18
	case RegX19:
		return C.HV_REG_X19
	case RegX20:
		return C.HV_REG_X20
	case RegX21:
		return C.HV_REG_X21
	case RegX22:
		return C.HV_REG_X22
	case RegX23:
		return C.HV_REG_X23
	case RegX24:
		return C.HV_REG_X24
	case RegX25:
		return C.HV_REG_X25
	case RegX26:
		return C.HV_REG_X26
	case RegX27:
		return C.HV_REG_X27
	case RegX28:
		return C.HV_REG_X28
	case RegFP:
		return C.HV_REG_FP
	case RegLR:
		return C.HV_REG_LR
	case RegPC:
		return C.HV_REG_PC
	case RegCPSR:
		return C.HV_REG_CPSR
	default:
		// This should not happen due to validation in GetReg/SetReg
		return C.HV_REG_X0
	}
}
