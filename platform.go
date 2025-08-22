//go:build darwin && arm64

package hypervisor

import (
	"golang.org/x/sys/unix"
)

// Supported returns true if the hypervisor is available and accessible.
func Supported() (bool, error) {
	supported, err := unix.SysctlUint32("kern.hv_support")
	if err != nil {
		return false, err
	}
	return supported != 0, nil
}
