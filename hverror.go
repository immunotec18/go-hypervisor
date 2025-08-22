package hypervisor

/*
#include <arm64/hv/hv_kern_types.h>
*/
import "C"
import (
	"fmt"
	"os"
	"strconv"
	"sync"
)

// Performance: Pre-allocated error message pools
var (
	errorMsgPool = sync.Pool{
		New: func() any {
			return make([]byte, 0, 256) // Pre-allocate 256 bytes for error formatting
		},
	}
)

// Hypervisor Framework hv_return_t constants for ARM64
const (
	HV_SUCCESS             uint32 = 0x00000000
	HV_ERROR               uint32 = 0xFAE94001
	HV_BUSY                uint32 = 0xFAE94002
	HV_BAD_ARGUMENT        uint32 = 0xFAE94003
	HV_ILLEGAL_GUEST_STATE uint32 = 0xFAE94004
	HV_NO_RESOURCES        uint32 = 0xFAE94005
	HV_NO_DEVICE           uint32 = 0xFAE94006
	HV_DENIED              uint32 = 0xFAE94007
	HV_EXISTS              uint32 = 0xFAE94008
	HV_UNSUPPORTED         uint32 = 0xFAE9400F
)

// HVError wraps an hv_return_t error code.
// Code stores the raw 32-bit hv_return_t value (often 0xFAE940xx).
type HVError struct {
	Code    uint32
	message string // Optional custom message for specific errors
}

func (e HVError) Error() string {
	// Use custom message if available
	if e.message != "" {
		return e.message
	}

	// Security: Check if we should sanitize error messages
	if isProductionEnv() {
		return e.sanitizedError()
	}
	return e.detailedError()
}

// detailedError provides full error context for development
func (e HVError) detailedError() string {
	switch e.Code {
	case HV_SUCCESS:
		return "hv: success"
	case HV_ERROR:
		return "hv: general error (HV_ERROR) - check system requirements and API usage"
	case HV_BUSY:
		return "hv: resource busy (HV_BUSY) - another operation is in progress"
	case HV_BAD_ARGUMENT:
		return "hv: invalid argument (HV_BAD_ARGUMENT) - check parameter values and alignment"
	case HV_ILLEGAL_GUEST_STATE:
		return "hv: illegal guest state (HV_ILLEGAL_GUEST_STATE) - guest CPU state is invalid"
	case HV_NO_RESOURCES:
		return "hv: insufficient resources (HV_NO_RESOURCES) - system memory or limits exceeded"
	case HV_NO_DEVICE:
		return "hv: device not found (HV_NO_DEVICE) - hardware virtualization unavailable"
	case HV_DENIED:
		return "hv: access denied (HV_DENIED) - missing entitlement 'com.apple.security.hypervisor' or insufficient privileges"
	case HV_EXISTS:
		return "hv: resource exists (HV_EXISTS) - VM or vCPU already created"
	case HV_UNSUPPORTED:
		return "hv: operation unsupported (HV_UNSUPPORTED) - feature not available on this hardware/OS"
	default:
		return fmt.Sprintf("hv: unknown error code 0x%08x - consult Apple Hypervisor.framework documentation", e.Code)
	}
}

// sanitizedError provides minimal error information for production
func (e HVError) sanitizedError() string {
	switch e.Code {
	case HV_SUCCESS:
		return "hv: success"
	case HV_ERROR:
		return "hv: general error"
	case HV_BUSY:
		return "hv: resource busy"
	case HV_BAD_ARGUMENT:
		return "hv: invalid argument"
	case HV_ILLEGAL_GUEST_STATE:
		return "hv: illegal guest state"
	case HV_NO_RESOURCES:
		return "hv: insufficient resources"
	case HV_NO_DEVICE:
		return "hv: device not found"
	case HV_DENIED:
		return "hv: access denied"
	case HV_EXISTS:
		return "hv: resource exists"
	case HV_UNSUPPORTED:
		return "hv: operation unsupported"
	default:
		return "hv: hypervisor error"
	}
}

// isProductionEnv checks if we're running in production environment
func isProductionEnv() bool {
	env := os.Getenv("HV_ENV")
	if env == "production" || env == "prod" {
		return true
	}

	// Check if debug mode is explicitly disabled
	if debug := os.Getenv("HV_DEBUG"); debug != "" {
		if val, err := strconv.ParseBool(debug); err == nil && !val {
			return true
		}
	}

	return false
}

func hvErr(code C.hv_return_t) error {
	if code == 0 {
		return nil
	}
	return HVError{Code: uint32(code)}
}

// Common specific errors for API consumers
var (
	ErrVMClosed         = &HVError{Code: HV_ERROR, message: "hv: VM is closed"}
	ErrVCPUClosed       = &HVError{Code: HV_ERROR, message: "hv: VCPU is closed"}
	ErrInvalidAlignment = &HVError{Code: HV_BAD_ARGUMENT, message: "hv: address not page-aligned"}
	ErrInvalidRegister  = &HVError{Code: HV_BAD_ARGUMENT, message: "hv: invalid register"}
	ErrMemoryNotMapped  = &HVError{Code: HV_BAD_ARGUMENT, message: "hv: memory not mapped"}
	ErrVMAlreadyActive  = &HVError{Code: HV_BUSY, message: "hv: VM already active in this process"}
)
