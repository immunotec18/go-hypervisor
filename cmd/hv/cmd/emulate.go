/*
Copyright © 2025 blacktop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"

	"github.com/blacktop/go-hypervisor"
	"github.com/blacktop/go-hypervisor/cmd/hv/cmd/utils"
	"github.com/blacktop/go-macho"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

func init() {
	rootCmd.AddCommand(emulateCmd)
	emulateCmd.Flags().Uint64P("addr", "a", 0, "Address to emulate (0 = use entry point)")
	emulateCmd.Flags().IntP("mem-size", "m", 0x10000, "Memory size to allocate (bytes)")
	emulateCmd.Flags().Uint64P("stack", "s", 0x8000, "Stack pointer address (within allocated memory)")
}

var emulateCmd = &cobra.Command{
	Use:     "emulate [FILE]",
	Aliases: []string{"emu"},
	Short:   "Emulate a function from a Mach-O binary and show stack contents",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check hypervisor support
		ok, err := hypervisor.Supported()
		if err != nil || !ok {
			return fmt.Errorf("hypervisor not supported: %v", err)
		}

		// Get flags
		addr, err := cmd.Flags().GetUint64("addr")
		if err != nil {
			return err
		}

		memSize, err := cmd.Flags().GetInt("mem-size")
		if err != nil {
			return err
		}

		// Validate memory size is page-aligned
		page := unix.Getpagesize()
		if memSize%page != 0 {
			return fmt.Errorf("mem-size must be a multiple of page size (%d bytes)", page)
		}

		stackPtr, err := cmd.Flags().GetUint64("stack")
		if err != nil {
			return err
		}

		// Validate stack pointer is within memory range
		baseAddr := uint64(0x4000) // Base address from execute command
		if stackPtr < baseAddr || stackPtr >= baseAddr+uint64(memSize) {
			return fmt.Errorf("stack pointer 0x%x must be within memory range 0x%x-0x%x",
				stackPtr, baseAddr, baseAddr+uint64(memSize))
		}

		// Open Mach-O file
		m, err := macho.Open(args[0])
		if err != nil {
			return fmt.Errorf("failed to open Mach-O file: %w", err)
		}
		defer m.Close()

		// Determine address to emulate
		if addr == 0 {
			if main := m.GetLoadsByName("LC_MAIN"); len(main) == 0 {
				return fmt.Errorf("failed to find LC_MAIN in target - use --addr to specify function address")
			} else {
				addr = main[0].(*macho.EntryPoint).EntryOffset + m.GetBaseAddress()
			}
		}

		fmt.Printf("Emulating function at address: 0x%x\n", addr)

		// Get function boundaries
		fn, err := m.GetFunctionForVMAddr(addr)
		if err != nil {
			return fmt.Errorf("failed to find function at address 0x%x: %w", addr, err)
		}

		fmt.Printf("Function: %s (0x%x - 0x%x, %d bytes)\n",
			fn.Name, fn.StartAddr, fn.EndAddr, fn.EndAddr-fn.StartAddr)

		// Extract function bytes
		instrs := make([]byte, fn.EndAddr-fn.StartAddr)
		if _, err := m.ReadAtAddr(instrs, fn.StartAddr); err != nil {
			return fmt.Errorf("failed to read function bytes: %w", err)
		}

		// Add brk instruction at the end to ensure proper exit
		instrs = append(instrs, 0x00, 0x00, 0x20, 0xd4) // brk #0

		// Execute the function
		result, err := emulateFunction(instrs, stackPtr, memSize)
		if err != nil {
			return fmt.Errorf("emulation failed: %w", err)
		}

		// Print results
		fmt.Printf("\n=== Execution Results ===\n")
		fmt.Printf("Exit Reason: %v\n", result.ExitInfo.Reason)
		fmt.Printf("Final SP: 0x%x (moved %d bytes)\n",
			result.State.SP, int64(result.State.SP)-int64(stackPtr))

		fmt.Printf("\nRegisters:\n")
		fmt.Printf("  X0=0x%x  X1=0x%x  X2=0x%x  X3=0x%x\n",
			result.State.X0, result.State.X1, result.State.X2, result.State.X3)
		fmt.Printf("  PC=0x%x  SP=0x%x  FP=0x%x  LR=0x%x\n",
			result.State.PC, result.State.SP, result.State.FP, result.State.LR)

		// Print stack contents
		printStackContents(result.Memory, baseAddr, stackPtr, result.State.SP)

		return nil
	},
}

// emulateFunction executes the function bytes and returns the result
func emulateFunction(code []byte, stackPtr uint64, memSize int) (*ExecuteResult, error) {
	// Create VM
	vm, err := hypervisor.NewVM()
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}
	defer vm.Close()

	// Create vCPU
	vcpu, err := vm.NewVCPU()
	if err != nil {
		return nil, fmt.Errorf("failed to create vCPU: %w", err)
	}
	defer vcpu.Close()

	// Allocate memory
	hostMem, err := unix.Mmap(-1, 0, memSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate memory: %w", err)
	}
	defer unix.Munmap(hostMem)

	// Copy code to memory
	if len(code) > len(hostMem) {
		return nil, fmt.Errorf("code size (%d) exceeds memory size (%d)", len(code), len(hostMem))
	}
	copy(hostMem, code)

	// Map memory into guest
	baseAddr := uint64(0x4000)
	perms := hypervisor.MemRead | hypervisor.MemWrite | hypervisor.MemExec
	err = vm.Map(hostMem, baseAddr, perms)
	if err != nil {
		return nil, fmt.Errorf("failed to map memory: %w", err)
	}
	defer vm.Unmap(baseAddr, uint64(len(hostMem)))

	// Set initial CPU state
	if err := vcpu.SetReg(hypervisor.RegSP, stackPtr); err != nil {
		return nil, fmt.Errorf("failed to set SP: %w", err)
	}
	if err := vcpu.SetPC(baseAddr); err != nil {
		return nil, fmt.Errorf("failed to set PC: %w", err)
	}

	// Execute
	exitInfo, err := vcpu.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to execute: %w", err)
	}

	// Get final CPU state
	finalState, err := getCPUState(vcpu)
	if err != nil {
		return nil, fmt.Errorf("failed to get final state: %w", err)
	}

	// Copy the memory for analysis
	memCopy := make([]byte, len(hostMem))
	copy(memCopy, hostMem)

	return &ExecuteResult{
		State:    *finalState,
		ExitInfo: exitInfo,
		Memory:   map[string][]byte{fmt.Sprintf("0x%x", baseAddr): memCopy},
	}, nil
}

// printStackContents displays the stack contents in a readable format
func printStackContents(memory map[string][]byte, baseAddr, initialSP, finalSP uint64) {
	fmt.Printf("\n=== Stack Analysis ===\n")

	// Get the memory region
	var memData []byte
	for _, data := range memory {
		memData = data
		break // Should only be one region
	}

	if memData == nil {
		fmt.Println("No memory data available")
		return
	}

	// Determine stack range to display
	stackStart := initialSP - baseAddr
	stackEnd := finalSP - baseAddr
	_ = stackEnd

	// Show some context around the stack safely (no underflow)
	displayStart := stackStart - min(stackStart, uint64(64))
	displayEnd := min(stackStart+64, uint64(len(memData)))

	fmt.Printf("Stack region: 0x%x - 0x%x (Initial SP: 0x%x, Final SP: 0x%x)\n",
		baseAddr+displayStart, baseAddr+displayEnd, initialSP, finalSP)
	fmt.Printf("Stack change: %d bytes\n\n", int64(finalSP)-int64(initialSP))

	// Show annotations for stack pointers
	fmt.Printf("Annotations: ISP=Initial SP, FSP=Final SP, STK=Stack Area\n")

	// Mark important addresses in the displayed region
	for offset := displayStart; offset < displayEnd; offset += 16 {
		addr := baseAddr + offset

		if addr == initialSP {
			fmt.Printf("ISP> ")
		} else if addr == finalSP {
			fmt.Printf("FSP> ")
		} else if addr >= finalSP && addr < initialSP && finalSP < initialSP {
			fmt.Printf("STK> ")
		} else {
			fmt.Printf("     ")
		}

		// Break early if we would exceed bounds
		endOffset := min(min(offset+16, uint64(len(memData))), displayEnd)

		if offset >= endOffset {
			break
		}

		fmt.Printf("%s", utils.HexDump(memData[int(offset):int(endOffset)], baseAddr+offset))
	}
}
