//go:build !darwin || !arm64

package hypervisor

import "fmt"

// Supported returns false on non-Darwin platforms.
func Supported() (bool, error) {
	return false, fmt.Errorf("hypervisor: not supported on this platform")
}

// NewVM returns an error on non-Darwin platforms.
func NewVM() (*VM, error) {
	return nil, fmt.Errorf("hypervisor: not supported on this platform")
}

// Stub implementations for VM methods
func (vm *VM) Close() error {
	return fmt.Errorf("hypervisor: not supported on this platform")
}

func (vm *VM) Map(host []byte, guestPhys uint64, perms MemPerm) error {
	return fmt.Errorf("hypervisor: not supported on this platform")
}

func (vm *VM) Unmap(guestPhys, size uint64) error {
	return fmt.Errorf("hypervisor: not supported on this platform")
}

func (vm *VM) NewVCPU() (*VCPU, error) {
	return nil, fmt.Errorf("hypervisor: not supported on this platform")
}

// Stub implementations for VCPU methods
func (c *VCPU) Close() error {
	return fmt.Errorf("hypervisor: not supported on this platform")
}

func (c *VCPU) GetReg(r Reg) (uint64, error) {
	return 0, fmt.Errorf("hypervisor: not supported on this platform")
}

func (c *VCPU) SetReg(r Reg, v uint64) error {
	return fmt.Errorf("hypervisor: not supported on this platform")
}

func (c *VCPU) GetPC() (uint64, error) {
	return 0, fmt.Errorf("hypervisor: not supported on this platform")
}

func (c *VCPU) SetPC(v uint64) error {
	return fmt.Errorf("hypervisor: not supported on this platform")
}

func (c *VCPU) Run() (ExitInfo, error) {
	return ExitInfo{}, fmt.Errorf("hypervisor: not supported on this platform")
}
