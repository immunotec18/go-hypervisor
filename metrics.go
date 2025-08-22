//go:build darwin && arm64

package hypervisor

import (
	"sync/atomic"
	"time"
)

// Performance metrics for monitoring hypervisor operations
var (
	// Operation counters
	vmCreateCount    uint64
	vmDestroyCount   uint64
	vcpuCreateCount  uint64
	vcpuDestroyCount uint64
	mapOperations    uint64
	unmapOperations  uint64
	registerOps      uint64
	runOperations    uint64

	// Timing metrics (nanoseconds)
	totalVMCreateTime uint64
	totalRunTime      uint64

	// Error counters
	securityErrors uint64
	resourceErrors uint64
)

// Metrics provides access to performance metrics
type Metrics struct {
	VMCreated         uint64 `json:"vm_created"`
	VMDestroyed       uint64 `json:"vm_destroyed"`
	VCPUCreated       uint64 `json:"vcpu_created"`
	VCPUDestroyed     uint64 `json:"vcpu_destroyed"`
	MapOperations     uint64 `json:"map_operations"`
	UnmapOperations   uint64 `json:"unmap_operations"`
	RegisterOps       uint64 `json:"register_operations"`
	RunOperations     uint64 `json:"run_operations"`
	AvgVMCreateTimeNs uint64 `json:"avg_vm_create_time_ns"`
	AvgRunTimeNs      uint64 `json:"avg_run_time_ns"`
	SecurityErrors    uint64 `json:"security_errors"`
	ResourceErrors    uint64 `json:"resource_errors"`
}

// GetMetrics returns current performance metrics
func GetMetrics() Metrics {
	vmCreated := atomic.LoadUint64(&vmCreateCount)
	runOps := atomic.LoadUint64(&runOperations)

	var avgVMCreate, avgRun uint64
	if vmCreated > 0 {
		avgVMCreate = atomic.LoadUint64(&totalVMCreateTime) / vmCreated
	}
	if runOps > 0 {
		avgRun = atomic.LoadUint64(&totalRunTime) / runOps
	}

	return Metrics{
		VMCreated:         vmCreated,
		VMDestroyed:       atomic.LoadUint64(&vmDestroyCount),
		VCPUCreated:       atomic.LoadUint64(&vcpuCreateCount),
		VCPUDestroyed:     atomic.LoadUint64(&vcpuDestroyCount),
		MapOperations:     atomic.LoadUint64(&mapOperations),
		UnmapOperations:   atomic.LoadUint64(&unmapOperations),
		RegisterOps:       atomic.LoadUint64(&registerOps),
		RunOperations:     runOps,
		AvgVMCreateTimeNs: avgVMCreate,
		AvgRunTimeNs:      avgRun,
		SecurityErrors:    atomic.LoadUint64(&securityErrors),
		ResourceErrors:    atomic.LoadUint64(&resourceErrors),
	}
}

// ResetMetrics clears all performance metrics
func ResetMetrics() {
	atomic.StoreUint64(&vmCreateCount, 0)
	atomic.StoreUint64(&vmDestroyCount, 0)
	atomic.StoreUint64(&vcpuCreateCount, 0)
	atomic.StoreUint64(&vcpuDestroyCount, 0)
	atomic.StoreUint64(&mapOperations, 0)
	atomic.StoreUint64(&unmapOperations, 0)
	atomic.StoreUint64(&registerOps, 0)
	atomic.StoreUint64(&runOperations, 0)
	atomic.StoreUint64(&totalVMCreateTime, 0)
	atomic.StoreUint64(&totalRunTime, 0)
	atomic.StoreUint64(&securityErrors, 0)
	atomic.StoreUint64(&resourceErrors, 0)
}

// Internal metric recording functions
func recordVMCreate(duration time.Duration) {
	atomic.AddUint64(&vmCreateCount, 1)
	atomic.AddUint64(&totalVMCreateTime, uint64(duration.Nanoseconds()))
}

func recordVMDestroy() {
	atomic.AddUint64(&vmDestroyCount, 1)
}

func recordVCPUCreate() {
	atomic.AddUint64(&vcpuCreateCount, 1)
}

func recordVCPUDestroy() {
	atomic.AddUint64(&vcpuDestroyCount, 1)
}

func recordMapOperation() {
	atomic.AddUint64(&mapOperations, 1)
}

func recordUnmapOperation() {
	atomic.AddUint64(&unmapOperations, 1)
}

func recordRegisterOp() {
	atomic.AddUint64(&registerOps, 1)
}

func recordRun(duration time.Duration) {
	atomic.AddUint64(&runOperations, 1)
	atomic.AddUint64(&totalRunTime, uint64(duration.Nanoseconds()))
}

func recordSecurityError() {
	atomic.AddUint64(&securityErrors, 1)
}

func recordResourceError() {
	atomic.AddUint64(&resourceErrors, 1)
}
