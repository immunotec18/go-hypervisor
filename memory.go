//go:build darwin && arm64

package hypervisor

/*
#include <Hypervisor/hv.h>
#include <Hypervisor/hv_error.h>

#ifndef HV_MEMORY_READ
#define HV_MEMORY_READ (1<<0)
#endif
#ifndef HV_MEMORY_WRITE
#define HV_MEMORY_WRITE (1<<1)
#endif
#ifndef HV_MEMORY_EXEC
#define HV_MEMORY_EXEC (1<<2)
#endif

extern int hv_vm_map(void* uva, unsigned long long gpa, size_t size, int flags);
extern int hv_vm_unmap(unsigned long long gpa, size_t size);

// Wrapper to construct flags using framework macros without exposing values to Go.
static int go_hv_vm_map(void* addr, unsigned long long gpa, unsigned long long size, int r, int w, int x) {
	int flags = 0;
	if (r) flags |= HV_MEMORY_READ;
	if (w) flags |= HV_MEMORY_WRITE;
	if (x) flags |= HV_MEMORY_EXEC;
	return hv_vm_map(addr, gpa, (size_t)size, flags);
}

static int go_hv_vm_unmap(unsigned long long gpa, unsigned long long size) {
	return hv_vm_unmap(gpa, (size_t)size);
}
*/
import "C"

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

var (
	cachedPageSize int
	cachedPageMask uint64 // For fast alignment checks: addr & mask == 0
	pageSizeOnce   sync.Once
)

// pageSize returns the system page size, cached for performance
func pageSize() int {
	pageSizeOnce.Do(func() {
		cachedPageSize = unix.Getpagesize()
		cachedPageMask = uint64(cachedPageSize - 1)
	})
	return cachedPageSize
}

// isPageAligned returns true if addr is page-aligned (fast path)
func isPageAligned(addr uint64) bool {
	pageSizeOnce.Do(func() {
		cachedPageSize = unix.Getpagesize()
		cachedPageMask = uint64(cachedPageSize - 1)
	})
	return addr&cachedPageMask == 0
}

// Map maps a host memory slice into the guest physical address space.
// The host slice base address, length, and guestPhys must be page-aligned.
func (vm *VM) Map(host []byte, guestPhys uint64, perms MemPerm) error {
	if vm == nil {
		return fmt.Errorf("hv: VM is nil")
	}
	if vm.closed {
		return fmt.Errorf("hv: VM is closed")
	}
	if len(host) == 0 {
		return fmt.Errorf("hv: map requires non-empty host buffer")
	}

	// Security: Prevent integer overflow vulnerabilities
	if len(host) > math.MaxInt32 {
		return fmt.Errorf("hv: host buffer too large (max %d bytes)", math.MaxInt32)
	}
	if guestPhys > math.MaxUint64-uint64(len(host)) {
		return fmt.Errorf("hv: guest address range would overflow")
	}

	// Validate permissions - must have at least one permission set
	if perms == 0 {
		return fmt.Errorf("hv: map requires at least one permission (read, write, or exec)")
	}
	// Check for invalid permission bits
	validPerms := MemRead | MemWrite | MemExec
	if perms&^validPerms != 0 {
		return fmt.Errorf("hv: invalid permission bits 0x%x (valid: 0x%x)", perms, validPerms)
	}

	// Performance: Fast alignment checks using cached masks
	if !isPageAligned(guestPhys) {
		return fmt.Errorf("hv: guestPhys not page-aligned: 0x%x (page size: %d)", guestPhys, pageSize())
	}
	if !isPageAligned(uint64(len(host))) {
		return fmt.Errorf("hv: host length not page multiple: %d (page size: %d)", len(host), pageSize())
	}
	// Pin the memory before passing to C to prevent GC from moving it
	runtime.KeepAlive(host)
	defer runtime.KeepAlive(host)

	ptr := unsafe.Pointer(&host[0])
	if !isPageAligned(uint64(uintptr(ptr))) {
		return fmt.Errorf("hv: host base not page-aligned: %p (page size: %d)", ptr, pageSize())
	}
	read := 0
	write := 0
	exec := 0
	if perms&MemRead != 0 {
		read = 1
	}
	if perms&MemWrite != 0 {
		write = 1
	}
	if perms&MemExec != 0 {
		exec = 1
	}
	ret := C.go_hv_vm_map(ptr, C.ulonglong(guestPhys), C.ulonglong(uint64(len(host))), C.int(read), C.int(write), C.int(exec))
	if err := hvErr(ret); err != nil {
		recordResourceError()
		return fmt.Errorf("failed to map %d bytes at 0x%x with perms 0x%x: %w", len(host), guestPhys, perms, err)
	}

	recordMapOperation()
	return nil
}

// Unmap removes a region from the guest physical address space.
func (vm *VM) Unmap(guestPhys, size uint64) error {
	if vm == nil {
		return fmt.Errorf("hv: VM is nil")
	}
	if vm.closed {
		return fmt.Errorf("hv: VM is closed")
	}
	if size == 0 {
		return fmt.Errorf("hv: unmap requires non-zero size")
	}

	// Security: Prevent integer overflow vulnerabilities
	if size > math.MaxInt32 {
		return fmt.Errorf("hv: unmap size too large (max %d bytes)", math.MaxInt32)
	}
	if guestPhys > math.MaxUint64-size {
		return fmt.Errorf("hv: guest address range would overflow")
	}

	// Performance: Fast alignment checks using cached masks
	if !isPageAligned(guestPhys) {
		return fmt.Errorf("hv: guestPhys not page-aligned: 0x%x (page size: %d)", guestPhys, pageSize())
	}
	if !isPageAligned(size) {
		return fmt.Errorf("hv: size not page multiple: %d (page size: %d)", size, pageSize())
	}

	ret := C.go_hv_vm_unmap(C.ulonglong(guestPhys), C.ulonglong(size))
	if err := hvErr(ret); err != nil {
		recordResourceError()
		return fmt.Errorf("failed to unmap region 0x%x+%d: %w", guestPhys, size, err)
	}

	recordUnmapOperation()
	return nil
}
