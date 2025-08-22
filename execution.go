//go:build darwin && arm64

package hypervisor

/*
#cgo darwin LDFLAGS: -framework Hypervisor
#include <Hypervisor/hv_vcpu.h>
#include <Hypervisor/hv_vcpu_types.h>

// Helper to get ESR and FAR from the exit information
static hv_return_t go_hv_get_esr_far(hv_vcpu_t vcpu, uint64_t* esr, uint64_t* far) {
	// For ARM64, we would get this from the exit structure, but for now
	// try to get it from system registers
	hv_return_t r1 = hv_vcpu_get_sys_reg(vcpu, HV_SYS_REG_ESR_EL1, esr);
	hv_return_t r2 = hv_vcpu_get_sys_reg(vcpu, HV_SYS_REG_FAR_EL1, far);
	return (r1 != HV_SUCCESS) ? r1 : r2;
}
*/
import "C"

import (
	"fmt"
	"time"
)

// Run executes the vCPU until it exits. Returns ExitInfo best-effort.
func (c *VCPU) Run() (ExitInfo, error) {
	start := time.Now()
	defer func() {
		recordRun(time.Since(start))
	}()

	var info ExitInfo
	if c == nil {
		return info, fmt.Errorf("hv: VCPU is nil")
	}

	// Security: Lock to prevent use-after-free
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return info, fmt.Errorf("hv: VCPU is closed")
	}

	ret := C.hv_vcpu_run(C.hv_vcpu_t(c.id))
	if err := hvErr(ret); err != nil {
		recordResourceError()
		return info, fmt.Errorf("failed to run vCPU: %w", err)
	}
	var esr, far C.uint64_t
	if C.go_hv_get_esr_far(C.hv_vcpu_t(c.id), &esr, &far) == C.HV_SUCCESS {
		info.ESR = uint64(esr)
		info.FAR = uint64(far)
		if info.ESR != 0 {
			info.Reason = ExitException
		} else {
			info.Reason = ExitUnknown
		}
	} else {
		info.Reason = ExitUnknown
	}
	return info, nil
}
