// Package hypervisor provides Go bindings for Apple's Hypervisor.framework
// on Darwin ARM64 systems.
//
// Provides VM and vCPU management with memory mapping, register access,
// and execution control.
//
// # Requirements
//
//   - macOS with Apple Silicon (ARM64)
//   - Hypervisor entitlement: com.apple.security.hypervisor
//   - Code signing with entitlements
//
// # Basic Usage
//
// Check if hypervisor is supported:
//
//	supported, err := hypervisor.Supported()
//	if err != nil || !supported {
//		log.Fatal("Hypervisor not supported on this system")
//	}
//
// Create and manage a virtual machine:
//
//	// Create a new VM (only one VM per process is allowed)
//	vm, err := hypervisor.NewVM()
//	if err != nil {
//		log.Fatal("Failed to create VM:", err)
//	}
//	defer vm.Close()
//
//	// Create a virtual CPU
//	vcpu, err := vm.NewVCPU()
//	if err != nil {
//		log.Fatal("Failed to create vCPU:", err)
//	}
//	defer vcpu.Close()
//
// Memory management:
//
//	// Allocate and map guest memory (must be page-aligned)
//	hostMem := make([]byte, 4096) // 4KB page
//	guestPhys := uint64(0x4000)   // Guest physical address
//	perms := hypervisor.MemRead | hypervisor.MemWrite | hypervisor.MemExec
//
//	err = vm.Map(hostMem, guestPhys, perms)
//	if err != nil {
//		log.Fatal("Failed to map memory:", err)
//	}
//	defer vm.Unmap(guestPhys, uint64(len(hostMem)))
//
// Register access and execution:
//
//	// Set program counter to start execution
//	err = vcpu.SetPC(0x4000)
//	if err != nil {
//		log.Fatal("Failed to set PC:", err)
//	}
//
//	// Execute guest code until exit
//	exitInfo, err := vcpu.Run()
//	if err != nil {
//		log.Fatal("Failed to run vCPU:", err)
//	}
//
//	// Handle exit reason
//	switch exitInfo.Reason {
//	case hypervisor.ExitException:
//		fmt.Printf("Guest exception: ESR=0x%x FAR=0x%x\n", exitInfo.ESR, exitInfo.FAR)
//	case hypervisor.ExitUnknown:
//		fmt.Println("Unknown exit reason")
//	}
//
//	// Read register values
//	x0, err := vcpu.GetReg(hypervisor.RegX0)
//	if err != nil {
//		log.Fatal("Failed to get register:", err)
//	}
//	fmt.Printf("X0 register: 0x%x\n", x0)
//
// # Error Handling
//
// All errors implement the standard Go error interface. Hypervisor-specific
// errors are wrapped in HVError types with Apple Hypervisor.framework
// error codes.
//
// # Resource Management
//
// All resources (VMs and vCPUs) must be explicitly closed using Close().
// Finalizers provide safety net cleanup. Only one VM can exist per process.
//
// # Platform Support
//
// Darwin ARM64 only (Apple Silicon). Other platforms return "not supported" errors.
//
// # Code Signing and Entitlements
//
// Applications must be code signed with hypervisor entitlement:
//
//	<?xml version="1.0" encoding="UTF-8"?>
//	<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
//	    "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
//	<plist version="1.0">
//	<dict>
//	    <key>com.apple.security.hypervisor</key>
//	    <true/>
//	</dict>
//	</plist>
//
// Then sign your binary:
//
//	codesign --sign - --force --entitlements=hypervisor.entitlements ./your-app
package hypervisor
