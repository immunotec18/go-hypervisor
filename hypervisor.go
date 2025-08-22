//go:build darwin && arm64

package hypervisor

/*
#cgo darwin LDFLAGS: -framework Hypervisor
#include <Hypervisor/hv.h>
#include <Hypervisor/hv_error.h>
#include <Hypervisor/hv_vm.h>
#include <Hypervisor/hv_vm_config.h>
#include <Hypervisor/hv_base.h>
#include <Hypervisor/hv_vcpu.h>
#include <Hypervisor/hv_vcpu_config.h>
#include <os/object.h>
#if __has_include(<Hypervisor/arm64/hv_arch_vcpu.h>)
#include <Hypervisor/arm64/hv_arch_vcpu.h>
#endif
#if __has_include(<Hypervisor/arm64/hv_arch_vtimer.h>)
#include <Hypervisor/arm64/hv_arch_vtimer.h>
#endif

// Helper function to create and configure a VM with proper error handling
static hv_return_t go_hv_vm_create_with_cfg() {
#if __has_include(<Hypervisor/hv_vm_config.h>)
	hv_vm_config_t config = hv_vm_config_create();
	if (!config) {
		return HV_ERROR;
	}

	// Get and set default IPA size
	uint32_t default_ipa_size = 0;
	hv_return_t ret = hv_vm_config_get_default_ipa_size(&default_ipa_size);
	if (ret == HV_SUCCESS) {
		ret = hv_vm_config_set_ipa_size(config, default_ipa_size);
		if (ret != HV_SUCCESS) {
			os_release(config);
			return ret;
		}
	}

	// Create the VM with the configuration
	ret = hv_vm_create(config);
	os_release(config);
	return ret;
#else
	// Fallback for older macOS versions without hv_vm_config
	return hv_vm_create(NULL);
#endif
}

// Helper function to create a vCPU with proper ARM64 API
static hv_return_t go_hv_vcpu_create(hv_vcpu_t *vcpu, hv_vcpu_exit_t **exit) {
	// For now, create with NULL config (uses defaults)
	return hv_vcpu_create(vcpu, exit, NULL);
}
*/
import "C"

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// MemPerm represents guest memory permissions.
type MemPerm uint

const (
	MemRead  MemPerm = 1 << 0
	MemWrite MemPerm = 1 << 1
	MemExec  MemPerm = 1 << 2
)

// Reg represents an ARM64 general/system register.
type Reg int

const (
	RegX0 Reg = iota
	RegX1
	RegX2
	RegX3
	RegX4
	RegX5
	RegX6
	RegX7
	RegX8
	RegX9
	RegX10
	RegX11
	RegX12
	RegX13
	RegX14
	RegX15
	RegX16
	RegX17
	RegX18
	RegX19
	RegX20
	RegX21
	RegX22
	RegX23
	RegX24
	RegX25
	RegX26
	RegX27
	RegX28
	RegFP // X29
	RegLR // X30
	RegSP // Stack pointer (SP_EL0)
	RegPC
	RegCPSR
)

// ExitReason categorizes vCPU exits.
type ExitReason int

const (
	ExitUnknown ExitReason = iota
	ExitException
	ExitTimer
)

// ExitInfo captures information about a recent vCPU exit.
type ExitInfo struct {
	Reason ExitReason
	ESR    uint64
	FAR    uint64
}

// VM represents a single hypervisor VM instance.
type VM struct {
	closed  bool
	closeMu sync.Mutex // Protect against concurrent Close() and finalizer
}

// VCPU represents a single vCPU associated with a VM.
type VCPU struct {
	id      uint64
	closed  bool
	closeMu sync.Mutex // Protect against concurrent Close() and finalizer
}

var (
	vmMu     sync.RWMutex // Use RWMutex for better read performance
	vmActive bool
	vmCount  int32 // Atomic counter for debugging
)

// NewVM creates a new Hypervisor VM for this process.
func NewVM() (*VM, error) {
	start := time.Now()
	defer func() {
		recordVMCreate(time.Since(start))
	}()

	vmMu.Lock()
	defer vmMu.Unlock()

	// Security: Double-check to prevent race conditions
	if vmActive {
		recordResourceError()
		return nil, ErrVMAlreadyActive
	}

	ret := C.go_hv_vm_create_with_cfg()
	if err := hvErr(ret); err != nil {
		recordResourceError()
		return nil, err
	}

	// Security: Atomic updates to prevent race conditions
	vmActive = true
	atomic.AddInt32(&vmCount, 1)
	vm := &VM{closed: false}

	// Set finalizer as safety net in case Close() is not called
	runtime.SetFinalizer(vm, (*VM).finalize)

	return vm, nil
}

// Close destroys the Hypervisor VM. Idempotent.
func (vm *VM) Close() error {
	if vm == nil {
		return nil
	}

	// Security: Lock instance first to prevent finalizer race
	vm.closeMu.Lock()
	defer vm.closeMu.Unlock()

	if vm.closed {
		return nil // Already closed
	}

	vmMu.Lock()
	defer vmMu.Unlock()

	// Security: Check global state under lock
	if !vmActive {
		return nil
	}

	ret := C.hv_vm_destroy()
	if err := hvErr(ret); err != nil {
		return fmt.Errorf("failed to destroy VM: %w", err)
	}

	// Security: Atomic updates to prevent race conditions
	vm.closed = true
	vmActive = false
	atomic.AddInt32(&vmCount, -1)

	// Clear finalizer since we've cleaned up properly
	runtime.SetFinalizer(vm, nil)

	recordVMDestroy()
	return nil
}

// finalize is called by the garbage collector as a safety net
func (vm *VM) finalize() {
	if vm == nil {
		return
	}
	// Security: Use non-blocking lock to prevent deadlock in finalizers
	if vm.closeMu.TryLock() {
		defer vm.closeMu.Unlock()
		if !vm.closed {
			// Direct cleanup without calling Close() to avoid potential deadlocks
			// Mark as closed first
			vm.closed = true

			// Best effort cleanup of hypervisor resources
			if vmActive {
				C.hv_vm_destroy()
				vmActive = false
				atomic.AddInt32(&vmCount, -1)
			}
		}
	}
}

// NewVCPU creates a new vCPU for the active VM.
func (vm *VM) NewVCPU() (*VCPU, error) {
	if vm == nil {
		return nil, fmt.Errorf("hv: VM is nil")
	}
	var vcpu C.hv_vcpu_t
	var exit *C.hv_vcpu_exit_t
	ret := C.go_hv_vcpu_create(&vcpu, &exit)
	if err := hvErr(ret); err != nil {
		return nil, err
	}

	c := &VCPU{id: uint64(vcpu), closed: false}

	// Set finalizer as safety net in case Close() is not called
	runtime.SetFinalizer(c, (*VCPU).finalize)

	recordVCPUCreate()
	return c, nil
}

// Close destroys this vCPU.
func (c *VCPU) Close() error {
	if c == nil {
		return nil
	}

	// Security: Lock instance to prevent finalizer race
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil // Already closed
	}

	ret := C.hv_vcpu_destroy(C.hv_vcpu_t(c.id))
	if err := hvErr(ret); err != nil {
		return fmt.Errorf("failed to destroy vCPU: %w", err)
	}

	c.closed = true

	// Clear finalizer since we've cleaned up properly
	runtime.SetFinalizer(c, nil)

	recordVCPUDestroy()
	return nil
}

// finalize is called by the garbage collector as a safety net
func (c *VCPU) finalize() {
	if c == nil {
		return
	}
	// Security: Use non-blocking lock to prevent deadlock in finalizers
	if c.closeMu.TryLock() {
		defer c.closeMu.Unlock()
		if !c.closed {
			// Log that we're using the finalizer (indicates a resource leak)
			// Note: We can't use log package here as it might cause issues in finalizers
			c.Close() // Best effort cleanup
		}
	}
}
