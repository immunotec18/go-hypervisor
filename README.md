<p align="center">
  <a href="https://github.com/blacktop/go-hypervisor"><img alt="go-hypervisor Logo" src="./docs/logo.png" width=300/></a>
  <h1 align="center">go-hypervisor</h1>
  <h4><p align="center">Apple Hypervisor.framework bindings for Golang</p></h4>
  <p align="center">
    <a href="https://github.com/blacktop/go-hypervisor/actions" alt="Actions">
          <img src="https://github.com/blacktop/go-hypervisor/actions/workflows/go.yml/badge.svg" /></a>
    <a href="https://github.com/blacktop/go-hypervisor/releases/latest" alt="Downloads">
          <img src="https://img.shields.io/github/downloads/blacktop/go-hypervisor/total.svg" /></a>
    <a href="https://github.com/blacktop/go-hypervisor/releases" alt="GitHub Release">
          <img src="https://img.shields.io/github/release/blacktop/go-hypervisor.svg" /></a>
    <a href="http://doge.mit-license.org" alt="LICENSE">
          <img src="https://img.shields.io/:license-mit-blue.svg" /></a>
</p>
<br>

## Why? 🤔

🤷 `unicorn` wasn't working.

## Requirements

* **macOS 26 Tahoe** (beta) or later
* **Apple Silicon** (M1/M2/M3/M4 series) 
* **Hypervisor entitlement**: `com.apple.security.hypervisor`
* **Code signing** with appropriate entitlements

## Getting Started

```bash
go get github.com/blacktop/go-hypervisor
```

### Basic Usage

```go
package main

import (
    "encoding/binary"
    "fmt"
    "log"
    
    "github.com/blacktop/go-hypervisor"
    "golang.org/x/sys/unix"
)

func main() {
    // Check if hypervisor is supported
    supported, err := hypervisor.Supported()
    if err != nil || !supported {
        log.Fatal("Hypervisor not supported on this system")
    }

    // Create a new VM (only one VM per process allowed)
    vm, err := hypervisor.NewVM()
    if err != nil {
        log.Fatal("Failed to create VM:", err)
    }
    defer vm.Close()

    // Create a virtual CPU
    vcpu, err := vm.NewVCPU()
    if err != nil {
        log.Fatal("Failed to create vCPU:", err)
    }
    defer vcpu.Close()

    // Allocate and map guest memory (must be page-aligned)
    page := unix.Getpagesize()
    hostMem, err := unix.Mmap(-1, 0, page, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
    if err != nil {
        log.Fatal("Failed to allocate memory:", err)
    }
    defer unix.Munmap(hostMem)

    // Write ARM64 assembly: mov x0, #0x42; brk #0
    binary.LittleEndian.PutUint32(hostMem[0:], 0xD2800840) // MOVZ X0,#0x42
    binary.LittleEndian.PutUint32(hostMem[4:], 0xD4200000) // BRK #0

    // Map host memory into guest physical address space
    guestPhys := uint64(0x4000)
    perms := hypervisor.MemRead | hypervisor.MemWrite | hypervisor.MemExec
    err = vm.Map(hostMem, guestPhys, perms)
    if err != nil {
        log.Fatal("Failed to map memory:", err)
    }
    defer vm.Unmap(guestPhys, uint64(len(hostMem)))

    // Set program counter and execute
    err = vcpu.SetPC(guestPhys)
    if err != nil {
        log.Fatal("Failed to set PC:", err)
    }

    // Execute guest code
    exitInfo, err := vcpu.Run()
    if err != nil {
        log.Fatal("Failed to run vCPU:", err)
    }

    // Read result
    x0, err := vcpu.GetReg(hypervisor.RegX0)
    if err != nil {
        log.Fatal("Failed to get register:", err)
    }

    fmt.Printf("Exit: reason=%v ESR=0x%x FAR=0x%x\n", exitInfo.Reason, exitInfo.ESR, exitInfo.FAR)
    fmt.Printf("X0=0x%x\n", x0) // Should print: X0=0x42
}
```

### Performance Monitoring

Built-in performance metrics:

```go
// Get performance metrics
metrics := hypervisor.GetMetrics()
fmt.Printf("VM Operations: %d created, %d destroyed\n", metrics.VMCreated, metrics.VMDestroyed)
fmt.Printf("Average VM creation time: %d ns\n", metrics.AvgVMCreateTimeNs)
fmt.Printf("Register operations: %d\n", metrics.RegisterOps)
```

### Batch Register Operations

Batch register operations for performance:

```go
// Read multiple registers efficiently
regs := []hypervisor.Reg{hypervisor.RegX0, hypervisor.RegX1, hypervisor.RegX2, hypervisor.RegPC}
batch, err := vcpu.GetRegs(regs)
if err != nil {
    log.Fatal(err)
}

// Modify and write back
batch[hypervisor.RegX1] = 0x1337
err = vcpu.SetRegs(batch)
```

## Code Signing & Entitlements

Your application must be code signed with hypervisor entitlements. Create `hypervisor.entitlements`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" 
    "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.security.hypervisor</key>
    <true/>
</dict>
</plist>
```

Then sign your binary:

```bash
codesign --sign - --force --entitlements=hypervisor.entitlements ./your-app
```

## Security

- Production-safe error messages: Set `HV_ENV=production` to sanitize error output
- Memory safety: Protection against integer overflow and use-after-free
- Resource management: Automatic cleanup with finalizers as safety nets
- Input validation: Comprehensive bounds checking and parameter validation

## CLI tool `hv`

Install with Homebrew:

```bash
brew install blacktop/tap/go-hypervisor
```

Install with Go:

```bash
go install github.com/blacktop/go-hypervisor/cmd/hv@latest
```

Or download from the latest [release](https://github.com/blacktop/go-hypervisor/releases/latest)

### CLI Examples

#### Check Hypervisor Support

```bash
hv check
```

#### Execute Raw Machine Code

```bash
# Execute from binary file
hv execute code.bin

# Execute from stdin with initial state
echo -ne '\x40\x08\x80\xd2\x00\x00\x20\xd4' | hv execute --state initial.json

# Custom memory size and base address
hv execute --mem-size 32768 --base-addr 0x8000 code.bin
```

Example initial state file (`initial.json`):
```json
{
  "x0": 100,
  "x1": 200,
  "sp": 32768
}
```

#### Emulate Mach-O Functions

```bash
# Emulate main function (default entry point)
hv emulate ./my-binary

# Emulate specific function by address
hv emulate --addr 0x1000003e8 ./my-binary

# Custom stack pointer and memory size
hv emulate --stack 0x8000 --mem 65536 --addr 0x1000003e8 ./my-binary
```

Example output showing stack string building:
```
Emulating function at address: 0x1000003e8
Function: _build_stack_string (0x1000003e8 - 0x10000043c, 84 bytes)

=== Execution Results ===
Exit Reason: 0
Final SP: 0x8000 (moved 0 bytes)

Registers:
  X0=0x7ff0  X1=0x0  X2=0x0  X3=0x0
  PC=0x0  SP=0x8000  FP=0x0  LR=0x0

=== Stack Analysis ===
Stack region: 0x7fc0 - 0x8040 (Initial SP: 0x8000, Final SP: 0x8000)
Stack change: 0 bytes

     0000000000007ff0:  48 65 6c 6c 6f 20 48 56  21 00 00 00 00 00 00 00  |Hello HV!.......|
ISP> 0000000000008000:  00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |................|
```

#### Using with jq for JSON Processing

```bash
# Extract specific register values
hv execute code.bin | jq '.state.x0'

# Get exit information
hv execute code.bin | jq '.exit_info'

# Check if execution succeeded
hv execute code.bin | jq -r 'if .error then "FAILED: " + .error else "SUCCESS" end'
```

## License

MIT Copyright (c) 2025 **blacktop**