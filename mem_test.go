//go:build darwin && arm64 && hypervisor

package hypervisor

import (
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

func TestMemPermConstants(t *testing.T) {
	// Test that our MemPerm constants are correct
	if MemRead != 1<<0 {
		t.Errorf("MemRead = %d, want %d", MemRead, 1<<0)
	}
	if MemWrite != 1<<1 {
		t.Errorf("MemWrite = %d, want %d", MemWrite, 1<<1)
	}
	if MemExec != 1<<2 {
		t.Errorf("MemExec = %d, want %d", MemExec, 1<<2)
	}

	// Test combinations
	readWrite := MemRead | MemWrite
	if readWrite != 3 {
		t.Errorf("MemRead|MemWrite = %d, want 3", readWrite)
	}

	rwx := MemRead | MemWrite | MemExec
	if rwx != 7 {
		t.Errorf("MemRead|MemWrite|MemExec = %d, want 7", rwx)
	}
}

func TestPageSize(t *testing.T) {
	ps := pageSize()
	expectedPS := unix.Getpagesize()

	if ps != expectedPS {
		t.Errorf("pageSize() = %d, want %d", ps, expectedPS)
	}

	// On Apple Silicon, page size should typically be 16KB
	if ps != 4096 && ps != 16384 {
		t.Logf("Unexpected page size: %d (expected 4K or 16K)", ps)
	}
}

func TestMemoryMapValidation(t *testing.T) {
	// These tests don't require actual hypervisor access, just validation logic
	supported, err := Supported()
	if err != nil {
		t.Fatalf("Failed to check hypervisor support: %v", err)
	}
	if !supported {
		t.Skip("Hypervisor not supported - skipping memory map validation tests")
	}

	vm, err := NewVM()
	if err != nil {
		t.Skipf("Cannot create VM (likely missing entitlements): %v", err)
	}
	defer vm.Close()

	pageSize := unix.Getpagesize()

	t.Run("nil VM", func(t *testing.T) {
		var nilVM *VM
		err := nilVM.Map(make([]byte, pageSize), 0x4000, MemRead)
		if err == nil {
			t.Error("Expected error for nil VM, got nil")
		}
	})

	t.Run("empty host buffer", func(t *testing.T) {
		err := vm.Map([]byte{}, 0x4000, MemRead)
		if err == nil {
			t.Error("Expected error for empty host buffer, got nil")
		}
		if err != nil && err.Error() != "hv: map requires non-empty host buffer" {
			t.Errorf("Wrong error message: %v", err)
		}
	})

	t.Run("unaligned guest address", func(t *testing.T) {
		alignedBuffer := make([]byte, pageSize)
		unalignedGuestAddr := uint64(0x4001) // Not page aligned

		err := vm.Map(alignedBuffer, unalignedGuestAddr, MemRead)
		if err == nil {
			t.Error("Expected error for unaligned guest address, got nil")
		}
	})

	t.Run("unaligned host buffer size", func(t *testing.T) {
		unalignedBuffer := make([]byte, pageSize+1) // Not page multiple

		err := vm.Map(unalignedBuffer, 0x4000, MemRead)
		if err == nil {
			t.Error("Expected error for unaligned buffer size, got nil")
		}
	})

	t.Run("unaligned host buffer address", func(t *testing.T) {
		// Create a larger buffer and use an unaligned slice
		largeBuffer := make([]byte, pageSize*2)
		unalignedSlice := largeBuffer[1 : pageSize+1] // Starts at offset 1

		err := vm.Map(unalignedSlice, 0x4000, MemRead)
		if err == nil {
			t.Error("Expected error for unaligned host buffer address, got nil")
		}
	})

	t.Run("valid aligned mapping", func(t *testing.T) {
		// Create properly aligned buffer
		alignedBuffer := make([]byte, pageSize)

		// Ensure the buffer is page-aligned (Go's allocator usually does this for large allocations)
		if uintptr(unsafe.Pointer(&alignedBuffer[0]))%uintptr(pageSize) != 0 {
			t.Skip("Cannot create page-aligned buffer in this test environment")
		}

		err := vm.Map(alignedBuffer, 0x4000, MemRead|MemWrite|MemExec)
		if err != nil {
			// This might fail due to other reasons (like missing entitlements),
			// but it shouldn't be an alignment error
			if err.Error() == "hv: denied (HV_DENIED)" {
				t.Skip("Mapping denied - likely insufficient entitlements")
			}
			t.Errorf("Unexpected error for valid mapping: %v", err)
		} else {
			// If mapping succeeded, test unmapping
			defer vm.Unmap(0x4000, uint64(pageSize))
		}
	})
}

func TestMemoryUnmapValidation(t *testing.T) {
	supported, err := Supported()
	if err != nil {
		t.Fatalf("Failed to check hypervisor support: %v", err)
	}
	if !supported {
		t.Skip("Hypervisor not supported - skipping memory unmap validation tests")
	}

	vm, err := NewVM()
	if err != nil {
		t.Skipf("Cannot create VM (likely missing entitlements): %v", err)
	}
	defer vm.Close()

	pageSize := uint64(unix.Getpagesize())

	t.Run("nil VM", func(t *testing.T) {
		var nilVM *VM
		err := nilVM.Unmap(0x4000, pageSize)
		if err == nil {
			t.Error("Expected error for nil VM, got nil")
		}
	})

	t.Run("unaligned guest address", func(t *testing.T) {
		err := vm.Unmap(0x4001, pageSize) // Unaligned address
		if err == nil {
			t.Error("Expected error for unaligned guest address, got nil")
		}
	})

	t.Run("unaligned size", func(t *testing.T) {
		err := vm.Unmap(0x4000, pageSize+1) // Unaligned size
		if err == nil {
			t.Error("Expected error for unaligned size, got nil")
		}
	})

	t.Run("valid aligned unmap", func(t *testing.T) {
		// This should not error on alignment, though it might fail for other reasons
		err := vm.Unmap(0x8000, pageSize)
		// We expect this to potentially fail because we haven't mapped anything,
		// but it shouldn't be an alignment error
		if err != nil {
			t.Logf("Unmap error (expected): %v", err)
		}
	})
}

func TestMemoryPermissions(t *testing.T) {
	// Test various permission combinations
	testCases := []struct {
		name  string
		perms MemPerm
	}{
		{"read-only", MemRead},
		{"read-write", MemRead | MemWrite},
		{"read-execute", MemRead | MemExec},
		{"read-write-execute", MemRead | MemWrite | MemExec},
		{"write-only", MemWrite},  // Might not be valid, but test validation
		{"execute-only", MemExec}, // Might not be valid, but test validation
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Just test that permission constants work as expected
			hasRead := (tc.perms & MemRead) != 0
			hasWrite := (tc.perms & MemWrite) != 0
			hasExec := (tc.perms & MemExec) != 0

			t.Logf("Permissions %v: Read=%v Write=%v Exec=%v", tc.perms, hasRead, hasWrite, hasExec)

			if tc.name == "read-only" && (!hasRead || hasWrite || hasExec) {
				t.Error("read-only permissions incorrect")
			}
			if tc.name == "read-write-execute" && (!hasRead || !hasWrite || !hasExec) {
				t.Error("read-write-execute permissions incorrect")
			}
		})
	}
}
